//go:build windows

package config

import (
	"github.com/headscaleclient/headscaleclient/internal/domain"
	"golang.org/x/sys/windows/registry"
)

const installedLanguageRegistryKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\io.headscaleclient.desktop`

func readInstalledDefaultLanguage() domain.Language {
	for _, path := range []string{installedLanguageRegistryKey, `Software\Microsoft\Windows\CurrentVersion\Uninstall\BIMCC., Ltd.HeadscaleClient`} {
		key, err := registry.OpenKey(
			registry.LOCAL_MACHINE,
			path,
			registry.QUERY_VALUE|registry.WOW64_64KEY,
		)
		if err != nil {
			continue
		}

		value, _, err := key.GetStringValue("DefaultLanguage")
		key.Close()
		if err != nil {
			continue
		}
		return domain.Language(value)
	}
	return ""
}
