# termtint

The theme studio for the [termsystem](https://github.com/0xbenc/termsystem)
family: author a theme against the full chrome vocabulary, preview it live on
every widget the family renders, and move it losslessly between passage,
ssherpa, dangit, and bitty.

One `.theme`, every app in the family — authored live.

## Usage

```sh
termtint                              # interactive studio (TTY)
termtint passage                      # open one theme in the studio (TTY)
termtint list                         # the theme set: builtins, named themes, live apps
termtint show [entry]                 # role table for a theme
termtint export [entry] [--out FILE]  # portable .theme on stdout
termtint import <file> [--name NAME]  # validate a .theme, optionally save it named
termtint install <app> [--theme ENTRY] [--theme-file FILE] [--dry-run]
                                      # write a theme to an app's theme.conf
termtint snapshot [--entry ENTRY] [--width N] [--height N]
                                      # render the studio frame to stdout
termtint version
```

Global flags, accepted anywhere: `--no-color` (or `NO_COLOR`), `--json`
(list/show/import), `--intro` / `--no-intro`.

Exit codes: `0` success, `1` failure, `2` usage.

## Install

```sh
brew install --cask 0xbenc/tap/termtint
# or: go install github.com/0xbenc/termtint/cmd/termtint@latest
```

## License

MIT — see [`LICENSE`](LICENSE).
