// Package injector provides dependency injection for HTTP handlers.
package injector

import (
	"context"
	"net/http"
	"reflect"
)

// Injector is a function that extracts a value from the request.
type Injector func(*http.Request) any

var injectors = map[reflect.Type]Injector{}

// RegisterResolver allows services to be registered for injection.
func RegisterResolver[T any](fn func(*http.Request) T) {
	var zero T
	// Two options here:
	// 1. Strip pointers from type for matching
	// t := reflect.TypeOf(zero)
	// if t.Kind() == reflect.Ptr {
	// 	t = t.Elem()
	// }
	// 2. Match types exactly
	t := reflect.TypeOf(zero)
	// This seems to behave better for the example but I think I'll
	// need to test it out in a bigger app.

	if _, exists := injectors[t]; exists {
		panic("injector already registered for type: " + t.String())
	}

	injectors[t] = func(r *http.Request) any {
		return fn(r)
	}
}

// Register is a convenience helper to register static instances.
func RegisterStatic[T any](val T) {
	RegisterResolver(func(_ *http.Request) T {
		return val
	})
}

// Inject wraps a function and builds an http.HandlerFunc with precompiled injection.
func Inject(fn any) http.HandlerFunc {
	v := reflect.ValueOf(fn)
	t := v.Type()

	if t.Kind() != reflect.Func {
		panic("injected: expected a function")
	}

	// Precompile resolvers at registration time
	resolvers := make([]func(http.ResponseWriter, *http.Request) reflect.Value, t.NumIn())

	for i := 0; i < t.NumIn(); i++ {
		param := t.In(i)

		switch param {
		case reflect.TypeOf((*http.Request)(nil)):
			resolvers[i] = func(_ http.ResponseWriter, r *http.Request) reflect.Value {
				return reflect.ValueOf(r)
			}
		case reflect.TypeOf((*http.ResponseWriter)(nil)).Elem():
			resolvers[i] = func(w http.ResponseWriter, _ *http.Request) reflect.Value {
				return reflect.ValueOf(w)
			}
		default:
			injector, ok := injectors[param]
			if !ok {
				panic("no injector for type: " + param.String())
			}
			resolvers[i] = func(_ http.ResponseWriter, r *http.Request) reflect.Value {
				return reflect.ValueOf(injector(r))
			}
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		args := make([]reflect.Value, len(resolvers))
		for i, resolver := range resolvers {
			args[i] = resolver(w, r)
		}
		v.Call(args)
	}
}

// Context helpers.
type ctxKey string

func contextKey[T any]() ctxKey {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return ctxKey(t.String())
}

// WithValue adds a value to the context with a type-based key.
func WithValue[T any](ctx context.Context, val T) context.Context {
	return context.WithValue(ctx, contextKey[T](), val)
}

// Use retrieves a value from the context, panicking if not found.
func Use[T any](ctx context.Context) T {
	v := ctx.Value(contextKey[T]())
	if v == nil {
		panic("missing context value for type: " + reflect.TypeOf((*T)(nil)).Elem().String())
	}
	return v.(T)
}

// Try attempts to retrieve a value from the context without panicking.
func Try[T any](ctx context.Context) (T, bool) {
	v := ctx.Value(contextKey[T]())
	if v == nil {
		var zero T
		return zero, false
	}
	return v.(T), true
}
