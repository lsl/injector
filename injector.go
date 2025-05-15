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

// RegisterResolver registers a function that resolves a type dynamically per request.
func RegisterResolver[T any](fn func(*http.Request) T) {
	var zero T
	t := reflect.TypeOf(zero)
	if _, exists := injectors[t]; exists {
		panic("injector already registered for type: " + t.String())
	}

	injectors[t] = func(r *http.Request) any {
		return fn(r)
	}
}

// RegisterStatic is a convenience helper to register static instances.
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

// Middleware wraps a function returning func(http.Handler) http.Handler and injects its dependencies.
func Middleware(fn any) func(http.Handler) http.Handler {
	v := reflect.ValueOf(fn)
	t := v.Type()

	if t.Kind() != reflect.Func || t.NumOut() != 1 {
		panic("injector: middleware must be a function returning one value")
	}

	if t.Out(0) != reflect.TypeOf((func(http.Handler) http.Handler)(nil)) {
		panic("injector: middleware must return func(http.Handler) http.Handler")
	}

	// Precompile argument resolvers
	resolvers := make([]func(*http.Request) reflect.Value, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		param := t.In(i)
		injectorFn, ok := injectors[param]
		if !ok {
			panic("no injector for middleware param: " + param.String())
		}
		resolvers[i] = func(r *http.Request) reflect.Value {
			return reflect.ValueOf(injectorFn(r))
		}
	}

	return func(next http.Handler) http.Handler {
		// Create dummy request to resolve dependencies
		dummyReq, _ := http.NewRequest("GET", "/", nil)
		args := make([]reflect.Value, len(resolvers))
		for i, resolver := range resolvers {
			args[i] = resolver(dummyReq)
		}
		return v.Call(args)[0].Interface().(func(http.Handler) http.Handler)(next)
	}
}

// Router is an http.Handler that supports dependency-injected handlers and middleware.
type Router struct {
	mux        *http.ServeMux
	middleware []func(http.Handler) http.Handler
}

// NewRouter creates a new injector-aware Router.
func NewRouter() *Router {
	return &Router{
		mux:        http.NewServeMux(),
		middleware: []func(http.Handler) http.Handler{},
	}
}

// Use appends a middleware to the Router.
func (r *Router) Use(mw any) {
	// Allow raw middleware or injector-aware middleware
	switch fn := mw.(type) {
	case func(http.Handler) http.Handler:
		r.middleware = append(r.middleware, fn)
	default:
		r.middleware = append(r.middleware, Middleware(fn))
	}
}

// HandleFunc registers a handler with injection support.
func (r *Router) HandleFunc(pattern string, handler any) {
	var h http.Handler = Inject(handler)
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}
	r.mux.Handle(pattern, h)
}

// Handle registers a handler or function with injection support.
func (r *Router) Handle(pattern string, h any) {
	var handler http.Handler

	switch v := h.(type) {
	case http.Handler:
		handler = v
	default:
		handler = Inject(v)
	}

	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	r.mux.Handle(pattern, handler)
}

// ServeHTTP dispatches the request to the appropriate handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
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
