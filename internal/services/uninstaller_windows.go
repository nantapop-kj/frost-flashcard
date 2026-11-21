//go:build windows

package services

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

const innoUninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\{44A7C97A-90AA-4A87-A4BC-F459B446E477}_is1`

func registeredUninstaller() (string, error) {
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		k, err := registry.OpenKey(root, innoUninstallKey, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		v, _, err := k.GetStringValue("UninstallString")
		k.Close()
		if err == nil && v != "" {
			return v, nil
		}
	}

	return "", errors.New("uninstaller is not registered")
}
