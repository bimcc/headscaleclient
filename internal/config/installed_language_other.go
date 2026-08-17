//go:build !windows

package config

import "github.com/headscaleclient/headscaleclient/internal/domain"

func readInstalledDefaultLanguage() domain.Language {
	return ""
}
