// Package ui implements the Fyne desktop interface of PC Diag.
package ui

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Karobwe/pcdiag/internal/collector"
	"github.com/Karobwe/pcdiag/internal/config"
	"github.com/Karobwe/pcdiag/internal/health"
	"github.com/Karobwe/pcdiag/internal/i18n"
	"github.com/Karobwe/pcdiag/internal/report"
)

// AppID is the unique application identifier used by Fyne.
const AppID = "io.github.karobwe.pcdiag"

// RepositoryURL is shown in the About dialog.
const RepositoryURL = "https://github.com/Karobwe/pcdiag"

// Params configures the user interface.
type Params struct {
	Version    string
	Config     config.Config
	ConfigPath string
	ConfigErr  error
}

// ui holds the window state. All fields are only accessed from the Fyne
// goroutine, except where noted.
type ui struct {
	app    fyne.App
	win    fyne.Window
	params Params

	snap     *collector.Snapshot
	findings []health.Finding

	tabs                                              *container.AppTabs
	overview, system, network, connectivity, printers *fyne.Container

	refreshBtn, checksBtn *widget.Button
	progress              *widget.ProgressBarInfinite
	status                *widget.Label
	showInactive          bool

	mu      sync.Mutex // guards running
	running bool
}

// Run creates the window and blocks until it is closed.
func Run(p Params) {
	u := newUI(app.NewWithID(AppID), p)
	if p.ConfigErr != nil {
		dialog.ShowError(p.ConfigErr, u.win)
	}
	u.refresh()
	u.win.ShowAndRun()
}

func newUI(a fyne.App, p Params) *ui {
	_ = i18n.Init() // errors are logged; the UI falls back to English
	if p.Version == "" || p.Version == "dev" {
		if v := a.Metadata().Version; v != "" && v != "0.0.1" {
			p.Version = v
		}
	}

	u := &ui{app: a, params: p}
	u.win = a.NewWindow(lang.L("PC Diag"))
	if icon := a.Metadata().Icon; icon != nil {
		u.win.SetIcon(icon)
	}
	u.win.SetContent(u.build())
	u.win.Resize(fyne.NewSize(1000, 720))
	u.win.CenterOnScreen()
	u.registerShortcuts()
	return u
}

func (u *ui) build() fyne.CanvasObject {
	u.overview = container.NewVBox()
	u.system = container.NewVBox()
	u.network = container.NewVBox()
	u.connectivity = container.NewVBox()
	u.printers = container.NewVBox()

	u.tabs = container.NewAppTabs(
		container.NewTabItemWithIcon(lang.L("Overview"), theme.HomeIcon(), scroll(u.overview)),
		container.NewTabItemWithIcon(lang.L("System"), theme.ComputerIcon(), scroll(u.system)),
		container.NewTabItemWithIcon(lang.L("Network"), networkIcon(), u.networkTab()),
		container.NewTabItemWithIcon(lang.L("Connectivity"), theme.UploadIcon(), u.connectivityTab()),
		container.NewTabItemWithIcon(lang.L("Printers"), theme.DocumentPrintIcon(), scroll(u.printers)),
	)

	u.refreshBtn = widget.NewButtonWithIcon(lang.L("Refresh"), theme.ViewRefreshIcon(), u.refresh)
	u.refreshBtn.Importance = widget.HighImportance
	copyBtn := widget.NewButtonWithIcon(lang.L("Copy report"), theme.ContentCopyIcon(), u.copyReport)
	saveBtn := widget.NewButtonWithIcon(lang.L("Save report…"), theme.DocumentSaveIcon(), u.saveReport)
	aboutBtn := widget.NewButtonWithIcon(lang.L("About"), theme.InfoIcon(), u.showAbout)

	u.progress = widget.NewProgressBarInfinite()
	u.progress.Hide()
	u.status = widget.NewLabel("")
	u.status.Truncation = fyne.TextTruncateEllipsis

	toolbar := container.NewBorder(nil, nil,
		container.NewHBox(u.refreshBtn, copyBtn, saveBtn),
		aboutBtn,
	)
	footer := container.NewBorder(nil, nil, nil, u.supportLabel(),
		container.NewStack(u.progress, u.status))

	return container.NewBorder(
		container.NewPadded(toolbar), container.NewPadded(footer), nil, nil, u.tabs)
}

func (u *ui) networkTab() fyne.CanvasObject {
	inactive := widget.NewCheck(lang.L("Show inactive and virtual adapters"), func(b bool) {
		u.showInactive = b
		u.renderNetwork()
	})
	return container.NewBorder(container.NewPadded(inactive), nil, nil, nil, scroll(u.network))
}

func (u *ui) connectivityTab() fyne.CanvasObject {
	u.checksBtn = widget.NewButtonWithIcon(lang.L("Run the checks again"), theme.MediaReplayIcon(), u.rerunChecks)
	return container.NewBorder(container.NewPadded(container.NewHBox(u.checksBtn)), nil, nil, nil, scroll(u.connectivity))
}

