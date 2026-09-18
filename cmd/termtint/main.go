// Command termtint is the theme studio for the termsystem family: author a
// theme against the full chrome vocabulary, preview it live, and move it
// losslessly between the family apps.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/0xbenc/termintro"
	"github.com/charmbracelet/x/term"

	"github.com/0xbenc/termtheme"
	"github.com/0xbenc/termtint/internal/state"
	"github.com/0xbenc/termtint/internal/termstyle"
	"github.com/0xbenc/termtint/internal/theme"
	"github.com/0xbenc/termtint/internal/ui"
)

// Version is set at release time via -ldflags.
var Version = "0.1.0-dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
}

type globalFlags struct {
	noColor  bool
	intro    bool
	noIntro  bool
	json     bool
	version  bool
	dryRun   bool
	themeArg string
	outArg   string
	nameArg  string
	fileArg  string
	entryArg string
	width    int
	height   int
}

const usage = `termtint — the theme studio for the termsystem family

Usage:
  termtint                          interactive studio (TTY)
  termtint <entry>                  open one theme in the studio (TTY)
  termtint list                     the theme set
  termtint show [entry]             role table for a theme
  termtint export [entry] [--out f] portable .theme on stdout
  termtint import <file> [--name n] validate and report (optionally save)
  termtint install <app> [--theme e] [--theme-file f] [--dry-run]
  termtint snapshot [--entry e] [--width n] [--height n]
  termtint version

Flags:
  --no-color      disable color
  --intro         force the startup intro
  --no-intro      suppress the startup intro
  --json          machine-readable output (list/show/import)
  --dry-run       (install) report without writing
  --theme-file    (install) explicit target config path
  --out FILE      (export) write to FILE instead of stdout
  --name NAME     (import) save as a named theme
  --entry NAME    (snapshot) the theme to render
Exit codes: 0 success, 1 failure, 2 usage.
`

func run(args []string, out, errw io.Writer, env []string) int {
	fl := &globalFlags{}
	fs := flag.NewFlagSet("termtint", flag.ContinueOnError)
	fs.SetOutput(errw)
	fs.BoolVar(&fl.noColor, "no-color", false, "disable color")
	fs.BoolVar(&fl.intro, "intro", false, "force the startup intro")
	fs.BoolVar(&fl.noIntro, "no-intro", false, "suppress the startup intro")
	fs.BoolVar(&fl.json, "json", false, "machine-readable output")
	fs.BoolVar(&fl.version, "version", false, "print the version")
	fs.BoolVar(&fl.dryRun, "dry-run", false, "install: report without writing")
	fs.StringVar(&fl.themeArg, "theme", "", "install: source theme entry")
	fs.StringVar(&fl.outArg, "out", "", "export: output file")
	fs.StringVar(&fl.nameArg, "name", "", "import: save as a named theme")
	fs.StringVar(&fl.fileArg, "theme-file", "", "install: target config path")
	fs.StringVar(&fl.entryArg, "entry", "", "snapshot: theme entry to render")
	fs.IntVar(&fl.width, "width", 100, "snapshot: terminal width")
	fs.IntVar(&fl.height, "height", 30, "snapshot: terminal height")
	fs.Usage = func() { fmt.Fprint(errw, usage) }
	flagArgs, posArgs := separateArgs(args)
	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}
	args = posArgs
	if fl.version {
		fmt.Fprintln(out, "termtint "+Version)
		return 0
	}

	store, err := theme.NewStore(env)
	if err != nil {
		fmt.Fprintf(errw, "termtint: %v\n", err)
		return 1
	}

	verb := ""
	verbArgs := args
	if len(verbArgs) > 0 {
		verb = verbArgs[0]
		verbArgs = verbArgs[1:]
	}

	switch verb {
	case "":
		if !isTTY(errw) {
			fmt.Fprint(errw, usage)
			return 2
		}
		return runStudio("", fl, out, errw, env, store)
	case "list":
		return runList(fl, out, errw, store)
	case "show":
		return runShow(verbArgs, fl, out, errw, store, env)
	case "export":
		return runExport(verbArgs, fl, out, errw, store)
	case "import":
		return runImport(verbArgs, fl, out, errw, store)
	case "install":
		return runInstall(verbArgs, fl, out, errw, store, env)
	case "snapshot":
		return runSnapshot(verbArgs, fl, out, errw, store, env)
	case "version":
		fmt.Fprintln(out, "termtint "+Version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(out, usage)
		return 0
	default:
		// Not a verb: it may be an entry name (termtint passage).
		if _, err := store.Load(verb); err == nil || theme.IsLiveApp(verb) {
			return runStudio(verb, fl, out, errw, env, store)
		}
		fmt.Fprintf(errw, "termtint: unknown verb %q\n\n%s", verb, usage)
		return 2
	}
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(f.Fd())
}

