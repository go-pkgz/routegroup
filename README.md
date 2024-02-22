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

For routes under a specific path prefix `Mount` method can be used to create a nested group:

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
	apiGroup := routegroup.Mount(mux, "/api")

	// add middleware
	apiGroup.Use(loggingMiddleware, corsMiddleware)

	// route handling
	apiGroup.Handle("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, API!"))
	})
	
	// add another group with its own set of middlewares
	protectedGroup := apiGroup.Group()
	protectedGroup.Use(authMiddleware)
	protectedGroup.Handle("GET /protected", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Protected API!"))
    })

    http.ListenAndServe(":8080", mux)
}
```

**Applying Middleware to Specific Routes**

You can also apply middleware to specific routes inside the group without modifying the group's middleware stack:

```go
apiGroup.With(corsMiddleware, helloHandler).Handle("GET /hello",helloHandler)
```

**Alternative Usage with `Route`**

You can also use the `Route` method to add routes and middleware in a single function call:

```go
mux := http.NewServeMux()
group := routegroup.New(mux)
group.Route(b func(*routegroup.Bundle) {
    b.Use(loggingMiddleware, corsMiddleware)
    b.Handle("GET /hello", helloHandler)
    b.Handle("GET /bye", byeHandler)
})
http.ListenAndServe(":8080", mux)
```

### Using derived groups

In some instances, it's practical to create an initial group that includes a set of middlewares, and then derive all other groups from it. This approach guarantees that every group incorporates a common set of middlewares as a foundation, allowing each to add its specific middlewares. To facilitate this scenario, `routegrou`p offers both `Bundle.Group` and `Bundle.Mount` methods, and it also implements the `http.Handler` interface. The following example illustrates how to utilize derived groups:

```go
// create a new bundle with a base set of middlewares
// note: the bundle is also http.Handler and can be passed to http.ListenAndServe
mux := routegroup.New(http.NewServeMux()) 
mux.Use(loggingMiddleware, corsMiddleware)

// add a new group with its own set of middlewares
// this group will inherit the middlewares from the base group
apiGroup := mux.Group()
apiGroup.Use(apiMiddleware)
apiGroup.Handle("GET /hello", helloHandler)
apiGroup.Handle("GET /bye", byeHandler)


// mount another group for the /admin path with its own set of middlewares, 
// using `Set` method to show the alternative usage.
// this group will inherit the middlewares from the base group as well
mux.Mount("/admin").Route(func(b *routegroup.Bundle) {
    b.Use(adminMiddleware)
    b.Handle("POST /do", doHandler)
})

// start the server, passing the wrapped mux as the handler
http.ListenAndServe(":8080", mux)
```

## Contributing

Contributions to `routegroup` are welcome! Please submit a pull request or open an issue for any bugs or feature requests.

## License

`routegroup` is available under the MIT license. See the [LICENSE](https://github.com/go-pkgz/routegroup/blob/master/LICENSE) file for more info.