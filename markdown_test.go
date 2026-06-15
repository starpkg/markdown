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
//   - TestArgumentValidation:  bad/missing/None arguments to convert and to the
//                              custom_converter closure produce clean script-level
//                              errors (or empty output for None), never a host panic.
//   - TestTextInputForms:      convert accepts text as string, bytes, or None;
//                              custom_converter accepts a string only.
//   - TestExtensionToggles:    per-extension on/off behavior reachable from script
//                              options (footnote, definition list, typographer,
//                              heading IDs) through both entry points.
//   - TestMaxInputBytesEdges:  cap edge cases — env value 0 and negative caps both
//                              disable the cap; the get/set accessors round-trip and
//                              a changed cap takes effect on the next conversion.
//   - TestNoHostPanic:         adversarial inputs (deep nesting, oversized cap values,
//                              malformed markdown) surface as errors or finite output,
//                              never crashing the host.

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

// runScript loads a fresh markdown module into a starlet interpreter and runs
// the given script, returning the execution error (nil on success).
func runScript(t *testing.T, script string) error {
	t.Helper()
	mod := markdown.NewModule()
	interpreter := starlet.NewWithLoaders(nil, nil, starlet.ModuleLoaderMap{
		"markdown": mod.LoadModule(),
	})
	_, err := interpreter.RunScript([]byte(script), nil)
	return err
}

