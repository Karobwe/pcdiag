// Package i18n registers the application translations with Fyne.
//
// Source strings are written in English and used directly as translation
// keys (gettext style), so a missing translation always falls back to
// readable English. To add a language, create translations/<lang>.json
// (for example de.json) with the same keys as fr.json. Note that go-i18n
// reserves a few keys (such as "Other", "One", "Description" or "ID") for
// plural forms and metadata: they cannot be used as source strings.
package i18n

import (
	"embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
)

//go:embed translations
var translations embed.FS

// Init loads the embedded translations. Fyne selects the language matching
// the operating system locale. Errors are logged and returned; the
// application keeps working in English.
func Init() error {
	err := lang.AddTranslationsFS(translations, "translations")
	if err != nil {
		fyne.LogError("Unable to load translations", err)
	}
	return err
}
