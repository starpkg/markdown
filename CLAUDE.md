# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`starpkg/markdown` is an **L4 domain module** of the Star\* ecosystem: it exposes
Markdown-to-HTML conversion to Starlark scripts. A script loads the module and
calls `convert(text=...)` to render Markdown straight to an HTML string, or
`create_converter(...)` to build a reusable converter with options frozen in.

The `starpkg` positioning is: **support for necessary LOCAL operations + simple
abstractions over common ONLINE services, for ease of use.** Markdown sits
firmly on the **local** side — rendering is a pure in-process computation built
on [goldmark](https://github.com/yuin/goldmark). It touches **no network and no
filesystem**, so even untrusted Markdown can be turned into HTML without the
module reaching out to anything.

Layer position: depends downward on `starpkg/base` (the configurable-module /
config-option system), `1set/starlet` (the Machine runner + `dataconv/types`
unpackers), and transitively `1set/starlight` + `go.starlark.net`. Nothing in
the ecosystem depends on it.

## Dev commands

Pure Go library with a Makefile. From this repo:

```bash
make test                                  # -race -cover, the working bar
make ci                                    # -race -cover profile + bench compile (what CI runs)
make bench                                 # benchmarks only
go test ./... -run TestMaxInputBytes       # a single test
gofmt -l . && go vet ./...                 # must be clean before commit
go run github.com/1set/meta/doccov@master .  # the doc-coverage gate (must exit 0)
```

**Verify on the go floor in Docker** — this repo's floor is **go 1.22** (see
Release discipline), and the local toolchain may be newer. Behavior on the floor
must be checked in a container:

```bash
docker run --rm -v "$PWD":/src -v "$HOME/go/pkg/mod":/go/pkg/mod -w /src golang:1.22 go test -race -count=1 ./...
```

Integration scripts under `../test/markdown/*.star` live in the **private
`starpkg/test` repo** and auto-skip when that directory is absent (e.g. in CI);
they are not wired into this repo's Go tests.

## Architecture (the part that spans files)

The module is a thin, single-file bridge over goldmark — there is one source
file plus its test file.

- **`markdown.go`** — the whole module.
  - `Module` wraps a `base.ConfigurableModule` plus its `Extend()` view
    (`ConfigurableModuleExt`, used for typed config reads). `NewModule()`
    registers the one config option, `max_input_bytes`
    (env `MARKDOWN_MAX_INPUT_BYTES`, default `5 << 20` = 5 MiB).
  - `LoadModule()` exposes two builtins via `base.ConfigurableModule.LoadModule`:
    **`convert`** (`genConvertFunc`) and **`create_converter`**
    (`genCreateConverterFunc`). The base module system *also* auto-generates a
    `get_max_input_bytes` / `set_max_input_bytes` getter/setter pair from the
    registered config option — those are script-facing too, so they are
    documented in the README even though they are not declared in this repo's
    source.
  - `create_converter` returns a second builtin named **`custom_converter`**:
    a closure that takes a single positional Markdown **string** and renders it
    with the options captured at `create_converter` time.
  - `createMarkdownConverter(markdownOptions)` translates the option struct into
    goldmark renderer/parser options and extenders (Table, Linkify, TaskList,
    Strikethrough, Footnote, DefinitionList, Typographer, Emoji; plus
    `WithUnsafe`/`WithHardWraps`/`WithAutoHeadingID`). This is the single
    goldmark wrap point.
  - `parseOptions(...)` maps the unpacked Starlark bools into `markdownOptions`.
  - `convertMarkdownToHTML(md, text)` is the actual render call; it wraps the
    goldmark `Convert` in a `recover()`.
  - `checkInputSize(text)` enforces `max_input_bytes` before goldmark sees the
    input.

**Data flow:** script `convert(text, opts...)` → `UnpackArgs` (the `text`
unpacker is `types.NewNullableStringOrBytesNoDefault`, so `text` accepts
`string` / `bytes` / `None`) → `parseOptions` → `createMarkdownConverter` →
`checkInputSize` → `convertMarkdownToHTML` → Starlark `String`. `create_converter`
runs the same pipeline except the options are bound up front and the inner
`custom_converter` unpacks exactly one positional **string** argument.

