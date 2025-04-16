// Package markdown provides a Starlark module for converting markdown to HTML.
package markdown

import (
	"bytes"
	"fmt"

	"github.com/1set/starlet"
	"github.com/starpkg/base"
	"github.com/yuin/goldmark"
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
	ext    *base.ConfigurableModuleExt
}

// NewModule creates a new instance of Module with default configurations.
func NewModule() *Module {
	cm := base.NewConfigurableModule()
	return &Module{
		cfgMod: cm,
		ext:    cm.Extend(),
	}
}

// LoadModule returns the Starlark module loader with markdown-specific functions.
func (m *Module) LoadModule() starlet.ModuleLoader {
	// Module functions
	funcs := starlark.StringDict{
		"convert":      m.genConvertFunc(),
		"with_options": m.genWithOptionsFunc(),
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

// genConvertFunc generates the Starlark callable function to convert markdown to HTML.
func (m *Module) genConvertFunc() starlark.Callable {
	return starlark.NewBuiltin(ModuleName+".convert", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var (
			markdownText     starlark.String
			unsafe           starlark.Value = starlark.Bool(true)
			enableHeadingID  starlark.Value = starlark.Bool(true)
			enableLinkify    starlark.Value = starlark.Bool(true)
			enableTable      starlark.Value = starlark.Bool(true)
			enableTaskList   starlark.Value = starlark.Bool(true)
			enableStrike     starlark.Value = starlark.Bool(true)
			enableFootnote   starlark.Value = starlark.Bool(false)
			enableDefinition starlark.Value = starlark.Bool(false)
			enableTypograph  starlark.Value = starlark.Bool(false)
		)

		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"text", &markdownText,
			"unsafe?", &unsafe,
			"heading_id?", &enableHeadingID,
			"linkify?", &enableLinkify,
			"table?", &enableTable,
			"task_list?", &enableTaskList,
			"strikethrough?", &enableStrike,
			"footnote?", &enableFootnote,
			"definition?", &enableDefinition,
			"typograph?", &enableTypograph,
		); err != nil {
			return none, err
		}

		// Configure markdown renderer
		mdOptions := []goldmark.Option{}

		// Add renderer options
		rendererOptions := []renderer.Option{}
		if isTruthy(unsafe) {
			rendererOptions = append(rendererOptions, html.WithUnsafe())
		}
		if len(rendererOptions) > 0 {
			mdOptions = append(mdOptions, goldmark.WithRendererOptions(rendererOptions...))
		}

		// Add parser options
		parserOptions := []parser.Option{}
		if isTruthy(enableHeadingID) {
			parserOptions = append(parserOptions, parser.WithAutoHeadingID())
		}
		if len(parserOptions) > 0 {
			mdOptions = append(mdOptions, goldmark.WithParserOptions(parserOptions...))
		}

		// Add extensions
		extensions := []goldmark.Extender{}
		if isTruthy(enableTable) {
			extensions = append(extensions, extension.Table)
		}
		if isTruthy(enableLinkify) {
			extensions = append(extensions, extension.Linkify)
		}
		if isTruthy(enableTaskList) {
			extensions = append(extensions, extension.TaskList)
		}
		if isTruthy(enableStrike) {
			extensions = append(extensions, extension.Strikethrough)
		}
		if isTruthy(enableFootnote) {
			extensions = append(extensions, extension.Footnote)
		}
		if isTruthy(enableDefinition) {
			extensions = append(extensions, extension.DefinitionList)
		}
		if isTruthy(enableTypograph) {
			extensions = append(extensions, extension.Typographer)
		}

		if len(extensions) > 0 {
			mdOptions = append(mdOptions, goldmark.WithExtensions(extensions...))
		}

		// Create markdown converter
		md := goldmark.New(mdOptions...)

		// Convert markdown to HTML
		var buf bytes.Buffer
		if err := md.Convert([]byte(markdownText.GoString()), &buf); err != nil {
			return none, fmt.Errorf("failed to convert markdown to HTML: %v", err)
		}

		return starlark.String(buf.String()), nil
	})
}

// genWithOptionsFunc generates the Starlark callable function to create a configured markdown converter.
func (m *Module) genWithOptionsFunc() starlark.Callable {
	return starlark.NewBuiltin(ModuleName+".with_options", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var (
			unsafe           starlark.Value = starlark.Bool(true)
			enableHeadingID  starlark.Value = starlark.Bool(true)
			enableLinkify    starlark.Value = starlark.Bool(true)
			enableTable      starlark.Value = starlark.Bool(true)
			enableTaskList   starlark.Value = starlark.Bool(true)
			enableStrike     starlark.Value = starlark.Bool(true)
			enableFootnote   starlark.Value = starlark.Bool(false)
			enableDefinition starlark.Value = starlark.Bool(false)
			enableTypograph  starlark.Value = starlark.Bool(false)
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
		); err != nil {
			return none, err
		}

		// Create converter function
		converter := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			var markdownText starlark.String
			if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &markdownText); err != nil {
				return none, err
			}

			// Configure markdown renderer
			mdOptions := []goldmark.Option{}

			// Add renderer options
			rendererOptions := []renderer.Option{}
			if isTruthy(unsafe) {
				rendererOptions = append(rendererOptions, html.WithUnsafe())
			}
			if len(rendererOptions) > 0 {
				mdOptions = append(mdOptions, goldmark.WithRendererOptions(rendererOptions...))
			}

			// Add parser options
			parserOptions := []parser.Option{}
			if isTruthy(enableHeadingID) {
				parserOptions = append(parserOptions, parser.WithAutoHeadingID())
			}
			if len(parserOptions) > 0 {
				mdOptions = append(mdOptions, goldmark.WithParserOptions(parserOptions...))
			}

			// Add extensions
			extensions := []goldmark.Extender{}
			if isTruthy(enableTable) {
				extensions = append(extensions, extension.Table)
			}
			if isTruthy(enableLinkify) {
				extensions = append(extensions, extension.Linkify)
			}
			if isTruthy(enableTaskList) {
				extensions = append(extensions, extension.TaskList)
			}
			if isTruthy(enableStrike) {
				extensions = append(extensions, extension.Strikethrough)
			}
			if isTruthy(enableFootnote) {
				extensions = append(extensions, extension.Footnote)
			}
			if isTruthy(enableDefinition) {
				extensions = append(extensions, extension.DefinitionList)
			}
			if isTruthy(enableTypograph) {
				extensions = append(extensions, extension.Typographer)
			}

			if len(extensions) > 0 {
				mdOptions = append(mdOptions, goldmark.WithExtensions(extensions...))
			}

			// Create markdown converter
			md := goldmark.New(mdOptions...)

			// Convert markdown to HTML
			var buf bytes.Buffer
			if err := md.Convert([]byte(markdownText.GoString()), &buf); err != nil {
				return none, fmt.Errorf("failed to convert markdown to HTML: %v", err)
			}

			return starlark.String(buf.String()), nil
		}

		return starlark.NewBuiltin("convert_with_options", converter), nil
	})
}
