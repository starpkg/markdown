# 📝 `markdown` — Markdown to HTML for Starlark

[![Go Reference](https://pkg.go.dev/badge/github.com/starpkg/markdown.svg)](https://pkg.go.dev/github.com/starpkg/markdown)

Convert [Markdown](https://commonmark.org/) to HTML from Starlark, built on
[goldmark](https://github.com/yuin/goldmark). CommonMark-compliant with
opt-in extensions (tables, task lists, strikethrough, autolinks, footnotes,
definition lists, typographer, emoji) and configurable rendering (hard wraps,
auto heading IDs).

This is a `starpkg` module: starpkg provides support for necessary **local**
operations plus simple abstractions over common **online** services, for ease
of use. Markdown rendering is a purely **local** capability — it runs entirely
in-process (no network, no filesystem), so a script can turn untrusted Markdown
into HTML without reaching out to anything.

The module exposes two builtins — `convert` and `create_converter` — and
`create_converter` hands back a callable named `custom_converter`. The host
config option `max_input_bytes` is reachable from scripts as the
`get_max_input_bytes` / `set_max_input_bytes` pair.

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
| `create_converter` | `create_converter(unsafe=False, heading_id=True, linkify=True, table=True, task_list=True, strikethrough=True, footnote=False, definition=False, typograph=False, emoji=False, hard_wraps=False) -> callable` | Build a converter with preset options; returns a `custom_converter` callable. |

The callable returned by `create_converter` is a builtin named `custom_converter`.
It takes a single positional argument — the Markdown **string** to render — and
returns the HTML string, applying the options frozen in at `create_converter`
time (it accepts no further option keywords). The same `max_input_bytes` cap is
enforced on every call. Note that `convert` requires the `text` argument and
accepts it as a `string`, `bytes`, or `None` (`None` renders as empty; omitting
`text` entirely is an error), whereas `custom_converter` accepts a `string` only.

Option reference (shared by both functions):

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `text` | `string` / `bytes` / `None` | (required) | Markdown text to convert (`convert` only); the argument is required, but `None` renders as empty. |
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

| Option | Type | Default | Environment Variable | Description |
|--------|------|---------|----------------------|-------------|
| `max_input_bytes` | `int` | `5242880` | `MARKDOWN_MAX_INPUT_BYTES` | Maximum input size in bytes (5 MiB); `0` disables the cap |

The config option is exposed to Starlark by the `base` module system as a
getter/setter pair: read it with `get_max_input_bytes()` and change it with
`set_max_input_bytes(n)`. It can also be set from the environment via
`MARKDOWN_MAX_INPUT_BYTES`.

```python
load("markdown", "convert", "set_max_input_bytes", "get_max_input_bytes")

set_max_input_bytes(1 << 20)   # cap input at 1 MiB
print(get_max_input_bytes())   # 1048576
```

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