// runStudio launches the interactive program.
func runStudio(entryName string, fl *globalFlags, out, errw io.Writer, env []string, store theme.Store) int {
	if !isTTY(errw) {
		fmt.Fprintln(errw, "termtint: the studio needs a TTY; try: termtint list | show | export | import | install | snapshot")
		return 2
	}

	model := ui.NewModel(store, Version, env)
	model = model.WithNoColor(fl.noColor || termtheme.EnvNoColor("termtint", env, false))
	if entryName != "" {
		entry, err := store.Load(entryName)
		if err != nil {
			if entry.Kind != theme.KindLive {
				fmt.Fprintf(errw, "termtint: %v\n", err)
				return 1
			}
			// A family app with no theme.conf yet: open an empty studio.
		}
		model = model.Open(entry)
		model = model.MaybeHint(env)
	}

	if shouldPlayIntro(fl, env) {
		termintro.Play(termintro.Options{
			Title:   "TERTINT",
			Tagline: "the family theme studio",
			Credits: []string{"0xbenc"},
			Version: Version,
			Output:  errw,
			NoColor: fl.noColor || termtheme.EnvNoColor("termtint", env, false),
		})
		recordIntroVersion(env)
	}

	program := tea.NewProgram(model,
		tea.WithContext(context.Background()),
		tea.WithInput(os.Stdin),
		tea.WithOutput(out),
		tea.WithFilter(dropBadWindowSize),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(errw, "termtint: %v\n", err)
		return 1
	}
	return 0
}

// dropBadWindowSize discards degenerate size reports (e.g. a 0x0 reply from a
// pty that lost its winsize). Acting on one leaves the renderer with an
// empty frame forever, so the size is left at the last good value.
func dropBadWindowSize(_ tea.Model, msg tea.Msg) tea.Msg {
	if ws, ok := msg.(tea.WindowSizeMsg); ok && (ws.Width <= 0 || ws.Height <= 0) {
		return nil
	}
	return msg
}

// shouldPlayIntro: explicit suppression wins (--no-intro / TERTINT_NO_INTRO),
// then explicit force (--intro / TERTINT_INTRO_ALWAYS), then once-per-version.
func shouldPlayIntro(fl *globalFlags, env []string) bool {
	values := termstyle.EnvMap(env)
	if fl.noIntro || termstyle.EnvTruthy(values["TERTINT_NO_INTRO"]) {
		return false
	}
	if fl.intro || termstyle.EnvTruthy(values["TERTINT_INTRO_ALWAYS"]) {
		return true
	}
	dir, err := state.ResolveDir(env)
	if err != nil {
		return false
	}
	return state.Load(dir).LastIntroVersion != Version
}

func recordIntroVersion(env []string) {
	dir, err := state.ResolveDir(env)
	if err != nil {
		return
	}
	st := state.Load(dir)
	st.SetLastIntroVersion(Version)
	_ = st.Save(dir)
}

// activeThemeEntry resolves the entry a verb operates on by default: termtint's
// own live theme when no name is given.
func activeThemeEntry(store theme.Store, name string) (theme.Entry, error) {
	if name != "" {
		return store.Load(name)
	}
	path, _ := termtheme.ResolveThemeFile("termtint", "", store.Env, false)
	entry := theme.Entry{Name: "termtint (live)", Kind: theme.KindLive, App: "termtint", Path: path}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return entry, fmt.Errorf("read %s: %w", path, err)
		}
		cfg, meta, err := termtheme.Unmarshal(data)
		if err != nil {
			return entry, err
		}
		entry.Exists = true
		entry.Config = cfg
		entry.Meta = meta
		entry.BaseName = cfg.BaseName
	}
	return entry, nil
}

