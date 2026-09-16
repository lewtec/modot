package main

import (
	"context"
	"reflect"
)

type described interface {
	Description() string
}

func selectedHasRun(v reflect.Value) bool {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return false
	}
	t := v.Type()
	for i := range t.NumField() {
		fv := v.Field(i)
		if !fv.CanInterface() || !isCommandPtr(fv) {
			continue
		}
		return selectedHasRun(fv)
	}
	return hasRunMethod(v)
}

func isCommandPtr(fv reflect.Value) bool {
	if fv.Kind() != reflect.Pointer || fv.IsNil() {
		return false
	}
	_, ok := fv.Interface().(described)
	return ok
}

func hasRunMethod(v reflect.Value) bool {
	if !v.CanAddr() {
		return false
	}
	m := v.Addr().MethodByName("Run")
	if !m.IsValid() {
		return false
	}
	mt := m.Type()
	return mt.NumIn() == 1 && mt.In(0) == reflect.TypeFor[context.Context]() &&
		mt.NumOut() == 1 && mt.Out(0) == reflect.TypeFor[error]()
}
