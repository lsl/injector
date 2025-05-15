# Injector

Injector is a Go library that injects values and services into your HTTP handler functions automagically.

The typical approach to dependency injection in Go web applications is to pass dependencies into handlers from your main function. This works ok for applications where the dependencies are defined in the same place as the routes but for webservers that define routes in non central locations you end up with a prop drilling problem and tedious glue code to add whenever new dependencies are added.

This problem is what injector solves, it centralizes dependency registration for your entire application and allows the handlers to express the dependencies in their call signatures.

## Features

- Type-safe dependency injection for HTTP handlers
- Automatic resolution of dependencies at runtime
- Zero cost request-time lookups for injected values
- Easy registration of services
- Context-based value storage and retrieval
- Built-in middleware support
- Optional router integration
- No external dependencies

## Installation

```bash
go get github.com/lsl/injector
```

## Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/lsl/injector"
)

// Create a handler that requires dependencies
func UserHandler(w http.ResponseWriter, r *http.Request, logger *log.Logger) {
    logger.Println("Processing user request")
    fmt.Fprintf(w, "Hello, user!")
}

func main() {
    // Create a logger
    logger := log.New(os.Stdout, "[APP] ", log.LstdFlags)

    // Register the logger for injection
    injector.RegisterStatic(logger)

    // Create a handler with injection
    http.HandleFunc("/user", injector.Inject(UserHandler))

    // Start the server
    http.ListenAndServe(":8080", nil)
}
```

## Advanced Usage

### Custom Dependency Resolution

You can register custom resolvers for more complex dependency injection:

```go
// Locale represents a language code
type Locale string

// Register a resolver that extracts the current locale from the request context
injector.RegisterResolver(func(r *http.Request) Locale {
    locale, ok := injector.Try[Locale](r.Context())
    if !ok {
        // Default to Accept-Language header or fallback to en
        if acceptLang := r.Header.Get("Accept-Language"); acceptLang != "" {
            return Locale(acceptLang[:2]) // Get first two chars of Accept-Language
        }
        return "en"
    }
    return locale
})
```

### Context Values

You can store and retrieve values from the request context:

```go
// Store a value in context
ctx := injector.WithValue(r.Context(), myValue)
r = r.WithContext(ctx)

// Later, retrieve the value
value := injector.Use[MyType](r.Context())
```

### Middleware Support

Injector supports middleware with dependency injection:

```go
// Create middleware that requires dependencies
func AuthMiddleware(userRepo *UserRepo, logger *Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Middleware logic with injected dependencies
            // ...
            next.ServeHTTP(w, r)
        })
    }
}

// Apply middleware
handler := injector.Middleware(AuthMiddleware)(yourHandler)
```

### Built-in Router

Injector provides a ServeMux based router that automatically handles injection for both routes and middleware. This is the typical use case for

```go
// Create a router
router := injector.NewRouter()

// Apply middleware with automatic injection
router.Use(AuthMiddleware)

// Register handlers (no need for explicit Inject calls)
router.HandleFunc("/", HomeHandler)
router.HandleFunc("/users", UserHandler)

// Start the server
http.ListenAndServe(":8080", router)
```

## Examples

Check the `examples` directory for more comprehensive examples:

- `injector-only` - Basic usage with standard Go HTTP
- `with-router` - Using the built-in router integration
