package i18n

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Karobwe/pcdiag/internal/health"
)

// sourceKeys returns every string literal passed to lang.L in the module.
func sourceKeys(t *testing.T) map[string]bool {
	t.Helper()
	keys := map[string]bool{}
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "vendor") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "L" {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "lang" {
				return true
			}
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				s, _ := strconv.Unquote(lit.Value)
				keys[s] = true
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return keys
}

func TestTranslationsAreComplete(t *testing.T) {
	keys := sourceKeys(t)
	for _, m := range []string{
		health.MsgUptime, health.MsgMemory, health.MsgCPU, health.MsgDiskFull, health.MsgNoNetwork,
		health.MsgAPIPA, health.MsgNoGateway, health.MsgCheckFailed, health.MsgCheckWarning,
		health.MsgSpoolerStopped, health.MsgPrinterOffline, health.MsgNoDefault, health.MsgAllGood,
	} {
		keys[m] = true
	}
	if len(keys) < 50 {
		t.Fatalf("only %d keys found: source scanning is broken", len(keys))
	}

	entries, err := translations.ReadDir("translations")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := translations.ReadFile("translations/" + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]string
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		for k := range keys {
			if strings.TrimSpace(m[k]) == "" {
				t.Errorf("%s: missing translation for %q", e.Name(), k)
			}
		}
		for k, v := range m {
			if strings.Count(k, "{{") != strings.Count(v, "{{") {
				t.Errorf("%s: placeholders differ for %q", e.Name(), k)
			}
		}
	}
}

func TestInit(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("translations rejected by go-i18n: %v", err)
	}
}
