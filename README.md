## routegroup [![Build Status](https://github.com/go-pkgz/routegroup/workflows/build/badge.svg)](https://github.com/go-pkgz/routegroup/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/routegroup)](https://goreportcard.com/report/github.com/go-pkgz/routegroup) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/routegroup/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/routegroup?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/routegroup?status.svg)](https://godoc.org/github.com/go-pkgz/routegroup)


`routegroup` is a tiny Go package providing a lightweight wrapper for efficient route grouping and middleware integration with the standard `http.ServeMux`.

## Features

- Simple and intuitive API for route grouping and route mounting.
- Easy middleware integration for individual routes or groups of routes.
- Seamless integration with Go's standard `http.ServeMux`.
- Fully compatible with the `http.Handler` interface and can be used as a drop-in replacement for `http.ServeMux`.

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
	apiGroup.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, API!"))
	})
	
	// add another group with its own set of middlewares
	protectedGroup := apiGroup.Group()
	protectedGroup.Use(authMiddleware)
	protectedGroup.HandleFunc("GET /protected", func(w http.ResponseWriter, r *http.Request) {
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
group.Route(func(b *routegroup.Bundle) {
    b.Use(loggingMiddleware, corsMiddleware)
    b.Handle("GET /hello", helloHandler)
    b.Handle("GET /bye", byeHandler)
})
http.ListenAndServe(":8080", mux)
```

### Using derived groups

In some instances, it's practical to create an initial group that includes a set of middlewares, and then derive all other groups from it. This approach guarantees that every group incorporates a common set of middlewares as a foundation, allowing each to add its specific middlewares. To facilitate this scenario, `routegroup` offers both `Bundle.Group` and `Bundle.Mount` methods, and it also implements the `http.Handler` interface. The following example illustrates how to use derived groups:

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
// using `Route` method to show the alternative usage.
// this group will inherit the middlewares from the base group as well
mux.Mount("/admin").Route(func(b *routegroup.Bundle) {
    b.Use(adminMiddleware)
    b.Handle("POST /do", doHandler)
})

// start the server, passing the wrapped mux as the handler
http.ListenAndServe(":8080", mux)
```
### Wrap Function

Sometimes route's group is not necessary, and all you need is to apply middleware(s) directly to a single route. In this case, `routegroup` provides a `Wrap` function that can be used to wrap a single `http.Handler` with one or more middlewares. Here's an example:

```go
mux := http.NewServeMux()
mux.HandleFunc("/hello", routegroup.Wrap(helloHandler, loggingMiddleware, corsMiddleware))
http.ListenAndServe(":8080", mux)
```

## Real-world example

Here's an example of how `routegroup` can be used in a real-world application. The following code snippet is taken from a web service that provides a set of routes for user authentication, session management, and user management. The service also serves static files from the "assets/static" embedded file system.

```go

// Routes returns http.Handler that handles all the routes for the Service.
// It also serves static files from the "assets/static" directory.
// The rootURL option sets prefix for the routes.
func (s *Service) Routes() http.Handler {
	router := routegroup.Mount(http.NewServeMux(), s.rootURL) // make a bundle with the rootURL base path
	// add common middlewares
	router.Use(rest.Maybe(handlers.CompressHandler, func(*http.Request) bool { return !s.skipGZ }))
	router.Use(rest.Throttle(s.limitActiveReqs))
	router.Use(s.middleware.securityHeaders(s.skipSecurityHeaders))

	// prepare csrf middleware
	csrfMiddleware := s.middleware.csrf(s.skipCSRFCheck)

	// add open routes
	router.HandleFunc("GET /login", s.loginPageHandler)
	router.HandleFunc("POST /login", s.loginCheckHandler)
	router.HandleFunc("GET /logout", s.logoutHandler)

	// add routes with auth middleware
	router.Group().Route(func(auth *routegroup.Bundle) {
		auth.Use(s.middleware.Auth())
		auth.HandleFunc("GET /update", s.pwdUpdateHandler)
		auth.With(csrfMiddleware).HandleFunc("PUT /update", s.pwdUpdateHandler)
	})

	// add admin routes
	router.Mount("/admin").Route(func(admin *routegroup.Bundle) {
		admin.Use(s.middleware.Auth("admin"))
		admin.Use(s.middleware.AdminOnly)
		admin.HandleFunc("GET /", s.admin.renderHandler)
		admin.With(csrfMiddleware).Route(func(csrf *routegroup.Bundle) {
			csrf.HandleFunc("DELETE /sessions", s.admin.deleteSessionsHandler)
			csrf.HandleFunc("POST /user", s.admin.addUserHandler)
			csrf.HandleFunc("DELETE /user", s.admin.deleteUserHandler)
		})
	})

	router.HandleFunc("GET /static/*", s.fileServerHandlerFunc()) // serve static files
	return router
}

// fileServerHandlerFunc returns http.HandlerFunc that serves static files from the "assets/static" directory.
// prefix is set by the rootURL option.
func (s *Service) fileServerHandlerFunc() http.HandlerFunc {
    staticFS, err := fs.Sub(assets, "assets/static") // error is always nil
    if err != nil {
        panic(err) // should never happen we load from embedded FS
    }
    return func(w http.ResponseWriter, r *http.Request) {
        webFS := http.StripPrefix(s.rootURL+"/static/", http.FileServer(http.FS(staticFS)))
        webFS.ServeHTTP(w, r)
    }
}
```

## Contributing

Contributions to `routegroup` are welcome! Please submit a pull request or open an issue for any bugs or feature requests.

## License

`routegroup` is available under the MIT license. See the [LICENSE](https://github.com/go-pkgz/routegroup/blob/master/LICENSE) file for more info.
