// Package markdown provides a Starlark module for converting markdown to HTML.
package markdown

import (
	"bytes"
	"fmt"

	"github.com/1set/starlet"
	"github.com/1set/starlet/dataconv/types"
	"github.com/starpkg/base"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"go.starlark.net/starlark"
)

// ModuleName defines the expected name for this module when used in Starlark's load() function, e.g., load('markdown', 'convert')
const ModuleName = "markdown"

var none = starlark.None

// Module wraps the ConfigurableModule with specific functionality for markdown conversion.
type Module struct {
	cfgMod *base.ConfigurableModule
}

// NewModule creates a new instance of Module with default configurations.
func NewModule() *Module {
	return &Module{
		cfgMod: base.NewConfigurableModule(),
	}
}

// LoadModule returns the Starlark module loader with markdown-specific functions.
func (m *Module) LoadModule() starlet.ModuleLoader {
	// Module functions
	funcs := starlark.StringDict{
		"convert":          m.genConvertFunc(),
		"create_converter": m.genCreateConverterFunc(),
	}
	return m.cfgMod.LoadModule(ModuleName, funcs)
}

// isTruthy checks if a starlark.Value is truthy (not None and not false)
func isTruthy(v starlark.Value) bool {
	if v == none {
		return false
	}
	if b, ok := v.(starlark.Bool); ok {
		return bool(b)
	}
	return true
}

// markdownOptions contains the options for configuring the markdown converter
type markdownOptions struct {
	unsafe           bool
	enableHeadingID  bool
	enableLinkify    bool
	enableTable      bool
	enableTaskList   bool
	enableStrike     bool
	enableFootnote   bool
	enableDefinition bool
	enableTypograph  bool
	enableEmoji      bool
	hardWraps        bool
}

// createMarkdownConverter creates a goldmark converter with the specified options
func createMarkdownConverter(opts markdownOptions) goldmark.Markdown {
	mdOptions := []goldmark.Option{}

	// Add renderer options
	rendererOptions := []renderer.Option{}
	if opts.unsafe {
		rendererOptions = append(rendererOptions, html.WithUnsafe())
	}
	if opts.hardWraps {
		rendererOptions = append(rendererOptions, html.WithHardWraps())
	}
	if len(rendererOptions) > 0 {
		mdOptions = append(mdOptions, goldmark.WithRendererOptions(rendererOptions...))
	}

	// Add parser options
	parserOptions := []parser.Option{}
	if opts.enableHeadingID {
		parserOptions = append(parserOptions, parser.WithAutoHeadingID())
	}
	if len(parserOptions) > 0 {
		mdOptions = append(mdOptions, goldmark.WithParserOptions(parserOptions...))
	}

	// Add extensions
	extensions := []goldmark.Extender{}
	if opts.enableTable {
		extensions = append(extensions, extension.Table)
	}
	if opts.enableLinkify {
		extensions = append(extensions, extension.Linkify)
	}
	if opts.enableTaskList {
		extensions = append(extensions, extension.TaskList)
	}
	if opts.enableStrike {
		extensions = append(extensions, extension.Strikethrough)
	}
	if opts.enableFootnote {
		extensions = append(extensions, extension.Footnote)
	}
	if opts.enableDefinition {
		extensions = append(extensions, extension.DefinitionList)
	}
	if opts.enableTypograph {
		extensions = append(extensions, extension.Typographer)
	}
	if opts.enableEmoji {
		extensions = append(extensions, emoji.Emoji)
	}

	if len(extensions) > 0 {
		mdOptions = append(mdOptions, goldmark.WithExtensions(extensions...))
	}

	// Create markdown converter
	return goldmark.New(mdOptions...)
}

// convertMarkdownToHTML converts markdown text to HTML using the given converter,
// recovering any goldmark panic into an error.
func convertMarkdownToHTML(md goldmark.Markdown, text string) (s string, err error) {
	defer func() {
		if r := recover(); r != nil {
			s, err = "", fmt.Errorf("failed to convert markdown to HTML: convert panic: %v", r)
		}
	}()
	var buf bytes.Buffer
	if cerr := md.Convert([]byte(text), &buf); cerr != nil {
		return "", fmt.Errorf("failed to convert markdown to HTML: %v", cerr)
	}
	return buf.String(), nil
}

