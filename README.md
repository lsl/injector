# Injector

Injector is a Go library that allows you to inject values and services into your HTTP handler functions.

## Features

- Type-safe dependency injection for HTTP handlers
- Automatic resolution of dependencies at runtime
- Zero cost request-time lookups for injected values
- Easy registration of services
- Context-based value storage and retrieval
- Built-in middleware support
- Optional router integration
- Zero external dependencies

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
    injector.Register(logger)

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
// Register a resolver that extracts a user from the request context
injector.RegisterResolver(func(r *http.Request) *User {
    user, ok := injector.Try[*User](r.Context())
    if !ok {
        panic("No user found in request context")
    }
    return user
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

Injector provides an optional router that automatically handles injection for both routes and middleware:

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