// TestArgumentValidation verifies that malformed or missing arguments to the
// script-facing builtins produce clean, descriptive script-level errors rather
// than crashing the host. It covers both the top-level convert() builtin and the
// custom_converter closure returned by create_converter().
func TestArgumentValidation(t *testing.T) {
	cases := []struct {
		name     string
		script   string
		wantErr  bool
		contains string // substring required in the error message when wantErr
	}{
		{
			name:     "convert missing text",
			script:   `load("markdown", "convert")` + "\n" + `convert()`,
			wantErr:  true,
			contains: "text",
		},
		{
			name:     "convert text wrong type int",
			script:   `load("markdown", "convert")` + "\n" + `convert(text=123)`,
			wantErr:  true,
			contains: "string, bytes or None",
		},
		{
			name:     "convert text wrong type list",
			script:   `load("markdown", "convert")` + "\n" + `convert(text=[1, 2, 3])`,
			wantErr:  true,
			contains: "string, bytes or None",
		},
		{
			name:     "convert unknown keyword",
			script:   `load("markdown", "convert")` + "\n" + `convert(text="# hi", nope=True)`,
			wantErr:  true,
			contains: "nope",
		},
		{
			name:     "convert bad option type",
			script:   `load("markdown", "convert")` + "\n" + `convert(text="# hi", emoji="yes")`,
			wantErr:  true,
			contains: "emoji",
		},
		{
			name:     "custom_converter no arg",
			script:   `load("markdown", "create_converter")` + "\nc = create_converter()\nc()",
			wantErr:  true,
			contains: "custom_converter",
		},
		{
			name:     "custom_converter int arg",
			script:   `load("markdown", "create_converter")` + "\nc = create_converter()\nc(123)",
			wantErr:  true,
			contains: "want string",
		},
		{
			name:     "custom_converter bytes arg rejected",
			script:   `load("markdown", "create_converter")` + "\nc = create_converter()\nc(b\"# hi\")",
			wantErr:  true,
			contains: "want string",
		},
		{
			name:     "custom_converter too many args",
			script:   `load("markdown", "create_converter")` + "\nc = create_converter()\nc(\"# a\", \"# b\")",
			wantErr:  true,
			contains: "custom_converter",
		},
		{
			name:    "create_converter bad option type",
			script:  `load("markdown", "create_converter")` + "\n" + `create_converter(table="nope")`,
			wantErr: true,
			// the rejected keyword is named in the unpack error
			contains: "table",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runScript(t, tc.script)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				if tc.contains != "" && !strings.Contains(err.Error(), tc.contains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.contains)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestTextInputForms verifies the accepted shapes of the text argument:
// convert() takes a string, bytes, or None (None renders as empty), while the
// custom_converter closure accepts a string only.
func TestTextInputForms(t *testing.T) {
	t.Run("convert accepts string", func(t *testing.T) {
		err := runScript(t, `
load("markdown", "convert")
def check():
    html = convert(text="# Title")
    if "<h1" not in html:
        fail("expected heading, got: " + html)
check()
`)
		if err != nil {
			t.Fatalf("string input failed: %v", err)
		}
	})

	t.Run("convert accepts bytes", func(t *testing.T) {
		err := runScript(t, `
load("markdown", "convert")
def check():
    html = convert(text=b"# Title")
    if "<h1" not in html:
        fail("expected heading from bytes input, got: " + html)
check()
`)
		if err != nil {
			t.Fatalf("bytes input failed: %v", err)
		}
	})

	t.Run("convert None renders empty", func(t *testing.T) {
		err := runScript(t, `
load("markdown", "convert")
def check():
    html = convert(text=None)
    # None is treated as empty input; goldmark emits nothing for it.
    if html != "":
        fail("expected empty output for None, got: [" + html + "]")
check()
`)
		if err != nil {
			t.Fatalf("None input failed: %v", err)
		}
	})

	t.Run("custom_converter accepts string", func(t *testing.T) {
		err := runScript(t, `
load("markdown", "create_converter")
def check():
    c = create_converter()
    html = c("# Title")
    if "<h1" not in html:
        fail("expected heading, got: " + html)
check()
`)
		if err != nil {
			t.Fatalf("custom_converter string input failed: %v", err)
		}
	})
}

// TestExtensionToggles verifies that the per-extension option flags reachable
// from script input actually turn goldmark features on and off. These are the
// option branches not exercised by the headline end-to-end test (footnote,
// definition list, typographer, heading IDs) and they are checked through both
// convert() and create_converter().
func TestExtensionToggles(t *testing.T) {
	t.Run("footnote off by default, on when enabled", func(t *testing.T) {
		err := runScript(t, `
load("markdown", "convert")
def check():
    src = "See note.[^1]\n\n[^1]: body text."
    # Default: footnotes disabled -> goldmark emits no footnote-ref markup.
    off = convert(text=src)
    if "footnote-ref" in off:
        fail("footnotes should be off by default, got: " + off)
    # Enabled: goldmark emits footnote anchors/refs.
    on = convert(text=src, footnote=True)
    if "footnote-ref" not in on:
        fail("footnote=True should render footnote refs, got: " + on)
check()
`)
		if err != nil {
			t.Fatalf("footnote toggle failed: %v", err)
		}
	})

	t.Run("typographer rewrites quotes only when enabled", func(t *testing.T) {
		err := runScript(t, `
load("markdown", "convert")
def check():
    src = '"quoted"'
    off = convert(text=src)
    if "&ldquo;" in off or "&rdquo;" in off:
        fail("typographer should be off by default, got: " + off)
    on = convert(text=src, typograph=True)
    if "&ldquo;" not in on:
        fail("typograph=True should produce smart quotes, got: " + on)
check()
`)
		if err != nil {
			t.Fatalf("typographer toggle failed: %v", err)
		}
	})

	t.Run("definition list only when enabled", func(t *testing.T) {
		err := runScript(t, `
load("markdown", "convert")
def check():
    src = "Term\n: Definition"
    off = convert(text=src)
    if "<dl>" in off:
        fail("definition lists should be off by default, got: " + off)
    on = convert(text=src, definition=True)
    if "<dl>" not in on:
        fail("definition=True should render a <dl>, got: " + on)
check()
`)
		if err != nil {
			t.Fatalf("definition-list toggle failed: %v", err)
		}
	})

	t.Run("heading id auto-generated by default, suppressed when off", func(t *testing.T) {
		err := runScript(t, `
load("markdown", "convert", "create_converter")
def check():
    src = "# Hello World"
    # Default: heading_id=True -> goldmark adds an id attribute.
    on = convert(text=src)
    if 'id="hello-world"' not in on:
        fail("heading_id should be on by default, got: " + on)
    # Disabled through create_converter: no id attribute.
    no_id = create_converter(heading_id=False)
    off = no_id(src)
    if 'id=' in off:
        fail("heading_id=False should omit id, got: " + off)
check()
`)
		if err != nil {
			t.Fatalf("heading-id toggle failed: %v", err)
		}
	})
}

// TestMaxInputBytesEdges exercises the input-size cap beyond the headline test:
// an env value of 0 disables the cap, a negative cap also disables it (matching
// the documented "0 disables" semantics), and the script-facing get/set accessors
// round-trip so a script-changed cap takes effect on the very next conversion.
func TestMaxInputBytesEdges(t *testing.T) {
	t.Run("env zero disables the cap", func(t *testing.T) {
		t.Setenv("MARKDOWN_MAX_INPUT_BYTES", "0")
		err := runScript(t, `
load("markdown", "convert")
def check():
    # Far larger than the 5 MiB default — would be rejected if the cap were active.
    big = "a " * 4000000
    html = convert(text=big)
    if len(html) == 0:
        fail("expected non-empty output with cap disabled")
check()
`)
		if err != nil {
			t.Fatalf("env=0 should disable cap, got: %v", err)
		}
	})

	t.Run("negative cap disables the cap", func(t *testing.T) {
		err := runScript(t, `
load("markdown", "convert", "set_max_input_bytes", "get_max_input_bytes")
def check():
    set_max_input_bytes(-1)
    if get_max_input_bytes() != -1:
        fail("expected cap to read back as -1")
    # A negative cap is non-positive, so checkInputSize treats it as disabled.
    html = convert(text="# this heading is comfortably over a couple of bytes")
    if "<h1" not in html:
        fail("negative cap should allow conversion, got: " + html)
check()
`)
		if err != nil {
			t.Fatalf("negative cap should disable the cap, got: %v", err)
		}
	})

	t.Run("set then enforce on next convert", func(t *testing.T) {
		err := runScript(t, `
load("markdown", "convert", "set_max_input_bytes", "get_max_input_bytes")
def check():
    set_max_input_bytes(8)
    if get_max_input_bytes() != 8:
        fail("expected cap of 8")
    # An input clearly over 8 bytes must now be rejected.
    convert(text="# this is well over eight bytes")
check()
`)
		if err == nil || !strings.Contains(err.Error(), "max_input_bytes") {
			t.Fatalf("expected max_input_bytes error after set, got: %v", err)
		}
	})

	t.Run("input exactly at cap passes, one over fails", func(t *testing.T) {
		// "abcd" is 4 bytes; a cap of 4 must accept it.
		err := runScript(t, `
load("markdown", "convert", "set_max_input_bytes")
def check():
    set_max_input_bytes(4)
    html = convert(text="abcd")
    if "abcd" not in html:
        fail("input exactly at cap should convert, got: " + html)
check()
`)
		if err != nil {
			t.Fatalf("input exactly at cap should pass, got: %v", err)
		}
		// One byte over the cap must fail.
		err = runScript(t, `
load("markdown", "convert", "set_max_input_bytes")
def check():
    set_max_input_bytes(4)
    convert(text="abcde")
check()
`)
		if err == nil || !strings.Contains(err.Error(), "max_input_bytes") {
			t.Fatalf("input one byte over cap should fail, got: %v", err)
		}
	})
}

// TestNoHostPanic asserts the no-host-panic invariant adversarially: inputs that
// could plausibly stress the renderer (deeply nested structures, malformed
// markup, oversized cap values) must surface as a normal script error or finite
// output, never as a Go panic that crashes the host. runScript already runs the
// script through the interpreter; a host panic would fail the test with a stack
// trace rather than a returned error, so reaching the assertions at all proves
// the property.
func TestNoHostPanic(t *testing.T) {
	cases := []struct {
		name string
		// wantErr: the adversarial input must surface as a clean script error
		// rather than a host panic (the big.Int cap). When false, the input must
		// convert successfully and the in-script fail() assertions must hold.
		wantErr bool
		script  string
	}{
		{
			name: "moderately deep blockquote nesting",
			script: `
load("markdown", "convert")
def check():
    deep = ">" * 2000 + " hi"
    html = convert(text=deep)
    if len(html) == 0:
        fail("expected output for nested blockquotes")
check()
`,
		},
		{
			name: "unterminated emphasis and code spans",
			script: `
load("markdown", "convert")
def check():
    html = convert(text="*** **bold _ ` + "`" + `code __ ~~strike")
    # Malformed markup must still produce a string, not crash.
    if type(html) != "string":
        fail("expected a string result")
check()
`,
		},
		{
			name: "all extensions on with mixed content",
			script: `
load("markdown", "convert")
def check():
    src = "# H\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n- [x] done\n\n~~s~~ :smile: term\n: def\n\nnote[^1]\n\n[^1]: body"
    html = convert(text=src, footnote=True, definition=True, typograph=True, emoji=True, hard_wraps=True, unsafe=True)
    if len(html) == 0:
        fail("expected output with all extensions enabled")
check()
`,
		},
		{
			name:    "oversized big.Int cap is a clean error",
			wantErr: true,
			script: `
load("markdown", "set_max_input_bytes")
set_max_input_bytes(999999999999999999999999999999)
`,
		},
		{
			name: "empty string input converts to empty output",
			script: `
load("markdown", "convert")
def check():
    html = convert(text="")
    if html != "":
        fail("empty input should yield empty output, got: [" + html + "]")
check()
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The invariant under test is "no host panic": a host panic would
			// crash the test binary with a stack trace instead of the
			// interpreter returning. Beyond that, we still assert the per-case
			// expectation so the in-script fail() checks (and the big.Int
			// clean-error claim) are actually enforced, not silently swallowed.
			err := runScript(t, tc.script)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected a clean script error, got nil")
				}
			} else if err != nil {
				t.Fatalf("expected clean conversion, got error: %v", err)
			}
		})
	}
}
