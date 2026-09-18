package termstyle

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/0xbenc/termtheme"
)

// Theme is termtint's concrete palette: a role->SGR-code map plus a NoColor
// switch. It shares an identical layout with termtheme.Theme (the conversion
// is free) but stays local to termtint along with its builtin palettes.
type Theme struct {
	Name    string
	NoColor bool
	Codes   map[Role]string
}

// TerminalTheme is the family canonical terminal palette — the same values the
// sibling apps use for their "terminal" base, so a theme resolved against it
// here reads the same in every app.
func TerminalTheme() Theme {
	return Theme{
		Name: "terminal",
		Codes: map[Role]string{
			RoleTitle:       "1;36",
			RolePrimary:     "36",
			RoleSecondary:   "34",
			RoleAccent:      "33",
			RoleMuted:       "90",
			RoleSubtle:      "2",
			RoleForeground:  "39",
			RoleSelected:    "39;4",
			RoleSelectedBar: "100",
			RoleBorder:      "1;32",
			RoleSuccess:     "32",
			RoleWarning:     "33",
			RoleDanger:      "31",
			RoleInfo:        "35",
			RoleSearch:      "1;39",
			RolePill:        "1;7",
		},
	}
}

// TintTheme is termtint's brand palette: a warm rose/amber truecolor scheme,
// deliberately distinct from passage's cyan/green brand.
func TintTheme() Theme {
	return Theme{
		Name: "tint",
		Codes: map[Role]string{
			RoleTitle:       "1;38;2;255;183;197",
			RolePrimary:     "1;38;2;255;183;197",
			RoleSecondary:   "1;38;2;255;214;153",
			RoleAccent:      "1;38;2;255;224;138",
			RoleMuted:       "38;2;150;140;148",
			RoleSubtle:      "38;2;74;68;74",
			RoleForeground:  "38;2;240;236;238",
			RoleSelected:    "1;38;2;255;255;255",
			RoleSelectedBar: "48;2;64;44;52",
			RoleBorder:      "38;2;110;84;96",
			RoleSuccess:     "1;38;2;148;226;160",
			RoleWarning:     "1;38;2;255;214;120",
			RoleDanger:      "1;38;2;255;138;138",
			RoleInfo:        "1;38;2;168;196;255",
			RoleSearch:      "1;38;2;255;255;255",
			RolePill:        "1;38;2;255;255;255;48;2;255;183;197",
		},
	}
}

// ChromeTheme is termtint's own UI palette. The studio's chrome — frame,
// section labels, role descriptions, hints, footer — always renders in this
// skin, never in the theme being edited, so the interface stays readable even
// while you author a dim one. It is built on the terminal defaults (the
// user's own, by definition readable scheme) with the low-contrast roles
// lifted: no SGR-dim text, and a bold-reverse current row instead of a
// faint underline.
func ChromeTheme() Theme {
	base := TerminalTheme()
	base.Name = "chrome"
	base.Codes[RoleSubtle] = "38;5;245"
	base.Codes[RoleSelected] = "1;7"
	base.Codes[RoleSelectedBar] = "48;5;236"
	return base
}

// Chrome contrast modes. "white" and "black" are monochrome high-contrast
// skins for the chrome — a safety switch for terminals where even the default
// chrome (or a half-edited theme) is hard to read — and starting points for
// the theme being edited: the preview wears them as its base, and saving
// while in the mode bakes them into the theme.
const (
	ChromeContrastDefault = "default"
	ChromeContrastWhite   = "white"
	ChromeContrastBlack   = "black"
)

// ChromeThemeVariant returns the chrome palette for a contrast mode. The
// monochrome modes paint every role in one high-contrast color — bold white
// or black — so the interface is legible on any terminal; the selected row
// inverts so the cursor position stays findable.
func ChromeThemeVariant(contrast string) Theme {
	switch contrast {
	case ChromeContrastWhite:
		return monoChrome("white", "1;37", "1;40", "40")
	case ChromeContrastBlack:
		return monoChrome("black", "30", "1;47", "47")
	default:
		return ChromeTheme()
	}
}

