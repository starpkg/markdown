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
    
    # Test emoji support
    emoji_md = "I :heart: markdown! :tada: :smile:"
    emoji_html = convert(text=emoji_md, emoji=True)
    print("="*50)
    print(emoji_html)
    
    has_emoji = "❤️" in emoji_html or "&#x2764;" in emoji_html
    if not has_emoji:
        fail("Emoji conversion failed to render the heart emoji")
    
    # Test hard wraps
    hard_wraps_md = """Line one
Line two"""
    
    # Without hard wraps (default)
    normal_html = convert(text=hard_wraps_md)
    print("="*50)
    print(normal_html)
    
    # With hard wraps enabled
    hard_wraps_html = convert(text=hard_wraps_md, hard_wraps=True)
    print("="*50)
    print(hard_wraps_html)
    
    has_br = "<br" in hard_wraps_html
    no_br = "<br" not in normal_html
    
    if not has_br:
        fail("Hard wraps conversion failed to add <br> tags")
    
    if not no_br:
        fail("Regular conversion should not add <br> tags")

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
