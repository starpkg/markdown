# 📝 `markdown` - Starlark Module for Markdown to HTML Conversion

A simple and powerful Starlark module for converting Markdown to HTML. Built on the [goldmark](https://github.com/yuin/goldmark) Markdown parser, this module provides a clean API for transforming Markdown content into HTML with customizable rendering options.

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

## Usage in Go

```go
package main

import (
	"fmt"
	"os"

	"github.com/1set/starlet"
	"github.com/starpkg/markdown"
)

func main() {
	// Create a new Markdown module
	mod := markdown.NewModule()

	// Create a Starlet interpreter with the module
	lazyLoaders := starlet.ModuleLoaderMap{
		"markdown": mod.LoadModule(),
	}
	interpreter := starlet.NewWithLoaders(nil, nil, lazyLoaders)

	// Run a Starlark script that uses markdown
	script := `
load("markdown", "convert")

# Sample markdown text
md = """
# Hello World

This is a **bold** statement and *italicized* text.

## Lists

* Item 1
* Item 2
  * Nested item
  * Another nested item
* Item 3

## Code

` + "```" + `go
func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

## Tables

| Name | Age | Occupation |
|------|-----|------------|
| Alice | 28 | Engineer |
| Bob | 32 | Designer |

## Emojis
I :heart: Markdown! :tada:
"""

# Convert markdown to HTML with emoji support
html = convert(text=md, emoji=True)
print(html)

# Convert with custom options
html_no_tables = convert(text=md, table=False)
print("HTML without tables:", html_no_tables)
`

	// Execute the script
	if _, err := interpreter.RunScript([]byte(script), nil); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
```

## Starlark API

### Functions

#### `convert(text, unsafe?, heading_id?, linkify?, table?, task_list?, strikethrough?, footnote?, definition?, typograph?, emoji?, hard_wraps?)`

Converts Markdown text to HTML. Parameters:

- `text`: The Markdown text to convert (required)
- `unsafe`: Allow raw HTML (default: true)
- `heading_id`: Auto-generate heading IDs (default: true)
- `linkify`: Auto-link URLs (default: true)
- `table`: Enable table support (default: true)
- `task_list`: Enable task list support (default: true)
- `strikethrough`: Enable strikethrough support (default: true)
- `footnote`: Enable footnote support (default: false)
- `definition`: Enable definition list support (default: false)
- `typograph`: Enable typographer extension (default: false)
- `emoji`: Enable emoji support for GitHub-style emojis like `:smile:` (default: false)
- `hard_wraps`: Convert newlines to `<br>` tags (default: false)

Returns the HTML result as a string.

#### `with_options(unsafe?, heading_id?, linkify?, table?, task_list?, strikethrough?, footnote?, definition?, typograph?, emoji?, hard_wraps?)`

Creates a configured converter function with preset options. Parameters are the same as `convert()`, but without the `text` parameter.

Returns a function that takes a markdown string and returns the converted HTML.

## Examples

### Basic Conversion

```python
load("markdown", "convert")

# Simple conversion
markdown_text = "# Hello World\n\nThis is a **bold** statement."
html = convert(text=markdown_text)
print(html)
```

Output:
```html
<h1 id="hello-world">Hello World</h1>
<p>This is a <strong>bold</strong> statement.</p>
```

### Advanced Features

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

# Without hard wraps (default)
html_normal = convert(text=markdown_text)
print(html_normal)
```

### Creating a Custom Converter

```python
load("markdown", "with_options")

# Create a converter with custom options
basic_converter = with_options(
    unsafe=False,
    table=False,
    strikethrough=False,
    emoji=True,
    hard_wraps=True
)

markdown_text = '''
# Hello

<script>alert('xss');</script>

| Name | Value |
|------|-------|
| Key1 | Val1  |

~~Strikethrough~~

I :heart: Markdown!
First line
Second line
'''

# Use the custom converter
html = basic_converter(markdown_text)
print(html)
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

MIT