func runList(fl *globalFlags, out, errw io.Writer, store theme.Store) int {
	entries, err := store.Entries()
	if err != nil {
		fmt.Fprintf(errw, "termtint: %v\n", err)
		return 1
	}
	if fl.json {
		payload := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			item := map[string]any{
				"name":   entry.Name,
				"kind":   string(entry.Kind),
				"exists": entry.Exists,
			}
			if entry.Path != "" {
				item["path"] = entry.Path
			}
			if entry.BaseName != "" {
				item["base"] = entry.BaseName
			}
			if len(entry.Config.Specs) > 0 {
				item["roles"] = len(entry.Config.Specs)
			}
			if len(entry.Warnings) > 0 {
				item["warnings"] = entry.Warnings
			}
			payload = append(payload, item)
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Fprintln(out, string(data))
		return 0
	}
	fmt.Fprintf(out, "%-18s %-8s %s\n", "THEME", "KIND", "PATH")
	for _, entry := range entries {
		path := entry.Path
		if entry.Kind == theme.KindBuiltin {
			path = "(builtin palette)"
		} else if entry.Kind == theme.KindLive && !entry.Exists {
			path += "  (no config yet)"
		}
		fmt.Fprintf(out, "%-18s %-8s %s\n", entry.Name, entry.Kind, path)
	}
	return 0
}