## Invariants / hardening (preserve when editing)

The iron rule is **opt-in / default-off so old scripts run identically** — any
new safety lever must default to the historical behavior.

1. **No host panics from script input.** `convertMarkdownToHTML` wraps the
    goldmark `Convert` call in a deferred `recover()` that turns any renderer
    panic into a normal Go error (and thus a script-level error). Don't remove
    the recover; new render paths must route through this function.
2. **Bounded input.** `checkInputSize` rejects input longer than
    `max_input_bytes` (when positive) *before* it reaches goldmark, so a hostile
    or buggy script can't feed an unbounded blob into the renderer. The default
    cap is 5 MiB; `0` disables it. Every conversion path (both `convert` and the
    `custom_converter` closure) must call `checkInputSize` before converting.
3. **Secure by default — raw HTML is filtered.** `unsafe` defaults to `False`,
    so goldmark strips raw inline/block HTML from the source. Passing
    `unsafe=True` is the explicit opt-in to pass raw HTML through. This matches
    the rest of the ecosystem; don't flip the default.
4. **Backward compatibility.** The default option set (heading IDs / linkify /
    tables / task lists / strikethrough on; footnotes / definition lists /
    typographer / emoji / hard wraps off) is the historical behavior. Changing a
    default is an observable behavior change — treat it as such.

## Test organization

Group by functional goal — **do not add one `*_test.go` per fix.**
`markdown_test.go` is the home, opened with a commented section list:
`TestMarkdownConversion` (end-to-end conversion + extensions through both
`convert` and `create_converter`), `TestUnsafeDefault` (the secure-by-default
raw-HTML posture for both entry points), and `TestMaxInputBytes` (the
`max_input_bytes` host cap via the `MARKDOWN_MAX_INPUT_BYTES` env var). Add a new
test as a **section here**, not a new file. Tests are script/table-driven; no
third-party test framework. The `../test/markdown/*.star` scripts in the private
`starpkg/test` repo are the integration counterpart and auto-skip when absent.

## Documentation

Three layers must stay in sync (enforced by the doc standard,
`plan/starpkg文档标准（DOC-STD）`):

- **`README.md`** — every script-facing builtin (`convert`, `create_converter`,
  the returned `custom_converter`) and the base-generated config accessors
  (`get_max_input_bytes` / `set_max_input_bytes`) documented as a backtick
  whole-word; function names/signatures/behaviour must match the code.
- **GoDoc** — package comment + a doc comment on every exported symbol
  (`ModuleName`, `Module`, `NewModule`, `LoadModule`), first word = symbol name
  (gated by `revive`'s `exported` rule in CI).
- **doc-coverage gate** — `go run github.com/1set/meta/doccov@<pin> .` runs in
  CI via the reusable `1set/meta` workflow with `doc-coverage: true`. It scans
  this repo's non-test Go for `starlark.NewBuiltin` calls and fails if any
  builtin name is not a backtick whole-word in the README.

## Release discipline

- **Floor = go 1.22**, registered under the ENG-09 SEP policy: this module
  tracks the **latest** goldmark (no downgrade) for upstream security and
  correctness fixes, and recent goldmark releases raise their own Go floor, so
  the module relaxes its floor to match (a module may raise its floor when a
  dependency requires it — same rationale as the `email` module).
- **CI matrix** = `[1.22.x, 1.25.x]` via the centralized reusable workflow in
  `1set/meta` (pinned by commit SHA; bump the pin when meta's workflow changes).
- **Bumping the version, the go floor, or tagging are user-confirmed actions** —
  never tag autonomously; default to patch bumps; published tags are immutable
  in the Go module proxy. The `go.starlark.net` / 1set-deps pin upgrade is the
  last PR of a series and happens before any tag.
