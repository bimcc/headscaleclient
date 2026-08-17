//go:build windows

package config

import (
	"github.com/headscaleclient/headscaleclient/internal/domain"
	"golang.org/x/sys/windows/registry"
)

const installedLanguageRegistryKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\BIMCC., Ltd.HeadscaleClient`

func readInstalledDefaultLanguage() domain.Language {
	key, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		installedLanguageRegistryKey,
		registry.QUERY_VALUE|registry.WOW64_64KEY,
	)
	if err != nil {
		return ""
	}
	defer key.Close()

	value, _, err := key.GetStringValue("DefaultLanguage")
	if err != nil {
		return ""
	}
	return domain.Language(value)
}
