//go:build !windows

package services

import "errors"

func registeredUninstaller() (string, error) {
	return "", errors.New("uninstaller is only registered on Windows")
}
