package transformer_test

import (
	"testing"

	"github.com/stephen/cssc/internal/parser"
	"github.com/stephen/cssc/internal/sources"
	"github.com/stephen/cssc/internal/transformer"
	"github.com/stephen/cssc/transforms"
	"github.com/stretchr/testify/assert"
)

func TestCustomMedia(t *testing.T) {
	assert.Equal(t, "@media (max-width:30em){.a{color:green}}@media (max-width:30em) and (script){.c{color:red}}", Transform(t, func(o *transformer.Options) {
		o.CustomMediaQueries = transforms.CustomMediaQueriesTransform
	}, `
	@custom-media --narrow-window (max-width: 30em);

	@media (--narrow-window) {
		.a { color: green; }
	}

	@media (--narrow-window) and (script) {
		.c { color: red; }
	}`))

}

func TestCustomMedia_Unsupported(t *testing.T) {
	_, err := parser.Parse(&sources.Source{
		Path: "main.css",
		Content: `
	@custom-media --narrow-window (max-width: 30em), print;

	@media (--narrow-window) {
		.a { color: green; }
	}

	@media (--narrow-window) and (script) {
		.c { color: red; }
	}`,
	})
	assert.EqualError(t, err, "main.css:2:55\n@custom-media rule requires a single media query argument:\n\t  @custom-media --narrow-window (max-width: 30em), print;\n\t                                                        ~")
}

func TestCustomMedia_Passthrough(t *testing.T) {
	assert.Equal(t, "@media (--narrow-window){.a{color:green}}@media (--narrow-window) and (script){.c{color:red}}", Transform(t, nil, `
	@custom-media --narrow-window (max-width: 30em);

	@media (--narrow-window) {
		.a { color: green; }
	}

	@media (--narrow-window) and (script) {
		.c { color: red; }
	}`))
}
