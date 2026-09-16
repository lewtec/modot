package main

import (
	"reflect"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
)

func TestSelectedHasRun(t *testing.T) {
	t.Parallel()
	cases := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{"home"}, false},
		{[]string{"home", "apply"}, true},
		{[]string{"codebase", "lint"}, true},
		{[]string{"self-install"}, true},
	}
	for _, tc := range cases {
		app := cmd.ParseOK[cmd.App[cli]](t, tc.args...)
		assert.Equal(t, tc.want, selectedHasRun(reflect.ValueOf(&app.Args).Elem()), "%q", tc.args)
	}
}