func (u *ui) supportLabel() fyne.CanvasObject {
	s := u.params.Config.Support
	if s.IsZero() {
		return widget.NewLabel("")
	}
	parts := []string{s.Name, s.Phone, s.Email}
	text := strings.Join(nonEmpty(parts), " · ")
	lbl := widget.NewLabelWithStyle(lang.L("Support")+": "+text, fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	lbl.Selectable = true
	return lbl
}

func (u *ui) registerShortcuts() {
	c := u.win.Canvas()
	c.SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyF5 {
			u.refresh()
		}
	})
	c.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierShortcutDefault},
		func(fyne.Shortcut) { u.refresh() })
	c.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShortcutDefault},
		func(fyne.Shortcut) { u.saveReport() })
}

// begin marks a background task as running; it returns false if one is
// already in progress.
func (u *ui) begin(msg string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.running {
		return false
	}
	u.running = true
	u.refreshBtn.Disable()
	u.checksBtn.Disable()
	u.status.SetText(msg)
	u.status.Hide()
	u.progress.Show()
	u.progress.Start()
	return true
}

func (u *ui) end() {
	u.mu.Lock()
	u.running = false
	u.mu.Unlock()
	u.progress.Stop()
	u.progress.Hide()
	u.status.Show()
	u.refreshBtn.Enable()
	u.checksBtn.Enable()
}

func (u *ui) refresh() {
	if !u.begin(lang.L("Collecting information…")) {
		return
	}
	opts := u.params.Config.Options()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		snap := collector.Collect(ctx, opts)
		fyne.Do(func() {
			u.setSnapshot(snap)
			u.end()
		})
	}()
}

func (u *ui) rerunChecks() {
	if u.snap == nil || !u.begin(lang.L("Running connectivity checks…")) {
		return
	}
	specs := collector.BuiltinChecks(u.snap.Network, u.params.Config.Options())
	go func() {
		results := collector.RunChecks(context.Background(), specs)
		fyne.Do(func() {
			u.snap.Connectivity = results
			u.setSnapshot(u.snap)
			u.end()
		})
	}()
}

func (u *ui) setSnapshot(s *collector.Snapshot) {
	u.snap = s
	th := health.DefaultThresholds()
	if v := u.params.Config.Thresholds.MemoryPercent; v > 0 {
		th.MemoryPercent = v
	}
	if v := u.params.Config.Thresholds.DiskPercent; v > 0 {
		th.DiskPercent = v
	}
	u.findings = health.Analyze(s, th)

	u.renderOverview()
	u.renderSystem()
	u.renderNetwork()
	u.renderConnectivity()
	u.renderPrinters()
	u.status.SetText(lang.L("Last updated at {{.Time}} ({{.Duration}} s)", map[string]any{
		"Time":     time.Now().Format("15:04:05"),
		"Duration": fmt.Sprintf("%.1f", s.Duration.Seconds()),
	}))
}

func (u *ui) textReport() string {
	var buf bytes.Buffer
	_ = report.Text(&buf, u.snap, u.findings, u.params.Version)
	return buf.String()
}

func (u *ui) copyReport() {
	if u.snap == nil {
		return
	}
	u.app.Clipboard().SetContent(u.textReport())
	u.status.SetText(lang.L("Report copied to the clipboard."))
}

func (u *ui) saveReport() {
	if u.snap == nil {
		return
	}
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, u.win)
			return
		}
		if w == nil {
			return // canceled by the user
		}
		defer w.Close()
		if strings.EqualFold(filepath.Ext(w.URI().Path()), ".json") {
			err = report.JSON(w, u.snap)
		} else {
			_, err = w.Write([]byte(u.textReport()))
		}
		if err != nil {
			dialog.ShowError(err, u.win)
			return
		}
		u.status.SetText(lang.L("Report saved to {{.Path}}", map[string]any{"Path": w.URI().Path()}))
	}, u.win)
	d.SetFileName(fmt.Sprintf("pcdiag-%s-%s.txt", sanitize(u.snap.System.Hostname), u.snap.CollectedAt.Format("20060102-1504")))
	d.SetFilter(storage.NewExtensionFileFilter([]string{".txt", ".json"}))
	d.Show()
}

func (u *ui) showAbout() {
	cfg := u.params.ConfigPath
	if cfg == "" {
		cfg = lang.L("none (built-in defaults)")
	}
	md := fmt.Sprintf("## %s\n\n%s\n\n- **%s** %s\n- **%s** %s\n- **%s** [%s](%s)\n",
		lang.L("PC Diag"),
		lang.L("Collects the information a helpdesk needs to troubleshoot a workstation."),
		lang.L("Version:"), u.params.Version,
		lang.L("Configuration:"), cfg,
		lang.L("Source code:"), RepositoryURL, RepositoryURL,
	)
	if s := u.params.Config.Support; !s.IsZero() {
		md += fmt.Sprintf("\n### %s\n\n%s\n", lang.L("Support"),
			strings.Join(nonEmpty([]string{s.Name, s.Phone, s.Email, s.URL}), "  \n"))
	}
	rt := widget.NewRichTextFromMarkdown(md)
	rt.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom(lang.L("About"), lang.L("Close"), rt, u.win)
	d.Resize(fyne.NewSize(520, 360))
	d.Show()
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, s)
}

func nonEmpty(in []string) []string {
	var out []string
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}
