# 📝 `markdown` - Starlark Module for Markdown to HTML Conversion

[![Go Report Card](https://goreportcard.com/badge/github.com/starpkg/markdown)](https://goreportcard.com/report/github.com/starpkg/markdown)
[![GoDoc](https://godoc.org/github.com/starpkg/markdown?status.svg)](https://godoc.org/github.com/starpkg/markdown)
[![License](https://img.shields.io/github/license/starpkg/markdown.svg)](https://github.com/starpkg/markdown/blob/master/LICENSE)

A powerful Starlark module for Markdown to HTML conversion. Built on [goldmark](https://github.com/yuin/goldmark), this module provides a clean API for transforming Markdown content with customizable rendering options.

## Features

- Simple API for Markdown to HTML conversion
- Support for CommonMark compliant Markdown
- Configurable extensions:
  - Tables
  - Strikethrough
  - Task lists
  - Auto-linked URLs
  - Footnotes
  - Definition lists
  - Typography enhancements
  - Emoji support (GitHub style emojis like `:smile:`)
- Auto heading ID generation
- Configurable HTML rendering options:
  - Hard wraps (convert newlines to `<br>` tags)
  - Allow unsafe HTML
- Create custom converters with preset options

## Installation

```bash
go get github.com/starpkg/markdown
```

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
	// Create a new Markdown module
	mod := markdown.NewModule()

	// Create a Starlet interpreter with the module
	interpreter := starlet.New(
		starlet.WithModuleLoader("markdown", mod.LoadModule()),
	)

	// Define a Starlark script using the markdown module
	script := `
load("markdown", "convert")

# Convert markdown to HTML
html = convert(text="# Hello World\n\nThis is **bold** text.")
print(html)
`

	// Execute the script
	if err := interpreter.ExecScript("example.star", script); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
```

### In Starlark

#### Basic Conversion

```python
load("markdown", "convert")

# Convert markdown to HTML
html = convert(text="# Hello World\n\nThis is **bold** text.")
print(html)

# Output:
# <h1 id="hello-world">Hello World</h1>
# <p>This is <strong>bold</strong> text.</p>
```

#### Advanced Features

```python
load("markdown", "convert")

# Markdown with various features
markdown_text = '''
# Task List Example

- [x] Completed task
- [ ] Incomplete task

# Table Example

| Name | Value |
|------|-------|
| Key1 | Val1  |
| Key2 | Val2  |

Visit https://example.com for more information.

~~Strikethrough text~~

I :heart: Markdown! :tada:
'''

html = convert(
    text=markdown_text,
    task_list=True,
    table=True,
    strikethrough=True,
    linkify=True,
    emoji=True
)
print(html)
```

#### Creating a Custom Converter

```python
load("markdown", "create_converter")

# Create a converter with custom options
basic_converter = create_converter(
    unsafe=True,         # Default: true
    heading_id=True,     # Default: true
    table=False,         # Default: true
    strikethrough=False, # Default: true
    linkify=True,        # Default: true
    task_list=True,      # Default: true
    emoji=True,          # Default: false
    footnote=False,      # Default: false
    definition=False,    # Default: false
    typograph=False,     # Default: false
    hard_wraps=True      # Default: false
)

# Use the custom converter
html = basic_converter("# Hello\n\nFirst line\nSecond line")
print(html)
```

## API Reference

### Functions

#### `convert(text, unsafe?, heading_id?, linkify?, table?, task_list?, strikethrough?, footnote?, definition?, typograph?, emoji?, hard_wraps?)`

Converts Markdown text to HTML.

**Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `text` | string | (required) | The Markdown text to convert |
| `unsafe` | bool | `true` | Allow raw HTML |
| `heading_id` | bool | `true` | Auto-generate heading IDs |
| `linkify` | bool | `true` | Auto-link URLs |
| `table` | bool | `true` | Enable table support |
| `task_list` | bool | `true` | Enable task list support |
| `strikethrough` | bool | `true` | Enable strikethrough support |
| `footnote` | bool | `false` | Enable footnote support |
| `definition` | bool | `false` | Enable definition list support |
| `typograph` | bool | `false` | Enable typographer extension |
| `emoji` | bool | `false` | Enable emoji support |
| `hard_wraps` | bool | `false` | Convert newlines to `<br>` tags |

**Returns:** HTML string

#### `create_converter(unsafe?, heading_id?, linkify?, table?, task_list?, strikethrough?, footnote?, definition?, typograph?, emoji?, hard_wraps?)`

Creates a configured converter function with preset options. Parameters are the same as `convert()`, but without the `text` parameter.

**Returns:** Function that takes a markdown string and returns HTML

## Examples

### Hard Wraps Example

```python
load("markdown", "convert")

# Markdown with line breaks
markdown_text = '''
First line
Second line
Third line
'''

# With hard wraps (newlines become <br> tags)
html_hard_wraps = convert(text=markdown_text, hard_wraps=True)
print(html_hard_wraps)
# Output:
# <p>First line<br>
# Second line<br>
# Third line</p>

# Without hard wraps (default)
html_normal = convert(text=markdown_text)
print(html_normal)
# Output:
# <p>First line
# Second line
# Third line</p>
```

## Supported Markdown Syntax

The `markdown` module supports all CommonMark syntax, plus the following extensions (when enabled):

- **Tables** - Create tables with headers and cells
- **Task Lists** - Create checkboxes with `- [ ]` and `- [x]` syntax
- **Strikethrough** - Strike out text with `~~text~~`
- **Linkify** - Automatically convert URLs to links
- **Footnotes** - Add footnotes with `[^1]` and `[^1]: explanation`
- **Definition Lists** - Create definition lists with term and description
- **Typography** - Smart quotes, dashes, ellipses, etc.
- **Emojis** - GitHub-style emojis like `:smile:` and `:heart:`
- **Hard Wraps** - Converting newlines to `<br>` tags

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.