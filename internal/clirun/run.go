// Package clirun runs a parsed lewkit x/cmd spec.
//
// App.Run does not walk anonymous / flatten command groups, so a generated
// `children` embed never reaches the selected leaf. This walker does.
package clirun

import (
	"context"
	"reflect"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lucasew/workspaced/internal/db"
	"github.com/lucasew/workspaced/pkg/logging"
)

// Run walks v (including flatten/anonymous structs) and calls Run(ctx) error
// on the selected command. A group with no Run returns cmd.ErrUsage.
func Run(ctx context.Context, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		rv = reflect.ValueOf(&v).Elem()
	}
	return runSelected(ctx, rv)
}

func runSelected(ctx context.Context, v reflect.Value) error {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return cmd.ErrUsage
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return cmd.ErrUsage
	}
	if child := selectedCommand(v); child.IsValid() {
		ctx, cleanup, err := injectDB(ctx, v)
		if err != nil {
			return err
		}
		defer cleanup()
		return runSelected(ctx, child)
	}
	if run := runMethod(v); run.IsValid() {
		return callRun(ctx, run)
	}
	return cmd.ErrUsage
}

type dbOpener interface {
	Open(context.Context) (*db.DB, error)
}

func injectDB(ctx context.Context, v reflect.Value) (context.Context, func(), error) {
	if _, ok := db.FromContext(ctx); ok {
		return ctx, func() {}, nil
	}
	opener, ok := v.Addr().Interface().(dbOpener)
	if !ok {
		return ctx, func() {}, nil
	}
	database, err := opener.Open(ctx)
	if err != nil {
		return ctx, nil, err
	}
	return db.WithDB(ctx, database), func() { logging.Close(ctx, database) }, nil
}

func selectedCommand(v reflect.Value) reflect.Value {
	t := v.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		fv := v.Field(i)
		if !fv.CanAddr() {
			continue
		}
		_, flatten := sf.Tag.Lookup("flatten")
		if (sf.Anonymous || flatten) && fv.Kind() == reflect.Struct {
			if child := selectedCommand(fv); child.IsValid() {
				return child
			}
			continue
		}
		if fv.Kind() != reflect.Pointer || fv.IsNil() || fv.Type().Elem().Kind() != reflect.Struct {
			continue
		}
		if isCommandField(fv) {
			return fv
		}
	}
	return reflect.Value{}
}

func isCommandField(fv reflect.Value) bool {
	elem := reflect.New(fv.Type().Elem())
	return !hasParse(elem) && !hasCount(elem)
}

func hasParse(ptr reflect.Value) bool {
	return methodSig(ptr, "Parse", reflect.TypeFor[string]())
}

func hasCount(ptr reflect.Value) bool {
	return methodSig(ptr, "Count", reflect.TypeFor[int]())
}

func methodSig(ptr reflect.Value, name string, in reflect.Type) bool {
	m := ptr.MethodByName(name)
	if !m.IsValid() {
		return false
	}
	t := m.Type()
	return t.NumIn() == 1 && t.In(0) == in && t.NumOut() == 1 && t.Out(0) == reflect.TypeFor[error]()
}

func runMethod(v reflect.Value) reflect.Value {
	m := v.Addr().MethodByName("Run")
	if !m.IsValid() {
		return reflect.Value{}
	}
	mt := m.Type()
	if mt.NumIn() != 1 || mt.In(0) != reflect.TypeFor[context.Context]() {
		return reflect.Value{}
	}
	if mt.NumOut() != 1 || mt.Out(0) != reflect.TypeFor[error]() {
		return reflect.Value{}
	}
	return m
}

func callRun(ctx context.Context, m reflect.Value) error {
	out := m.Call([]reflect.Value{reflect.ValueOf(ctx)})[0].Interface()
	if err, ok := out.(error); ok {
		return err
	}
	return nil
}
