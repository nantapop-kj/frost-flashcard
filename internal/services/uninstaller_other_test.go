//go:build !windows

package services

import "testing"

func TestRegisteredUninstallerIsWindowsOnly(t *testing.T) {
	if path, err := registeredUninstaller(); err == nil || path != "" {
		t.Errorf("registeredUninstaller = %q, %v; want an error", path, err)
	}
}
