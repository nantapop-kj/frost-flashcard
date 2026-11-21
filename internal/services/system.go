package services

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/nantapop-kj/frost-flashcard/internal/dto"
)

type Platform interface {
	Quit()
	DisableAutostart() error
}

type AppInfo struct {
	Version    string `json:"version"`
	Platform   string `json:"platform"`
	CanInstall bool   `json:"canInstall"`
}

type System struct {
	cards       *Service
	version     string
	dataDirs    []string
	platform    Platform
	goos        string
	appsDir     string
	bundleID    string
	exe         func() (string, error)
	uninstaller func() (string, error)
	run         func(name string, args ...string) ([]byte, error)
	start       func(name string, args ...string) error
	quitDelay   time.Duration
}

func NewSystem(cards *Service, version string, p Platform, dataDirNames ...string) *System {
	bundleID := ""
	if len(dataDirNames) > 0 {
		bundleID = dataDirNames[0]
	}
	return &System{
		cards:       cards,
		version:     version,
		dataDirs:    dataDirNames,
		platform:    p,
		goos:        runtime.GOOS,
		appsDir:     "/Applications",
		bundleID:    bundleID,
		exe:         executable,
		uninstaller: registeredUninstaller,
		run:         func(name string, args ...string) ([]byte, error) { return exec.Command(name, args...).CombinedOutput() },
		start:       func(name string, args ...string) error { return exec.Command(name, args...).Start() },
		quitDelay:   150 * time.Millisecond,
	}
}

func (s *System) AppInfo() AppInfo {
	info := AppInfo{Version: s.version, Platform: s.goos}
	if bundle := s.bundle(); bundle != "" {
		info.CanInstall = !strings.HasPrefix(bundle, s.appsDir+"/")
	}
	return info
}

func (s *System) InstallApp() error {
	src := s.bundle()
	if s.goos != "darwin" || src == "" {
		return &dto.Error{
			Code:   dto.CodeBadInput,
			Detail: "not running from a macOS .app bundle",
		}
	}
	if strings.HasPrefix(src, s.appsDir+"/") {
		return &dto.Error{
			Code:   dto.CodeBadInput,
			Detail: "already installed in " + s.appsDir,
		}
	}
	dst := filepath.Join(s.appsDir, filepath.Base(src))
	if _, err := os.Lstat(dst); err == nil && !s.isOurBundle(dst) {
		return &dto.Error{
			Code:   dto.CodeBadInput,
			Detail: dst + " belongs to another app",
		}
	}
	if err := os.RemoveAll(dst); err != nil {
		return toError(err, dto.CodeWriteFailed)
	}
	if out, err := s.run("ditto", src, dst); err != nil {
		return &dto.Error{
			Code:   dto.CodeWriteFailed,
			Detail: strings.TrimSpace(string(out)),
		}
	}
	if err := s.start("open", "-n", dst); err != nil {
		return toError(err, dto.CodeWriteFailed)
	}
	s.Quit()

	return nil
}

func (s *System) Uninstall() error {
	dir := filepath.Dir(s.cards.path)
	if !slices.Contains(s.dataDirs, filepath.Base(dir)) {
		return &dto.Error{
			Code:   dto.CodeBadInput,
			Detail: "unexpected data folder: " + dir,
		}
	}

	s.cards.stopSaving()
	if err := s.platform.DisableAutostart(); err != nil {
		s.cards.resumeSaving()

		return toError(err, dto.CodeWriteFailed)
	}
	if err := os.RemoveAll(dir); err != nil {
		s.cards.resumeSaving()

		return toError(err, dto.CodeWriteFailed)
	}

	switch s.goos {
	case "darwin":
		if bundle := s.bundle(); bundle != "" && s.isOurBundle(bundle) {
			if err := os.RemoveAll(bundle); err != nil {
				_ = s.start("open", "-R", bundle)
			}
		}
	case "windows":
		if uninstaller := s.ownUninstaller(); uninstaller != "" {
			_ = s.start(uninstaller, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART")
		}
	}

	s.Quit()

	return nil
}

func (s *System) Quit() {
	time.AfterFunc(s.quitDelay, s.platform.Quit)
}

func (s *System) bundle() string {
	exe, err := s.exe()
	if err != nil {
		return ""
	}

	return bundleFromExecutable(exe)
}

var bundleIDPattern = regexp.MustCompile(`<key>CFBundleIdentifier</key>\s*<string>([^<]*)</string>`)

func (s *System) isOurBundle(path string) bool {
	if s.bundleID == "" {
		return false
	}

	if fi, err := os.Lstat(path); err != nil || !fi.IsDir() {
		return false
	}

	plist, err := os.ReadFile(filepath.Join(path, "Contents", "Info.plist"))
	if err != nil {
		return false
	}

	m := bundleIDPattern.FindSubmatch(plist)
	return m != nil && string(m[1]) == s.bundleID
}

var innoUninstallerName = regexp.MustCompile(`(?i)^unins\d{3}\.exe$`)

func (s *System) ownUninstaller() string {
	exe, err := s.exe()
	if err != nil {
		return ""
	}

	cmd, err := s.uninstaller()
	if err != nil {
		return ""
	}

	path := strings.TrimSpace(cmd)
	if len(path) >= 2 && path[0] == '"' {
		end := strings.IndexByte(path[1:], '"')
		if end < 0 {
			return ""
		}
		path = path[1 : end+1]
	}

	if !innoUninstallerName.MatchString(windowsBase(path)) ||
		!strings.EqualFold(windowsDir(path), windowsDir(exe)) {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}

	return path
}

func windowsDir(p string) string {
	i := strings.LastIndexAny(p, `\/`)
	if i < 0 {
		return ""
	}

	return strings.TrimRight(p[:i], `\/`)
}

func windowsBase(p string) string {
	return p[strings.LastIndexAny(p, `\/`)+1:]
}

var osExecutable = os.Executable

func executable() (string, error) {
	exe, err := osExecutable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}

	return exe, nil
}

func bundleFromExecutable(exe string) string {
	const marker = ".app/Contents/MacOS/"
	i := strings.LastIndex(exe, marker)
	if i < 0 {
		return ""
	}

	return exe[:i+len(".app")]
}

var errUninstalled = errors.New("the app was uninstalled")
