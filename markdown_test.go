package markdown_test

// Test sections:
//   - TestMarkdownConversion:  end-to-end conversion + extensions (headings, bold,
//                              tables, autolinks, strikethrough, task lists, emoji,
//                              hard wraps) via convert and create_converter.
//   - TestUnsafeDefault:       secure-by-default raw-HTML handling — raw HTML is
//                              filtered out by default and only passed through when
//                              opted in with unsafe=True (for both convert and
//                              create_converter).
//   - TestMaxInputBytes:       the max_input_bytes host config cap — a tiny cap
//                              rejects oversized input with a clean error (for both
//                              convert and create_converter), while normal-sized
//                              input still converts.

import (
	"strings"
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
load("markdown", "convert", "create_converter")

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
    minimal_converter = create_converter(
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

// TestUnsafeDefault verifies the secure-by-default posture: raw HTML in the
// markdown source is filtered out unless the caller opts in with unsafe=True.
// It covers both convert() and create_converter().
func TestUnsafeDefault(t *testing.T) {
	mod := markdown.NewModule()
	lazyLoaders := starlet.ModuleLoaderMap{
		"markdown": mod.LoadModule(),
	}
	interpreter := starlet.NewWithLoaders(nil, nil, lazyLoaders)

	script := `
load("markdown", "convert", "create_converter")

raw = "Hello <script>alert('x')</script> world"

def test_unsafe_default():
    # Default: raw HTML is filtered out (secure by default).
    safe_html = convert(text=raw)
    if "<script>" in safe_html:
        fail("default convert should strip raw HTML, got: " + safe_html)

    # Opt in: unsafe=True passes raw HTML through.
    unsafe_html = convert(text=raw, unsafe=True)
    if "<script>" not in unsafe_html:
        fail("convert(unsafe=True) should pass raw HTML through, got: " + unsafe_html)

    # create_converter shares the same default.
    safe_conv = create_converter()
    if "<script>" in safe_conv(raw):
        fail("default create_converter should strip raw HTML")

    unsafe_conv = create_converter(unsafe=True)
    if "<script>" not in unsafe_conv(raw):
        fail("create_converter(unsafe=True) should pass raw HTML through")

test_unsafe_default()
`

	if _, err := interpreter.RunScript([]byte(script), nil); err != nil {
		t.Fatalf("Failed to execute unsafe-default test script: %v", err)
	}
}

// TestMaxInputBytes verifies the max_input_bytes host config cap: a tiny cap set
// via the MARKDOWN_MAX_INPUT_BYTES environment variable rejects oversized input
// with a clean error (for both convert and create_converter), while normal-sized
// input still converts to HTML.
func TestMaxInputBytes(t *testing.T) {
	run := func(t *testing.T, script string) error {
		t.Helper()
		mod := markdown.NewModule()
		interpreter := starlet.NewWithLoaders(nil, nil, starlet.ModuleLoaderMap{
			"markdown": mod.LoadModule(),
		})
		_, err := interpreter.RunScript([]byte(script), nil)
		return err
	}

	// Tiny cap rejects oversized input — via convert().
	t.Run("convert rejects oversized", func(t *testing.T) {
		t.Setenv("MARKDOWN_MAX_INPUT_BYTES", "8")
		err := run(t, `
load("markdown", "convert")
convert(text="# this heading is well over eight bytes")
`)
		if err == nil || !strings.Contains(err.Error(), "max_input_bytes") {
			t.Fatalf("expected max_input_bytes error, got %v", err)
		}
	})

	// Tiny cap rejects oversized input — via create_converter().
	t.Run("create_converter rejects oversized", func(t *testing.T) {
		t.Setenv("MARKDOWN_MAX_INPUT_BYTES", "8")
		err := run(t, `
load("markdown", "create_converter")
to_html = create_converter()
to_html("# this heading is well over eight bytes")
`)
		if err == nil || !strings.Contains(err.Error(), "max_input_bytes") {
			t.Fatalf("expected max_input_bytes error, got %v", err)
		}
	})

	// Normal-sized input passes (default 5 MiB cap, env unset).
	t.Run("normal input passes", func(t *testing.T) {
		err := run(t, `
load("markdown", "convert", "create_converter")

def check():
    html = convert(text="# Hello")
    if "<h1" not in html:
        fail("expected heading in convert output, got: " + html)
    to_html = create_converter()
    html2 = to_html("# Hello")
    if "<h1" not in html2:
        fail("expected heading in create_converter output, got: " + html2)

check()
`)
		if err != nil {
			t.Fatalf("normal input should convert, got %v", err)
		}
	})
}
