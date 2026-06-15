# 📝 `markdown` — Markdown to HTML for Starlark

[![Go Reference](https://pkg.go.dev/badge/github.com/starpkg/markdown.svg)](https://pkg.go.dev/github.com/starpkg/markdown)
[![codecov](https://codecov.io/gh/starpkg/markdown/graph/badge.svg)](https://codecov.io/gh/starpkg/markdown)
![binary footprint](https://img.shields.io/badge/binary_footprint-%2B1.2_MB-blue)

Convert [Markdown](https://commonmark.org/) to HTML from Starlark, built on
[goldmark](https://github.com/yuin/goldmark). CommonMark-compliant with opt-in
extensions (tables, task lists, strikethrough, autolinks, footnotes, definition
lists, typographer, emoji) and configurable rendering (hard wraps, auto heading
IDs).

## Overview

This is a `starpkg` module: starpkg provides support for necessary **local**
operations plus simple abstractions over common **online** services, for ease of
use. Markdown rendering is a purely **local** capability — it runs entirely
in-process (no network, no filesystem), so a script can turn untrusted Markdown
into HTML without reaching out to anything.

- **Two builtins** — `convert` renders in one call; `create_converter` builds a
  reusable converter with rendering options frozen in.
- **Secure by default** — raw HTML in the source is filtered out unless you
  opt in with `unsafe=True`, and renderer panics are recovered into errors.
- **Bounded input** — a `max_input_bytes` cap rejects oversized input before it
  reaches the renderer.

For the complete per-builtin reference — signatures, parameters, returns,
errors, examples — and the configuration accessors, see
**[docs/API.md](docs/API.md)**.

## Installation

```bash
go get github.com/starpkg/markdown
```

> **Go floor:** this module requires **Go 1.22+**. It tracks the **latest**
> [goldmark](https://github.com/yuin/goldmark) (no downgrade) for upstream
> security and correctness fixes; the recent goldmark releases raise their own
> Go floor, so this module relaxes its floor to match (per ENG-09 SEP — a module
> may raise its floor when a dependency requires it, like the `email` module).

## Quickstart

Wire the module into a Starlet interpreter, then `load("markdown", …)` from a
script:

```go
package main

import (
	"fmt"

	"github.com/1set/starlet"
	"github.com/starpkg/markdown"
)

func main() {
	mod := markdown.NewModule()
	interpreter := starlet.NewWithLoaders(nil, nil, starlet.ModuleLoaderMap{
		"markdown": mod.LoadModule(),
	})

	script := `
load("markdown", "convert")
html = convert(text="# Hello World\n\nThis is **bold** text.")
print(html)
`
	if _, err := interpreter.RunScript([]byte(script), nil); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
```

From Starlark, render in one call or build a reusable converter:

```python
load("markdown", "convert", "create_converter")

# One-shot conversion.
html = convert(text="# Hello World\n\nThis is **bold** text.")
print(html)
# <h1 id="hello-world">Hello World</h1>
# <p>This is <strong>bold</strong> text.</p>

# Reusable converter with preset options.
to_html = create_converter(table=False, linkify=False, hard_wraps=True)
print(to_html("# Title\n\nFirst line\nSecond line"))
```

## Starlark API at a glance

Top-level builtins (`load("markdown", …)`):

- `convert(text, unsafe?, heading_id?, linkify?, table?, task_list?, strikethrough?, footnote?, definition?, typograph?, emoji?, hard_wraps?)` — render Markdown to HTML in one call (`text` accepts string / bytes / None).
- `create_converter(unsafe?, heading_id?, linkify?, table?, task_list?, strikethrough?, footnote?, definition?, typograph?, emoji?, hard_wraps?)` — build a reusable `custom_converter` with options frozen in.

Converter object:

- `custom_converter(text)` — the callable returned by `create_converter`; renders a Markdown **string** with the preset options (no further keywords).

Configuration accessors (`get_<key>` / `set_<key>`):

- `get_max_input_bytes()` / `set_max_input_bytes(n)` — read / set the input-size cap (`0` disables it).

See **[docs/API.md](docs/API.md)** for the full signatures, return values,
errors, the shared rendering-option table, and examples of every builtin above.

## Configuration

The module's one option, `max_input_bytes`, bounds untrusted input before it
reaches the renderer; it is configured via the `MARKDOWN_MAX_INPUT_BYTES`
environment variable or the `get_max_input_bytes` / `set_max_input_bytes`
accessor builtins. See the
[Configuration section of docs/API.md](docs/API.md#configuration) for the full
option table, default, accessors, and details.

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file
for details.
