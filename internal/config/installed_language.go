package config

import "github.com/headscaleclient/headscaleclient/internal/domain"

// InstalledDefaultLanguage returns the installer's language choice when the
// platform exposes one. Chinese remains the fallback for portable builds.
func InstalledDefaultLanguage() domain.Language {
	language := readInstalledDefaultLanguage()
	if !language.Valid() {
		return domain.LanguageChinese
	}
	return language
}
