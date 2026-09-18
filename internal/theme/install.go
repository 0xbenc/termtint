package theme

import (
	"fmt"

	"github.com/0xbenc/termtheme"
)

// InstallOptions selects the target and payload for an install.
type InstallOptions struct {
	App       string
	ThemeFile string // explicit target config path override
	Config    termtheme.ThemeConfig
	Base      termtheme.Theme
	Version   string
	Env       []string
}

// InstallResult reports the install outcome.
type InstallResult struct {
	Path       string
	Changed    bool
	BackupPath string
}

// Install writes the theme to the target app's live theme.conf as a portable
// .theme dump: a versioned header plus every role inline, so the file is
// byte-droppable into the app's config and needs no base palette of its own.
// The write is atomic and takes a backup of the previous file.
func Install(opts InstallOptions) (InstallResult, error) {
	app := opts.App
	if app == "" {
		return InstallResult{}, fmt.Errorf("install requires a target app")
	}
	data := termtheme.Marshal(opts.Config, opts.Base, termtheme.MarshalOptions{
		App:        "termtint",
		AppVersion: opts.Version,
		Roles:      termtheme.Roles(),
	})
	target, _ := termtheme.ResolveThemeFile(app, opts.ThemeFile, opts.Env, false)
	if target == "" {
		return InstallResult{}, fmt.Errorf("no theme config path for %s", app)
	}
	res, err := AtomicWrite(target, data, WriteOptions{Backup: true, BackupPrefix: "termtint"})
	if err != nil {
		return InstallResult{}, err
	}
	return InstallResult{Path: res.Path, Changed: res.Changed, BackupPath: res.BackupPath}, nil
}
