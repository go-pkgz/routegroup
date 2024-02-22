## routegroup [![Build Status](https://github.com/go-pkgz/routegroup/workflows/build/badge.svg)](https://github.com/go-pkgz/routegroup/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/routegroup)](https://goreportcard.com/report/github.com/go-pkgz/routegroup) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/routegroup/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/routegroup?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/routegroup?status.svg)](https://godoc.org/github.com/go-pkgz/routegroup)


`routegroup` is a tiny Go package providing a lightweight wrapper for efficient route grouping and middleware integration with the standard `http.ServeMux`.

## Features

- Simple and intuitive API for route grouping.
- Easy middleware integration for individual routes or groups of routes.
- Seamless integration with Go's standard `http.ServeMux`.

## Install and update

`go get -u github.com/go-pkgz/routegroup`

## Usage

*Creating a New Route Group*

To start, create a new route group without a base path:

```go
func main() {
    mux := http.NewServeMux()
    group := routegroup.New(mux)
}
```

*Adding Routes with Middleware*

Add routes to your group, optionally with middleware:

```go
    group.Use(loggingMiddleware, corsMiddleware)
    group.Handle("/hello", helloHandler)
    group.Handle("/bye", byeHandler)
```
*Creating a Nested Route Group*

For routes under a specific path prefix:

```go
    apiGroup := routegroup.Mount(mux, "/api")
    apiGroup.Use(loggingMiddleware, corsMiddleware)
    apiGroup.Handle("/v1", apiV1Handler)
    apiGroup.Handle("/v2", apiV2Handler)
```

*Complete Example*

Here's a complete example demonstrating route grouping and middleware usage:

```go
package main

import (
	"fmt"
	"net/http"
	
	"github.com/go-pkgz/routegroup"
)

func main() {
	mux := http.NewServeMux()
	apiGroup := routegroup.WithBasePath(mux, "/api")

	// add middleware
	apiGroup.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("Request received")
			next.ServeHTTP(w, r)
		})
	})

	// Route handling
	apiGroup.Handle("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, API!"))
	})

	http.ListenAndServe(":8080", mux)
}
```

*Applying Middleware to Specific Routes*

You can also apply middleware to specific routes:

```go
    apiGroup.With(corsMiddleware, helloHandler).Handle("GET /hello", loggingMiddleware)
```

*Alternative Usage with `Set`*

You can also use the `Set` method to add routes and middleware:

```go
    mux := http.NewServeMux()
	group := routegroup.New(mux)
	group.Set(b func(*routegroup.Bundle) {
		b.Use(loggingMiddleware, corsMiddleware)
		b.Handle("GET /hello", helloHandler)
		b.Handle("GET /bye", byeHandler)
    })
    http.ListenAndServe(":8080", mux)
```