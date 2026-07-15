// Package markdown provides a Starlark module for converting markdown to HTML.
package markdown

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/1set/starlet"
	"github.com/1set/starlet/dataconv/types"
	"github.com/starpkg/base"
	"github.com/starpkg/base/util"
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

const (
	configKeyMaxInputBytes  = "max_input_bytes"
	configKeyMaxConvertTime = "max_convert_time"
)

const defaultMaxInputBytes = 5 << 20 // 5 MiB

// defaultMaxConvertTime bounds a single conversion's wall-clock time. 0 = no
// module-level limit (the conversion is still bounded by the thread context
// deadline, if the host set one). Default 0 keeps historical behavior.
const defaultMaxConvertTime = 0.0

var (
	none              = starlark.None
	errConvertTimeout = errors.New("markdown.convert: conversion exceeded the time limit")
)

// Module wraps the ConfigurableModule with specific functionality for markdown conversion.
type Module struct {
	cfgMod *base.ConfigurableModule
	ext    *base.ConfigurableModuleExt
}

// NewModule creates a new instance of Module with default configurations.
func NewModule() *Module {
	cm, _ := base.NewConfigurableModuleWithConfigOptions(
		// max_input_bytes is the DoS byte cap; it is HOST-ONLY, so base generates
		// no set_max_input_bytes builtin — a script can neither disable nor widen
		// the limit (only the Go host can, via the option default or the env var).
		genConfigOption(configKeyMaxInputBytes, "Maximum input size in bytes when converting", defaultMaxInputBytes).
			SetHostOnly(true),
		// max_convert_time bounds a single conversion's wall-clock time. goldmark's
		// nested block-quote / list parsing is ~O(n²), so an input well under the
		// byte cap can still hang the host; the byte cap doesn't bound parse
		// complexity. HOST-ONLY: a script must not be able to disable the limit.
		genConfigOption(configKeyMaxConvertTime, "Maximum wall-clock seconds for a single conversion (0 = no module limit)", defaultMaxConvertTime).
			SetHostOnly(true),
	)
	return &Module{cfgMod: cm, ext: cm.Extend()}
}

func genConfigOption[T any](name, description string, defaultValue T) *base.ConfigOption[T] {
	return base.NewNamedConfigOption(ModuleName, name, description, defaultValue)
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

// convertMarkdownToHTML converts markdown text to HTML using the given
// converter, bounded by a wall-clock deadline and recovering any goldmark panic
// into an error.
//
// goldmark renders synchronously with no context support, and nested
// block-quote / list parsing is ~O(n²), so a pathological input (well under the
// byte cap) can spin uninterruptibly. It runs in a goroutine and the deadline —
// the thread context (so a host/sandbox execution deadline is honored) plus, if
// positive, max_convert_time — returns errConvertTimeout rather than wedging the
// interpreter. NOTE: on timeout the abandoned goroutine keeps running until
// goldmark finishes; a hard CPU bound requires an OS/sandbox limit. The input is
// an immutable string, so nothing is shared with the abandoned goroutine.
func convertMarkdownToHTML(thread *starlark.Thread, md goldmark.Markdown, text string, timeout float64) (string, error) {
	ctx, cancel := util.OpContext(thread, util.DurationFromSeconds(timeout))
	defer cancel()
	if err := ctx.Err(); err != nil {
		return "", errConvertTimeout
	}

	type result struct {
		out string
		err error
	}
	ch := make(chan result, 1) // buffered so an abandoned goroutine never blocks
	go func() {
		var out string
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					out, err = "", fmt.Errorf("failed to convert markdown to HTML: convert panic: %v", r)
				}
			}()
			var buf bytes.Buffer
			if cerr := md.Convert([]byte(text), &buf); cerr != nil {
				out, err = "", fmt.Errorf("failed to convert markdown to HTML: %v", cerr)
				return
			}
			out = buf.String()
		}()
		ch <- result{out, err}
	}()

	select {
	case res := <-ch:
		return res.out, res.err
	case <-ctx.Done():
		return "", errConvertTimeout
	}
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

		// Reject oversized input before handing it to goldmark.
		text := markdownText.GoString()
		if err := m.checkInputSize(text); err != nil {
			return none, err
		}

		// Convert markdown to HTML
		html, err := convertMarkdownToHTML(thread, md, text, m.maxConvertTime())
		if err != nil {
			return none, err
		}

		return starlark.String(html), nil
	})
}

// checkInputSize rejects input longer than the configured max_input_bytes
// (when positive) before it reaches goldmark.
func (m *Module) maxConvertTime() float64 {
	return m.ext.GetFloat(configKeyMaxConvertTime, defaultMaxConvertTime)
}

func (m *Module) checkInputSize(text string) error {
	if maxBytes := m.ext.GetInt(configKeyMaxInputBytes); maxBytes > 0 && len(text) > maxBytes {
		return fmt.Errorf("%s.convert: input exceeds max_input_bytes (%d)", ModuleName, maxBytes)
	}
	return nil
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

			// Reject oversized input before handing it to goldmark.
			text := markdownText.GoString()
			if err := m.checkInputSize(text); err != nil {
				return none, err
			}

			// Convert markdown to HTML
			html, err := convertMarkdownToHTML(thread, md, text, m.maxConvertTime())
			if err != nil {
				return none, err
			}

			return starlark.String(html), nil
		}

		return starlark.NewBuiltin("custom_converter", converter), nil
	})
}
