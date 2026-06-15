# `markdown` — Starlark API Reference

The complete reference for every script-facing builtin, the converter object it
returns, and the configuration accessor exposed by the `markdown` module. For an
overview, installation, and a quickstart, see the [README](../README.md).

The module exposes two top-level builtins via `load("markdown", …)` — `convert`
and `create_converter` — plus a configuration accessor pair (`get_<key>` /
`set_<key>`) generated from the module's one option. `create_converter` hands
back a callable named `custom_converter` with its rendering options frozen in.

## Contents

- [Functions](#functions)
  - [`convert`](#converttext-unsafe-heading_id-linkify-table-task_list-strikethrough-footnote-definition-typograph-emoji-hard_wraps)
  - [`create_converter`](#create_converterunsafe-heading_id-linkify-table-task_list-strikethrough-footnote-definition-typograph-emoji-hard_wraps)
- [Converter object](#converter-object)
  - [`custom_converter`](#custom_convertertext)
- [Rendering options](#rendering-options)
- [Safety](#safety)
- [Configuration](#configuration)

## Functions

### `convert(text, unsafe=False, heading_id=True, linkify=True, table=True, task_list=True, strikethrough=True, footnote=False, definition=False, typograph=False, emoji=False, hard_wraps=False)`

Converts Markdown `text` to an HTML string in one call.

**Parameters:**

- `text` (string / bytes / None): Markdown source to convert. The argument is
  **required** — omitting it entirely is an error — but `None` renders as an
  empty document, and `bytes` are decoded as UTF-8.
- `unsafe` (bool): Pass raw inline/block HTML through instead of filtering it
  (default: `False`; see [Safety](#safety)).
- `heading_id` (bool): Auto-generate `id` attributes for headings (default: `True`).
- `linkify` (bool): Auto-link bare URLs (default: `True`).
- `table` (bool): Enable GFM tables (default: `True`).
- `task_list` (bool): Enable task-list checkboxes (`- [ ]` / `- [x]`) (default: `True`).
- `strikethrough` (bool): Enable `~~strikethrough~~` (default: `True`).
- `footnote` (bool): Enable footnotes (`[^1]`) (default: `False`).
- `definition` (bool): Enable definition lists (default: `False`).
- `typograph` (bool): Enable the typographer — smart quotes, dashes, ellipses (default: `False`).
- `emoji` (bool): Enable GitHub-style emoji shortcodes (`:smile:`) (default: `False`).
- `hard_wraps` (bool): Convert source newlines to `<br>` (default: `False`).

See [Rendering options](#rendering-options) for the shared option reference.

**Returns:** The rendered HTML (string).

**Errors:** Fails if `text` is omitted, if the input exceeds the configured
`max_input_bytes` cap (see [Configuration](#configuration)), or if the underlying
renderer fails. A panic in the renderer is recovered and surfaced as a normal
script error, so malformed input never crashes the host.

**Example:**

```python
load("markdown", "convert")

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
```

### `create_converter(unsafe=False, heading_id=True, linkify=True, table=True, task_list=True, strikethrough=True, footnote=False, definition=False, typograph=False, emoji=False, hard_wraps=False)`

Builds a reusable converter with rendering options preset, and returns a
callable named `custom_converter` that renders Markdown with those frozen-in
options. Useful when many strings share the same rendering configuration.

**Parameters:** the same option set as [`convert`](#converttext-unsafe-heading_id-linkify-table-task_list-strikethrough-footnote-definition-typograph-emoji-hard_wraps),
**minus** `text` — the source string is supplied later, on each call to the
returned converter. See [Rendering options](#rendering-options) for the shared
option reference.

**Returns:** A [`custom_converter`](#custom_convertertext) callable.

**Example:**

```python
load("markdown", "create_converter")

# Build a reusable converter with preset options.
to_html = create_converter(table=False, linkify=False, hard_wraps=True)
print(to_html("# Title\n\nFirst line\nSecond line"))
```

## Converter object

### `custom_converter(text)`

The callable returned by
[`create_converter`](#create_converterunsafe-heading_id-linkify-table-task_list-strikethrough-footnote-definition-typograph-emoji-hard_wraps).
It renders Markdown using the options captured at `create_converter` time and
accepts **no** further option keywords.

**Parameters:**

- `text` (string): Markdown source to render — a single positional argument.
  Unlike [`convert`](#converttext-unsafe-heading_id-linkify-table-task_list-strikethrough-footnote-definition-typograph-emoji-hard_wraps),
  this entry point accepts a **string only** (not `bytes` or `None`).

**Returns:** The rendered HTML (string).

**Errors:** The same `max_input_bytes` cap is enforced on every call; oversized
input and renderer failures surface as script errors (renderer panics are
recovered), exactly as with `convert`.

**Example:**

```python
load("markdown", "create_converter")

render = create_converter(emoji=True, footnote=True)
print(render("# Notes\n\nReady :tada:"))
print(render("Another **document**."))
```

## Rendering options

Both [`convert`](#converttext-unsafe-heading_id-linkify-table-task_list-strikethrough-footnote-definition-typograph-emoji-hard_wraps)
and [`create_converter`](#create_converterunsafe-heading_id-linkify-table-task_list-strikethrough-footnote-definition-typograph-emoji-hard_wraps)
accept the same rendering options (only `convert` takes the `text` argument).
The defaults reproduce the historical behavior: heading IDs, linkify, tables,
task lists, and strikethrough on; everything else off.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `text` | string / bytes / None | (required) | Markdown text to convert (`convert` only); the argument is required, but `None` renders as empty |
| `unsafe` | bool | `False` | Pass raw inline/block HTML through instead of filtering it (see [Safety](#safety)) |
| `heading_id` | bool | `True` | Auto-generate `id` attributes for headings |
| `linkify` | bool | `True` | Auto-link bare URLs |
| `table` | bool | `True` | Enable GFM tables |
| `task_list` | bool | `True` | Enable task-list checkboxes (`- [ ]` / `- [x]`) |
| `strikethrough` | bool | `True` | Enable `~~strikethrough~~` |
| `footnote` | bool | `False` | Enable footnotes (`[^1]`) |
| `definition` | bool | `False` | Enable definition lists |
| `typograph` | bool | `False` | Enable the typographer (smart quotes, dashes, ellipses) |
| `emoji` | bool | `False` | Enable GitHub-style emoji shortcodes (`:smile:`) |
| `hard_wraps` | bool | `False` | Convert source newlines to `<br>` |

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

The module's one configuration option is exposed to scripts as a pair of
generated accessor builtins (loaded from the `markdown` module alongside the
functions above):

- **`get_<key>()`** — returns the current value of the option.
- **`set_<key>(value)`** — sets the option (returns `None`).

An option's value resolves in priority order: an explicit `set_<key>` value, the
environment variable, then the default.

`max_input_bytes` is **not** a secret option, so it exposes **both**
`get_max_input_bytes` and `set_max_input_bytes`. (A secret option would expose
only its `set_<key>` accessor — never a getter — but this module has none.)

| Option | Getter | Setter | Type | Env var | Default | Description |
|--------|--------|--------|------|---------|---------|-------------|
| `max_input_bytes` | `get_max_input_bytes` | `set_max_input_bytes` | int | `MARKDOWN_MAX_INPUT_BYTES` | `5242880` | Maximum input size in bytes when converting (5 MiB); `0` disables the cap |

Untrusted Markdown is bounded by this input-size cap before it reaches the
renderer — input longer than `max_input_bytes` is rejected with a clean error
(matching the `yaml` / `toml` modules). The cap is enforced on every conversion
path: both `convert` and the `custom_converter` returned by `create_converter`.

**Example:**

```python
load("markdown", "convert", "get_max_input_bytes", "set_max_input_bytes")

set_max_input_bytes(1 << 20)   # cap input at 1 MiB
print(get_max_input_bytes())   # 1048576

html = convert(text="# Within the cap")
```
