package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple", input: "hello world", want: "Hello World"},
		{name: "hyphen", input: "foo-bar", want: "Foo Bar"},
		{name: "underscore", input: "foo_bar", want: "Foo Bar"},
		{name: "mixed separators", input: "hello-world_test", want: "Hello World Test"},
		{name: "already title case", input: "Hello World", want: "Hello World"},
		{name: "all upper", input: "HELLO WORLD", want: "Hello World"},
		{name: "single word", input: "hello", want: "Hello"},
		{name: "empty", input: "", want: ""},
		{name: "extra spaces", input: "  foo   bar  ", want: "Foo Bar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToTitleCase(tt.input)
			assert.Equal(t, tt.want, got, "ToTitleCase(%q)", tt.input)
		})
	}
}