func runShow(args []string, fl *globalFlags, out, errw io.Writer, store theme.Store, env []string) int {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	entry, err := activeThemeEntry(store, name)
	if err != nil {
		fmt.Fprintf(errw, "termtint: %v\n", err)
		return 1
	}
	cfg := entry.Config
	base := termstyle.TerminalTheme()
	baseLabel := orDash(entry.BaseName)
	if entry.BaseName != "" {
		if b, ok := termstyle.BuiltinTheme(entry.BaseName); ok {
			base = b
		} else {
			baseLabel += " (not a termtint palette; shown on the terminal base)"
		}
	}
	theme := cfg.Resolve(base.ToShared())
	if fl.json {
		payload := map[string]any{
			"entry": entry.Name,
			"base":  entry.BaseName,
			"roles": map[string]string{},
		}
		roles := payload["roles"].(map[string]string)
		for _, role := range termtheme.Roles() {
			roles[string(role)] = theme.Codes[role]
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Fprintln(out, string(data))
		return 0
	}
	fmt.Fprintf(out, "%s\nbase %s\n\n", entry.Name, baseLabel)
	fmt.Fprintf(out, "%-14s %-24s %s\n", "ROLE", "SPEC", "SGR")
	for _, role := range termtheme.Roles() {
		spec := strings.TrimSpace(cfg.Specs[role])
		if spec == "" {
			spec = "(inherit)"
		}
		fmt.Fprintf(out, "%-14s %-24s %s\n", role, spec, theme.Codes[role])
	}
	return 0
}

func runExport(args []string, fl *globalFlags, out, errw io.Writer, store theme.Store) int {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	entry, err := activeThemeEntry(store, name)
	if err != nil {
		fmt.Fprintf(errw, "termtint: %v\n", err)
		return 1
	}
	base := baseFor(entry)
	data := termtheme.Marshal(entry.Config, base.ToShared(), termtheme.MarshalOptions{
		App:        "termtint",
		AppVersion: Version,
		Roles:      termtheme.Roles(),
	})
	if fl.outArg != "" {
		path := termstyle.ExpandPath(fl.outArg)
		if err := os.MkdirAll(dirOf(path), 0o700); err != nil {
			fmt.Fprintf(errw, "termtint: %v\n", err)
			return 1
		}
		if _, err := theme.AtomicWrite(path, data, theme.WriteOptions{}); err != nil {
			fmt.Fprintf(errw, "termtint: %v\n", err)
			return 1
		}
		fmt.Fprintf(out, "exported %s\n", path)
		return 0
	}
	out.Write(data)
	return 0
}

func runImport(args []string, fl *globalFlags, out, errw io.Writer, store theme.Store) int {
	if len(args) != 1 {
		fmt.Fprintln(errw, "termtint: import requires a file")
		return 2
	}
	path := termstyle.ExpandPath(args[0])
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(errw, "termtint: %v\n", err)
		return 1
	}
	cfg, meta, err := termtheme.Unmarshal(data)
	if err != nil {
		fmt.Fprintf(errw, "termtint: %s: %v\n", path, err)
		return 1
	}
	if fl.json {
		payload := map[string]any{
			"file":     path,
			"source":   meta.App,
			"version":  meta.AppVersion,
			"format":   meta.Format,
			"roles":    len(cfg.Specs),
			"warnings": meta.Warnings,
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Fprintln(out, string(data))
	} else {
		fmt.Fprintf(out, "file    %s\n", path)
		if meta.App != "" {
			fmt.Fprintf(out, "source  %s %s\n", meta.App, orDash(meta.AppVersion))
		}
		fmt.Fprintf(out, "format  %d\nroles   %d\n", meta.Format, len(cfg.Specs))
		for _, warning := range meta.Warnings {
			fmt.Fprintf(out, "warning %s\n", warning)
		}
	}
	if fl.nameArg != "" {
		path, res, err := store.SaveNamed(fl.nameArg, cfg, baseFor(entryFromConfig(cfg)).ToShared(), Version)
		if err != nil {
			fmt.Fprintf(errw, "termtint: %v\n", err)
			return 1
		} else if res.Changed {
			fmt.Fprintf(out, "saved   %s\n", path)
		}
	}
	return 0
}

func runInstall(args []string, fl *globalFlags, out, errw io.Writer, store theme.Store, env []string) int {
	if len(args) != 1 {
		fmt.Fprintln(errw, "termtint: install requires a target app")
		return 2
	}
	app := args[0]
	known := false
	for _, candidate := range theme.LiveApps {
		if candidate == app {
			known = true
		}
	}
	if !known && app != "termtint" {
		fmt.Fprintf(errw, "termtint: unknown target %q (family apps: %s)\n", app, strings.Join(theme.LiveApps, ", "))
		return 2
	}
	entry, err := activeThemeEntry(store, fl.themeArg)
	if err != nil {
		fmt.Fprintf(errw, "termtint: %v\n", err)
		return 1
	}
	opts := theme.InstallOptions{
		App:       app,
		ThemeFile: fl.fileArg,
		Config:    entry.Config,
		Base:      baseFor(entry).ToShared(),
		Version:   Version,
		Env:       env,
	}
	if fl.dryRun {
		target, _ := termtheme.ResolveThemeFile(app, fl.fileArg, env, false)
		fmt.Fprintf(out, "would install %s -> %s\n", entry.Name, target)
		return 0
	}
	res, err := theme.Install(opts)
	if err != nil {
		fmt.Fprintf(errw, "termtint: %v\n", err)
		return 1
	}
	if !res.Changed {
		fmt.Fprintf(out, "unchanged %s\n", res.Path)
		return 0
	}
	fmt.Fprintf(out, "installed %s -> %s%s\n", entry.Name, res.Path, backupNote(res.BackupPath))
	return 0
}

func runSnapshot(args []string, fl *globalFlags, out, errw io.Writer, store theme.Store, env []string) int {
	model := ui.NewModel(store, Version, env)
	model = model.WithNoColor(fl.noColor || termtheme.EnvNoColor("termtint", env, false))
	if fl.entryArg != "" {
		entry, err := store.Load(fl.entryArg)
		if err != nil {
			fmt.Fprintf(errw, "termtint: %v\n", err)
			return 1
		}
		model = model.Open(entry)
	}
	// Seed the size, then render.
	model = model.WithSize(fl.width, fl.height)
	fmt.Fprint(out, model.View().Content)
	return 0
}

// isValueFlag reports whether a (dashed) flag name takes a value.
func isValueFlag(name string) bool {
	base := strings.TrimLeft(name, "-")
	switch base {
	case "theme", "theme-file", "out", "name", "entry", "width", "height":
		return true
	}
	return false
}

// separateArgs lifts value flags out of an argument list wherever they appear,
// so `install ssherpa --theme passage` and `install --theme passage ssherpa`
// both parse (the stdlib flag package stops at the first positional).
func separateArgs(args []string) (flags, positionals []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		name, inline, hasInline := arg, "", false
		if idx := strings.IndexByte(arg, '='); idx > 0 {
			name, inline, hasInline = arg[:idx], arg[idx+1:], true
		}
		if name == "--help" || name == "-h" {
			positionals = append(positionals, "help")
			continue
		}
		if !isValueFlag(name) {
			// Boolean flags (--no-color, --intro, ...) pass through as-is;
			// the flag package parses them.
			flags = append(flags, arg)
			continue
		}
		if hasInline {
			flags = append(flags, name+"="+inline)
			continue
		}
		if i+1 < len(args) {
			flags = append(flags, name, args[i+1])
			i++
			continue
		}
		flags = append(flags, name)
	}
	return flags, positionals
}

func baseFor(entry theme.Entry) termstyle.Theme {
	base := termstyle.TerminalTheme()
	if entry.BaseName != "" {
		if b, ok := termstyle.BuiltinTheme(entry.BaseName); ok {
			base = b
		}
	}
	return base
}

func entryFromConfig(cfg termtheme.ThemeConfig) theme.Entry {
	return theme.Entry{BaseName: cfg.BaseName, Config: cfg}
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func dirOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i > 0 {
		return path[:i]
	}
	return "."
}

func backupNote(backup string) string {
	if backup == "" {
		return ""
	}
	return " (backup " + backup + ")"
}
