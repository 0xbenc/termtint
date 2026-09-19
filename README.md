# termtint

The theme studio for the [termsystem](https://github.com/0xbenc/termsystem) family. Point it at an app, tweak colors, and watch the whole thing repaint live as you go. When you're happy, save — and the same theme works in every app in the family: passage, ssherpa, dangit, bitty.

One `.theme` file. Every app in the family.

```
╭ demo ────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ >> base          terminal           starting palette (←~ | PREVIEW                                         │
│ TEXT                                                     | ╭ vocabulary ─────────────────────────────────╮ │
│    title         bold bright-cyan   window title and ba~ | │ TERTINT LIVE                                │ │
│    primary       37                 primary content and~ | │ 16 roles  ·  base terminal                  │ │
│    secondary     (inherit)          secondary text, cou~ | │                                             │ │
│    accent        (inherit)          highlights, the >> ~ | │ >>  selected row                            │ │
│    muted         (inherit)          metadata, timestamps | │   primary row   12 / 92                     │ │
│    subtle        (inherit)          faint rules and hin~ | │   archive | backup | token                  │ │
│    foreground    (inherit)          default text         | │                                             │ │
│ SELECTION                                                | │ label   value                               │ │
│    selected      (inherit)          selected row text    | │ success line                                │ │
│    selected_bar  48;2;45;55;72      selected row backgr~ | │ warning line                                │ │
│ CHROME                                                   | │ danger line                                 │ │
│    border        (inherit)          box borders and rul~ | │ info line                                   │ │
│ STATE                                                    | │                                             │ │
│    success       (inherit)          success and OK lines | │ ⠸ ▰▰▱▱▱▱▱▱▱▱ 2s                             │ │
│    warning       (inherit)          warnings             | │                                             │ │
│    danger        (inherit)          errors and destruct~ | │ a alpha / b bravo / c charlie / +3          │ │
│    info          (inherit)          informational lines  | ╰─────────────────────────────────────────────╯ │
│ … 2 more                                                 | SWATCHES                                        │
│ c contrast:  » █ current  █ white-out  █ black-out       | title        The quick                          │
│                                                          | primary      brown fox                          │
├────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ↑↓ move / type filter / enter edit / s save / c contrast / ? help                                          │
╰────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

## Why this exists

Every app in the family used to ship its own theme editor. They looked almost the same, behaved a little differently, and — the worst part — wrote themes that weren't interchangeable. You'd spend ten minutes getting passage to look the way you wanted, then open ssherpa and do the whole thing over with different keys and different options.

termtint is the one editor for all of them. Same 16 roles, same file format, same preview. A theme built here is a file you can drop into any app in the family and it just works.

## Install

```sh
brew install --cask 0xbenc/tap/termtint
# or
go install github.com/0xbenc/termtint/cmd/termtint@latest
```

## The studio

Running `termtint` (it wants a TTY) drops you into the theme set:

```
THEME              KIND     PATH
terminal           builtin  (builtin palette)
tint               builtin  (builtin palette)
midnight           named    ~/.config/termtint/themes/midnight.theme
passage (live)     live     ~/.config/passage/theme.conf
ssherpa (live)     live     ~/.config/ssherpa/theme.conf
dangit (live)      live     ~/.config/dangit/theme.conf  (no config yet)
bitty (live)       live     ~/.config/bitty/theme.conf  (no config yet)
```

Three kinds of rows:

- **builtin** — the base palettes. `terminal` is the family's canonical one: the same values the sibling apps use, so what you see in the preview is what the app will render. `tint` is the brand palette.
- **named** — themes you've saved, living in `~/.config/termtint/themes/`.
- **live** — each app's actual `theme.conf`. Saving a live entry writes straight into that app's config, atomically, with a backup of whatever was there before.

Open a row (arrows + enter, or `termtint passage` from the shell) and you're in the studio: roles on the left, live preview on the right. The preview is the point — it renders every role an app can paint, from the selection bar to fuzzy-match highlights to the spinner and the footer overflow. Nothing about a color choice is a guess.

### The roles

Sixteen, in five groups: **text** (title, primary, secondary, accent, muted, subtle, foreground), **selection** (selected, selected_bar), **chrome** (border), **state** (success, warning, danger, info), and **special** (search, pill).

Each role takes any spec your terminal understands — a name (`green`, `bright-cyan`), a 256-color index (`38;5;253`), truecolor (`38;2;255;183;197`), SGR styles (`bold`, `dim`, `underline`, `reverse`) — or left empty, which means "inherit from the base."

Editing a role gives you four lanes — foreground, background, style, preset — plus a raw lane for when you'd rather just type the spec. ⏎ applies, esc walks away.

### Contrast modes

When a theme is dim, half-finished, or just hard to read on your terminal, hit `c`. The whole surface cycles **current → white-out → black-out**:

- **current** — the theme as it actually is.
- **white-out** — everything in bold white.
- **black-out** — everything in black.

White-out and black-out are lenses. They flatten the theme to one color so you can see all sixteen roles at once without squinting — handy for starting from scratch or for a theme that's gone weird. The lens itself never touches your theme.

What it does touch is the save. If you press `s` while a lens is up, termtint takes it literally: the save rewrites **all sixteen roles** in that color, and the review shows you the entire change before you confirm. That's the fast way to make a clean high-contrast theme. Want to keep some colors? Switch back to current first. What you confirm is exactly what the app gets — no surprises.

## From the shell

Not everything needs the studio:

```
termtint                       the studio (wants a TTY)
termtint <entry>               open one theme in the studio
termtint list                  the theme set
termtint show [entry]          the role table
termtint export [entry]        a portable .theme (stdout, or --out FILE)
termtint import <file>         validate a .theme; --name saves it as a named theme
termtint install <app>         write a theme into an app's theme.conf
                               (--theme ENTRY, --theme-file FILE, --dry-run)
termtint snapshot              render the studio frame to stdout
termtint version
```

Flags are accepted anywhere on the line: `--no-color`, `--json` (list/show/import), `--intro` / `--no-intro`. Exit codes: 0 fine, 1 not fine, 2 you typoed something.

`snapshot` is the one I reach for in tests and bug reports — it renders the studio frame at a given size with no TTY required.

## The `.theme` file

A saved theme is a full dump — every role written out:

```
# termtheme v1
# source = termtint 0.1.0-dev

format = 1
title = 1;36
primary = bold red
selected_bar = 48;2;50;40;60
...
```

That's deliberate. The file is self-contained, so a theme can't land on another machine — or in another app — missing its base palette. And `import` parses and validates before it saves, so a broken file gets rejected at the door instead of blowing up an app at runtime.

## Environment

- `NO_COLOR` — plain output (`--no-color` does the same)
- `TERTINT_NO_INTRO` / `TERTINT_INTRO_ALWAYS` — kill or force the startup intro
- `TERTINT_STATE_DIR` — where termtint keeps its state (the first-run hint and friends)
- `<APP>_THEME_FILE` — e.g. `PASSAGE_THEME_FILE=/elsewhere/theme.conf` points an app at a different config; termtint honors the same variable when reading and writing live entries

## Development

Plain Go, bubbletea v2, and the family's own libraries (`termtheme`, `termnav`, `termchrome`, `termintro`). The bar for a change:

```sh
gofmt -l .            # empty
go vet ./...          # clean
go test -race ./...   # green, including a PTY e2e that drives the real binary
```

## License

MIT — do what you want, no warranty, the usual.
