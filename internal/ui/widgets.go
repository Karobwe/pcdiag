package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Karobwe/pcdiag/internal/collector"
	"github.com/Karobwe/pcdiag/internal/health"
)

// networkSVG is a Material "lan" glyph.
var networkSVG = fyne.NewStaticResource("network.svg", []byte(
	`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#000" d="M13 22h8v-7h-3v-4h-5V9h3V2H8v7h3v2H6v4H3v7h8v-7H8v-2h8v2h-3z"/></svg>`))

// networkIcon returns the network glyph recoloured by the theme.
func networkIcon() fyne.Resource { return theme.NewThemedResource(networkSVG) }

// Status icons are built lazily: theme lookups need a running Fyne app, which
// does not exist in report (command line) mode.
func iconOK() fyne.Resource      { return theme.NewSuccessThemedResource(theme.ConfirmIcon()) }
func iconWarning() fyne.Resource { return theme.NewWarningThemedResource(theme.WarningIcon()) }
func iconError() fyne.Resource   { return theme.NewErrorThemedResource(theme.ErrorIcon()) }
func iconInfo() fyne.Resource    { return theme.NewPrimaryThemedResource(theme.InfoIcon()) }
func iconSkipped() fyne.Resource { return theme.NewDisabledResource(theme.QuestionIcon()) }

func scroll(c fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVScroll(container.NewPadded(c))
}

// kv is a key/value row.
type kv struct{ key, value string }

// form lays out key/value pairs on two aligned columns. Empty values are
// skipped so that platform-specific fields do not leave blank rows.
func form(rows ...kv) *fyne.Container {
	var objs []fyne.CanvasObject
	for _, r := range rows {
		if r.value == "" {
			continue
		}
		key := widget.NewLabelWithStyle(r.key, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		val := widget.NewLabel(r.value)
		val.Selectable = true
		val.Wrapping = fyne.TextWrapWord
		objs = append(objs, key, val)
	}
	return container.New(layout.NewFormLayout(), objs...)
}

// usageBar is a progress bar showing a usage ratio with a custom caption.
func usageBar(percent float64, caption string) *widget.ProgressBar {
	b := widget.NewProgressBar()
	b.Max = 100
	b.SetValue(percent)
	b.TextFormatter = func() string { return caption }
	return b
}

func statusIcon(s collector.Status) fyne.Resource {
	switch s {
	case collector.StatusOK:
		return iconOK()
	case collector.StatusWarning:
		return iconWarning()
	case collector.StatusFailed:
		return iconError()
	default:
		return iconSkipped()
	}
}

func levelIcon(l health.Level) fyne.Resource {
	switch l {
	case health.Critical:
		return iconError()
	case health.Warning:
		return iconWarning()
	case health.Info:
		return iconInfo()
	default:
		return iconOK()
	}
}

// rowStyle tunes how a row is drawn.
type rowStyle struct {
	bold         bool // title in bold
	monoTrailing bool // trailing text in a monospace font (numbers)
}

// row is an icon, a bold title with a secondary line, and an optional
// trailing text.
func row(icon fyne.Resource, title, subtitle, trailing string) fyne.CanvasObject {
	return styledRow(icon, title, subtitle, trailing, rowStyle{bold: true})
}

// message is a row with a regular-weight text, used for diagnosis findings.
func message(icon fyne.Resource, text string) fyne.CanvasObject {
	return styledRow(icon, text, "", "", rowStyle{})
}

func styledRow(icon fyne.Resource, title, subtitle, trailing string, st rowStyle) fyne.CanvasObject {
	t := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: st.bold})
	t.Wrapping = fyne.TextWrapWord
	var body fyne.CanvasObject = t
	if subtitle != "" {
		s := widget.NewLabel(subtitle)
		s.Wrapping = fyne.TextWrapWord
		s.Selectable = true
		s.SizeName = theme.SizeNameCaptionText
		// Labels carry their own inner padding: remove the extra gap so
		// the subtitle reads as part of the title.
		body = container.New(layout.NewCustomPaddedVBoxLayout(-2*theme.Padding()), t, s)
	}
	var right fyne.CanvasObject
	if trailing != "" {
		right = widget.NewLabelWithStyle(trailing, fyne.TextAlignTrailing, fyne.TextStyle{Monospace: st.monoTrailing})
	}
	return container.NewBorder(nil, nil, container.NewCenter(widget.NewIcon(icon)), right, body)
}

// tile is a compact card used on the overview tab.
func tile(icon fyne.Resource, title string, content ...fyne.CanvasObject) fyne.CanvasObject {
	header := container.NewHBox(widget.NewIcon(icon),
		widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	body := container.NewVBox(append([]fyne.CanvasObject{header, widget.NewSeparator()}, content...)...)
	return widget.NewCard("", "", body)
}

func label(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextWrapWord
	l.Selectable = true
	return l
}

func heading(text string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}