func monoChrome(name, fg, selected, bar string) Theme {
	t := Theme{Name: "chrome-" + name, Codes: make(map[Role]string, len(Roles()))}
	for _, role := range Roles() {
		t.Codes[role] = fg
	}
	t.Codes[RoleSelected] = selected
	t.Codes[RoleSelectedBar] = bar
	return t
}

// BuiltinThemeNames lists the selectable base palettes in display order. It is
// the single source of truth the theme editor cycles through.
func BuiltinThemeNames() []string {
	return []string{"terminal", "tint"}
}

// BuiltinTheme resolves a base palette name; unknown names report ok=false so
// callers can fall back.
func BuiltinTheme(name string) (Theme, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "terminal", "default", "auto":
		return TerminalTheme(), true
	case "tint", "brand":
		return TintTheme(), true
	default:
		return Theme{}, false
	}
}

func (t Theme) IsZero() bool {
	return t.Name == "" && len(t.Codes) == 0 && !t.NoColor
}

// Normalized fills any missing roles from the builtin base with the same name
// (falling back to the terminal palette), so a Theme's Codes are always
// complete enough to hand to Style.
func (t Theme) Normalized() Theme {
	base, _ := BuiltinTheme(t.Name)
	if base.IsZero() {
		base = TerminalTheme()
	}
	base.NoColor = t.NoColor
	if t.Name != "" {
		base.Name = t.Name
	}
	if base.Codes == nil {
		base.Codes = make(map[Role]string)
	} else {
		copied := make(map[Role]string, len(base.Codes))
		for role, code := range base.Codes {
			copied[role] = code
		}
		base.Codes = copied
	}
	for role, code := range t.Codes {
		base.Codes[role] = code
	}
	return base
}

// Style wraps value in the SGR code for role, normalizing first so a partial
// theme still renders every role.
func (t Theme) Style(role Role, value string) string {
	code, ok := t.Codes[role]
	if !ok {
		code = t.Normalized().Codes[role]
	}
	return Apply(t.NoColor, code, value)
}

func (t Theme) WithNoColor(noColor bool) Theme {
	t = t.Normalized()
	t.NoColor = noColor
	return t
}

// ToShared converts to the termtheme.Theme the portable format works with.
func (t Theme) ToShared() termtheme.Theme {
	return termtheme.Theme(t)
}

// FromShared converts a termtheme.Theme back to termtint's local type.
func FromShared(t termtheme.Theme) Theme {
	return Theme(t)
}

// ResolveOptions selects and loads termtint's own theme config.
type ResolveOptions struct {
	File            string
	NoColor         bool
	Env             []string
	SkipDefaultFile bool
}

// ResolveTheme loads termtint's theme.conf fail-open: a missing default file
// is not an error, an explicit missing file is. A config base termtint does not
// know (e.g. "vivid", which belongs to passage) falls back to the terminal
// palette — the role overrides still apply, and the base name is preserved on
// re-save.
func ResolveTheme(opts ResolveOptions) (Theme, error) {
	file, explicitFile := termtheme.ResolveThemeFile("termtint", opts.File, opts.Env, opts.SkipDefaultFile)

	var cfg ThemeConfig
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			if explicitFile || !errors.Is(err, fs.ErrNotExist) {
				return Theme{}, fmt.Errorf("load theme config %s: %w", file, err)
			}
		} else {
			parsed, err := ParseThemeConfig(data)
			if err != nil {
				return Theme{}, fmt.Errorf("load theme config %s: %w", file, err)
			}
			cfg = parsed
		}
	}

	base := TerminalTheme()
	if cfg.BaseName != "" {
		if b, ok := BuiltinTheme(cfg.BaseName); ok {
			base = b
		}
	}
	theme := cfg.Resolve(base.ToShared())
	if cfg.BaseName != "" {
		theme.Name = cfg.BaseName
	}
	if termtheme.EnvNoColor("termtint", opts.Env, opts.NoColor) {
		theme.NoColor = true
	}
	return FromShared(theme), nil
}
