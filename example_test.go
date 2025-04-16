package markdown_test

import (
	"testing"

	"github.com/1set/starlet"
	"github.com/starpkg/markdown"
)

func TestMarkdownConversion(t *testing.T) {
	// Create a new markdown module
	mod := markdown.NewModule()

	// Create a new starlet interpreter with the markdown module
	lazyLoaders := starlet.ModuleLoaderMap{
		"markdown": mod.LoadModule(),
	}
	interpreter := starlet.NewWithLoaders(nil, nil, lazyLoaders)

	// Define a Starlark script that uses the markdown module
	script := `
load("markdown", "convert", "with_options")

def test_markdown():
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

Visit https://example.com

~~Strikethrough text~~

- [x] Completed task
- [ ] Incomplete task

| Name | Age |
|------|-----|
| Alice | 28 |
| Bob | 32 |
"""

    # Basic conversion
    html = convert(text=md)
    print("="*50)
    print(html)

    # Verify basic conversion includes expected elements
    has_heading = "<h1" in html
    has_bold = "<strong>" in html
    has_autolink = '<a href="https://example.com">' in html
    has_table = "<table>" in html

    if not has_heading or not has_bold or not has_autolink or not has_table:
        fail("Basic conversion missing expected elements")

    # Create a custom converter with specific options (disable extensions)
    minimal_converter = with_options(
        table=False,
        linkify=False,
        strikethrough=False,
        task_list=False
    )

    # Use the custom converter
    minimal_html = minimal_converter(md)
    print("="*50)
    print(minimal_html)

    # Verify minimal conversion excludes disabled elements
    has_heading = "<h1" in minimal_html  # should still have headings
    has_autolink = '<a href="https://example.com">' in minimal_html  # should NOT have autolinks
    has_table = "<table>" in minimal_html  # should NOT have tables

    if not has_heading:
        fail("Minimal conversion should still include headings")
        
    if has_autolink:
        fail("Minimal conversion should not include autolinks")
        
    if has_table:
        fail("Minimal conversion should not include tables")

    print("All tests passed!")
    return True

# Run the test
test_result = test_markdown()
`

	// Execute the script
	_, err := interpreter.RunScript([]byte(script), nil)
	if err != nil {
		t.Fatalf("Failed to execute test script: %v", err)
	}
}
