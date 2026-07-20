# inifiles

[![Go Reference](https://pkg.go.dev/badge/github.com/inverse-inc/go-config-inifiles.svg)](https://pkg.go.dev/github.com/inverse-inc/go-config-inifiles)

A Go port of the Perl [Config::IniFiles](https://metacpan.org/release/Config-IniFiles)
module for reading and writing `.ini`-style configuration files.

The goal is behavioral parity: the same INI dialect is accepted/produced and
the same round-trip-preservation guarantees hold, with an idiomatic Go API.

## Features ported

- Sections (`[name]`), parameters (`key=value`), comments (`#`/`;`)
- Multi-line / here-doc values (`parm=<<EOT … EOT`), with collision-safe EOT
  marker selection
- Backslash line continuation (`-allowcontinue`)
- Trailing (end-of-line) comments (`-handle_trailing_comment`)
- Verbatim comment, blank-line-on-write, and ordering preservation
- Line-ending detection and round-trip (`\n`, `\r`, `\r\n`, `\x15`)
- UTF-8 BOM stripping on the first line
- Case-insensitive mode (`-nocase`)
- `-default` section fallback and `-fallback` section for loose keys
- `-import` overlay/shadowing with delta output (`-delta`)
- Negative deltas (`; [sect] is deleted` / `; parm is deleted`)
- `-nomultiline` output, `-php_compat`, `-allowempty`
- Groups derived from section names (`Group Section Member`)
- Atomic file writes (temp file + rename) honoring the recorded file mode
- Refusal to write to a read-only destination

## Quick start

```go
import "github.com/inverse-inc/go-config-inifiles"

cfg, err := inifiles.NewFromFile("config.ini")
if err != nil { log.Fatal(err) }

v, ok := cfg.Val("Section", "Parameter")          // joined with "\n" if multi-line
lines, _ := cfg.Vals("Section", "Mult")            // []string

cfg.NewVal("Section", "New", "value")
if err := cfg.WriteFile("config.ini"); err != nil { log.Fatal(err) }
```

Constructors: `NewFromFile(path, opts...)`, `NewFromReader(r, opts...)`,
`NewFromBytes(b, opts...)`, `New(opts...)`.

### Options (functional)

`WithDefault`, `WithFallback`, `WithNoCase`, `WithNoMultiline`,
`WithAllowContinue`, `WithAllowEmpty`, `WithNegativeDeltas`, `WithCommentChar`,
`WithAllowedCommentChars`, `WithTrailingComment`, `WithPHPCompat`,
`WithImport`, `WithReloadWarn`.

## API notes (Perl → Go)

This port replaces Perl-only constructs with idiomatic Go:

- The tied-hash interface (`$cfg{Section}{param}`) is exposed as explicit
  methods: `Sections`, `Parameters`, `Val`/`Vals`, `NewVal`/`SetVal`/
  `PushVal`/`DelVal`, `AddSection`/`DeleteSection`/`RenameSection`/
  `CopySection`, etc.
- `wantarray` dual-context `val()` becomes `Val` (scalar, joined with `\n`)
  and `Vals` (slice).
- Errors use returned `error`s; accumulated parse errors are available via
  `Config.Errors() []ParseError`.
- `-file` accepts an `io.Reader`, `string` path, or `*os.File`; in-memory
  configs use `NewFromBytes`.

## Dropped / approximated Perl behaviors

- **Taint mode** (Perl `-T`) has no Go equivalent; the taint-only assertions of
  tests 17 are dropped. (Their non-taint assertions are covered elsewhere.)
- **`OutputConfig()`** (writing to Perl's globally-selected filehandle) has no
  Go equivalent; use `WriteTo(io.Writer)`.
- **`-allowedcommentchars`** is interpreted as a set of runes (printed,
  non-alphanumeric, non-`[]:=`) rather than a raw regex character-class body —
  identical observable behavior for all documented inputs.
- **Array-context assignment** (`$ini{s} = @arr` assigning the element count)
  has no Go equivalent and is not reproduced.

## Testing

The Perl test suite (`config-inifiles/t/*.t`) has been translated to Go table
tests. Run:

```
go test ./...
go test -cover ./...
```

Test fixtures live under `testdata/` (copied verbatim from the Perl
distribution).

## License

MIT, derived from the Perl Config::IniFiles distribution. See [LICENSE](LICENSE).