// parseOptions unpacks the markdown options from Starlark values
func parseOptions(
	unsafe, headingID, linkify, table, taskList, strike,
	footnote, definition, typograph, emojiEnabled, hardWraps starlark.Bool,
) markdownOptions {
	return markdownOptions{
		unsafe:           bool(unsafe),
		enableHeadingID:  bool(headingID),
		enableLinkify:    bool(linkify),
		enableTable:      bool(table),
		enableTaskList:   bool(taskList),
		enableStrike:     bool(strike),
		enableFootnote:   bool(footnote),
		enableDefinition: bool(definition),
		enableTypograph:  bool(typograph),
		enableEmoji:      bool(emojiEnabled),
		hardWraps:        bool(hardWraps),
	}
}

// genConvertFunc generates the Starlark callable function to convert markdown to HTML.
func (m *Module) genConvertFunc() starlark.Callable {
	return starlark.NewBuiltin(ModuleName+".convert", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var (
			markdownText     = types.NewNullableStringOrBytesNoDefault()
			unsafe           = starlark.Bool(false)
			enableHeadingID  = starlark.Bool(true)
			enableLinkify    = starlark.Bool(true)
			enableTable      = starlark.Bool(true)
			enableTaskList   = starlark.Bool(true)
			enableStrike     = starlark.Bool(true)
			enableFootnote   = starlark.Bool(false)
			enableDefinition = starlark.Bool(false)
			enableTypograph  = starlark.Bool(false)
			enableEmoji      = starlark.Bool(false)
			hardWraps        = starlark.Bool(false)
		)

		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"text", markdownText,
			"unsafe?", &unsafe,
			"heading_id?", &enableHeadingID,
			"linkify?", &enableLinkify,
			"table?", &enableTable,
			"task_list?", &enableTaskList,
			"strikethrough?", &enableStrike,
			"footnote?", &enableFootnote,
			"definition?", &enableDefinition,
			"typograph?", &enableTypograph,
			"emoji?", &enableEmoji,
			"hard_wraps?", &hardWraps,
		); err != nil {
			return none, err
		}

		// Parse options and create the markdown converter
		opts := parseOptions(
			unsafe, enableHeadingID, enableLinkify, enableTable,
			enableTaskList, enableStrike, enableFootnote, enableDefinition,
			enableTypograph, enableEmoji, hardWraps,
		)
		md := createMarkdownConverter(opts)

		// Convert markdown to HTML
		html, err := convertMarkdownToHTML(md, markdownText.GoString())
		if err != nil {
			return none, err
		}

		return starlark.String(html), nil
	})
}

// genCreateConverterFunc generates the Starlark callable function to create a configured markdown converter.
func (m *Module) genCreateConverterFunc() starlark.Callable {
	return starlark.NewBuiltin(ModuleName+".create_converter", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var (
			unsafe           = starlark.Bool(false)
			enableHeadingID  = starlark.Bool(true)
			enableLinkify    = starlark.Bool(true)
			enableTable      = starlark.Bool(true)
			enableTaskList   = starlark.Bool(true)
			enableStrike     = starlark.Bool(true)
			enableFootnote   = starlark.Bool(false)
			enableDefinition = starlark.Bool(false)
			enableTypograph  = starlark.Bool(false)
			enableEmoji      = starlark.Bool(false)
			hardWraps        = starlark.Bool(false)
		)

		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"unsafe?", &unsafe,
			"heading_id?", &enableHeadingID,
			"linkify?", &enableLinkify,
			"table?", &enableTable,
			"task_list?", &enableTaskList,
			"strikethrough?", &enableStrike,
			"footnote?", &enableFootnote,
			"definition?", &enableDefinition,
			"typograph?", &enableTypograph,
			"emoji?", &enableEmoji,
			"hard_wraps?", &hardWraps,
		); err != nil {
			return none, err
		}

		// Parse options
		opts := parseOptions(
			unsafe, enableHeadingID, enableLinkify, enableTable,
			enableTaskList, enableStrike, enableFootnote, enableDefinition,
			enableTypograph, enableEmoji, hardWraps,
		)

		// Create a converter function that takes a markdown string and returns HTML
		converter := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var markdownText starlark.String
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &markdownText); err != nil {
				return none, err
			}

			// Create the markdown converter with the preset options
			md := createMarkdownConverter(opts)

			// Convert markdown to HTML
			html, err := convertMarkdownToHTML(md, markdownText.GoString())
			if err != nil {
				return none, err
			}

			return starlark.String(html), nil
		}

		return starlark.NewBuiltin("custom_converter", converter), nil
	})
}
