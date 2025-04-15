# Injector

Injector is a Go library that allows you to inject values and services into your HTTP handler functions.

## Features

- Type-safe dependency injection for HTTP handlers
- Automatic resolution of dependencies at runtime
- Zero cost request-time lookups for injected values
- Easy registration of services
- Context-based value storage and retrieval
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
    http.HandleFunc("/user", injector.Injected(UserHandler))

    // Start the server
    http.ListenAndServe(":8080", nil)
}
```

## Context Values

You can also store and retrieve values from the request context:

```go
// Store a value in context
ctx := injector.WithValue(r.Context(), myValue)
r = r.WithContext(ctx)

// Later, retrieve the value
value := injector.Use[MyType](r.Context())
```

## Examples

Check the `examples` directory for more comprehensive examples:

- Basic usage
- Web application with multiple services
- Middleware integration
