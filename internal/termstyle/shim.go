// Package termstyle is termtint's thin adapter over the shared termtheme
// engine. The cross-compat-critical data layer — the semantic roles, the
// theme.conf parser, the style-spec interpreter, the SGR/grapheme render
// helpers, and the portable .theme format — is re-exported from termtheme so a
// theme written by any sibling app parses and renders identically here.
// termtint keeps its own builtin palettes and fail-open theme resolution
// local, since those legitimately differ per app.
package termstyle

import (
	"github.com/0xbenc/termchrome"
	"github.com/0xbenc/termtheme"
)

// Role is the shared semantic styling slot.
type Role = termtheme.Role

const (
	RoleTitle       = termtheme.RoleTitle
	RolePrimary     = termtheme.RolePrimary
	RoleSecondary   = termtheme.RoleSecondary
	RoleAccent      = termtheme.RoleAccent
	RoleMuted       = termtheme.RoleMuted
	RoleSubtle      = termtheme.RoleSubtle
	RoleForeground  = termtheme.RoleForeground
	RoleSelected    = termtheme.RoleSelected
	RoleSelectedBar = termtheme.RoleSelectedBar
	RoleBorder      = termtheme.RoleBorder
	RoleSuccess     = termtheme.RoleSuccess
	RoleWarning     = termtheme.RoleWarning
	RoleDanger      = termtheme.RoleDanger
	RoleInfo        = termtheme.RoleInfo
	RoleSearch      = termtheme.RoleSearch
	RolePill        = termtheme.RolePill
)

// ThemeConfig is the parsed theme file (base name + per-role overrides).
type ThemeConfig = termtheme.ThemeConfig

// ThemeMeta is the header/version info recovered from a portable .theme file.
type ThemeMeta = termtheme.Meta

// Roles is termtint's rendered role set. termtint paints every role — it is
// the family's studio, so it is the full universal superset.
func Roles() []Role { return termtheme.Roles() }

// Engine functions re-exported from termtheme. Sharing these is what makes a
// theme written by any sibling app parse and render identically here.
var (
	ParseThemeConfig  = termtheme.ParseThemeConfig
	ParseStyleSpec    = termtheme.ParseStyleSpec
	Apply             = termtheme.Apply
	VisibleWidth      = termtheme.VisibleWidth
	Strip             = termtheme.Strip
	Sanitize          = termtheme.Sanitize
	PadRight          = termtheme.PadRight
	Truncate          = termtheme.Truncate
	TruncateWith      = termtheme.TruncateWith
	Marshal           = termtheme.Marshal
	Unmarshal         = termtheme.Unmarshal
	FormatThemeConfig = termtheme.FormatThemeConfig
	ResolveThemeFile  = termtheme.ResolveThemeFile
	EnvNoColor        = termtheme.EnvNoColor
	EnvTruthy         = termtheme.EnvTruthy
	ExpandPath        = termtheme.ExpandPath
	EnvMap            = termtheme.EnvMap
)

// Glyphs and countdown re-exported from termchrome.
type GlyphSet = termchrome.GlyphSet

var (
	ResolveGlyphs = termchrome.ResolveGlyphs
	DefaultGlyphs = termchrome.DefaultGlyphs
	UrgencyRole   = termchrome.UrgencyRole
)

// Footer grammar re-exported from termchrome.
type KeyHint = termchrome.KeyHint

const FooterSep = termchrome.FooterSep

var Footer = termchrome.Footer
