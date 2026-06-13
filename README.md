# 📝 `markdown` — Markdown to HTML for Starlark

[![Go Reference](https://pkg.go.dev/badge/github.com/starpkg/markdown.svg)](https://pkg.go.dev/github.com/starpkg/markdown)

Convert [Markdown](https://commonmark.org/) to HTML from Starlark, built on
[goldmark](https://github.com/yuin/goldmark). CommonMark-compliant with
opt-in extensions (tables, task lists, strikethrough, autolinks, footnotes,
definition lists, typographer, emoji) and configurable rendering (hard wraps,
auto heading IDs).

## Installation

```bash
go get github.com/starpkg/markdown
```

> **Go floor:** this module requires **Go 1.22+**. It tracks the **latest**
> [goldmark](https://github.com/yuin/goldmark) (no downgrade) for upstream
> security and correctness fixes; the recent goldmark releases raise their own
> Go floor, so this module relaxes its floor to match (per ENG-09 SEP — a
> module may raise its floor when a dependency requires it, like the `email`
> module).

## Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `convert` | `convert(text, unsafe=False, heading_id=True, linkify=True, table=True, task_list=True, strikethrough=True, footnote=False, definition=False, typograph=False, emoji=False, hard_wraps=False) -> str` | Convert Markdown `text` to an HTML string. |
| `create_converter` | `create_converter(unsafe=False, heading_id=True, linkify=True, table=True, task_list=True, strikethrough=True, footnote=False, definition=False, typograph=False, emoji=False, hard_wraps=False) -> callable` | Build a converter with preset options; the returned callable takes a Markdown string and returns HTML. |

Option reference (shared by both functions):

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `text` | `string` | (required) | Markdown text to convert (`convert` only). |
| `unsafe` | `bool` | `False` | Pass raw inline/block HTML through instead of filtering it (see Safety). |
| `heading_id` | `bool` | `True` | Auto-generate `id` attributes for headings. |
| `linkify` | `bool` | `True` | Auto-link bare URLs. |
| `table` | `bool` | `True` | Enable GFM tables. |
| `task_list` | `bool` | `True` | Enable task list checkboxes (`- [ ]` / `- [x]`). |
| `strikethrough` | `bool` | `True` | Enable `~~strikethrough~~`. |
| `footnote` | `bool` | `False` | Enable footnotes (`[^1]`). |
| `definition` | `bool` | `False` | Enable definition lists. |
| `typograph` | `bool` | `False` | Enable the typographer (smart quotes, dashes, ellipses). |
| `emoji` | `bool` | `False` | Enable GitHub-style emoji shortcodes (`:smile:`). |
| `hard_wraps` | `bool` | `False` | Convert source newlines to `<br>`. |

## Usage

### In Go

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

### In Starlark

```python
load("markdown", "convert", "create_converter")

# Basic conversion.
html = convert(text="# Hello World\n\nThis is **bold** text.")
print(html)
# <h1 id="hello-world">Hello World</h1>
# <p>This is <strong>bold</strong> text.</p>

# Enable extra extensions per call.
html = convert(
    text="""
- [x] Done
- [ ] Todo

| Name | Value |
|------|-------|
| Key  | Val   |

Visit https://example.com

I :heart: Markdown!
""",
    footnote=True,
    emoji=True,
)

# Build a reusable converter with preset options.
to_html = create_converter(table=False, linkify=False, hard_wraps=True)
print(to_html("# Title\n\nFirst line\nSecond line"))
```

## Safety

Raw HTML embedded in the Markdown source is **filtered out by default**
(`unsafe=False`) — goldmark replaces it with an HTML comment placeholder. This
secure-by-default posture matches the rest of the ecosystem and prevents
script-supplied Markdown from injecting arbitrary HTML/JS.

```python
load("markdown", "convert")

src = "Hello <script>alert('x')</script> world"

convert(text=src)                # raw <script> is stripped (default)
convert(text=src, unsafe=True)   # opt in: raw HTML passes through
```

Only set `unsafe=True` when you fully trust the Markdown source. Conversion
panics from the underlying renderer are also recovered into normal errors, so
malformed input never crashes the host.

## Configuration

Untrusted Markdown is bounded by an input-size cap before it reaches the
renderer — input longer than `max_input_bytes` is rejected with a clean error
(matching the `yaml`/`toml` modules).

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `max_input_bytes` | `int` | `5242880` | Maximum input size in bytes (5 MiB); `0` disables the cap |

Settable from Starlark via `set_max_input_bytes(n)` or from the environment via
`MARKDOWN_MAX_INPUT_BYTES`.

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
