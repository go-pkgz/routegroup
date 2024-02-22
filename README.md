## routegroup [![Build Status](https://github.com/go-pkgz/routegroup/workflows/build/badge.svg)](https://github.com/go-pkgz/routegroup/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/routegroup)](https://goreportcard.com/report/github.com/go-pkgz/routegroup) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/routegroup/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/routegroup?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/routegroup?status.svg)](https://godoc.org/github.com/go-pkgz/routegroup)


`routegroup` is a tiny Go package providing a lightweight wrapper for efficient route grouping and middleware integration with the standard `http.ServeMux`.

## Features

- Simple and intuitive API for route grouping and route mounting.
- Easy middleware integration for individual routes or groups of routes.
- Seamless integration with Go's standard `http.ServeMux`.

## Why One More Router?

Despite what the section title might suggest, `routegroup` is not another router. With Go's 1.22 release, the standard library's [routing enhancements](https://go.dev/blog/routing-enhancements) have made it possible to implement sophisticated routing logic without the need for external libraries. These enhancements provide the foundation for building fully functional HTTP servers directly with the tools Go offers.

However, while the standard `http.ServeMux` has become more powerful, it still lacks some conveniences, particularly in route grouping and middleware management. This is where `routegroup` steps in. Rather than reinventing the wheel, `routegroup` aims to supplement the existing routing capabilities by providing a minimalist abstraction layer for efficiently grouping routes and applying middleware to these groups.


## Install and update

`go get -u github.com/go-pkgz/routegroup`

## Usage

**Creating a New Route Group**

To start, create a new route group without a base path:

```go
func main() {
    mux := http.NewServeMux()
    group := routegroup.New(mux)
}
```

**Adding Routes with Middleware**

Add routes to your group, optionally with middleware:

```go
    group.Use(loggingMiddleware, corsMiddleware)
    group.Handle("/hello", helloHandler)
    group.Handle("/bye", byeHandler)
```
**Creating a Nested Route Group**

For routes under a specific path prefix:

```go
    apiGroup := routegroup.Mount(mux, "/api")
    apiGroup.Use(loggingMiddleware, corsMiddleware)
    apiGroup.Handle("/v1", apiV1Handler)
    apiGroup.Handle("/v2", apiV2Handler)
```

**Complete Example**

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

**Applying Middleware to Specific Routes**

You can also apply middleware to specific routes:

```go
    apiGroup.With(corsMiddleware, helloHandler).Handle("GET /hello", helloHandler)
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

## Contributing

Contributions to `routegroup` are welcome! Please submit a pull request or open an issue for any bugs or feature requests.

## License

`routegroup` is available under the MIT license. See the [LICENSE](https://github.com/go-pkgz/routegroup/blob/master/LICENSE) file for more info.