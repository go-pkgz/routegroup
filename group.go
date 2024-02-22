// Package routegroup provides a way to group routes and applies middleware to them.
// Works with the standard library's http.ServeMux.
package routegroup

import (
	"net/http"
	"regexp"
)

// Bundle represents a group of routes with associated middleware.
type Bundle struct {
	mux         *http.ServeMux
	basePath    string
	middlewares []func(http.Handler) http.Handler
}

// New creates a new Group.
func New(mux *http.ServeMux) *Bundle {
	return &Bundle{
		mux: mux,
	}
}

// Mount creates a new group with a specified base path.
func Mount(mux *http.ServeMux, basePath string) *Bundle {
	return &Bundle{
		mux:      mux,
		basePath: basePath,
	}
}

// ServeHTTP implements the http.Handler interface
func (b *Bundle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mux.ServeHTTP(w, r)
}

// Group creates a new group with the same middleware stack as the original on top of the existing bundle.
func (b *Bundle) Group() *Bundle {
	// copy the middlewares to avoid modifying the original
	middlewares := make([]func(http.Handler) http.Handler, len(b.middlewares))
	copy(middlewares, b.middlewares)
	return &Bundle{
		mux:         b.mux,
		basePath:    b.basePath,
		middlewares: middlewares,
	}
}

// Mount creates a new group with a specified base path on top of the existing bundle.
func (b *Bundle) Mount(basePath string) *Bundle {
	// copy the middlewares to avoid modifying the original
	middlewares := make([]func(http.Handler) http.Handler, len(b.middlewares))
	copy(middlewares, b.middlewares)
	return &Bundle{
		mux:         b.mux,
		basePath:    basePath,
		middlewares: middlewares,
	}
}

// Use adds middleware(s) to the Group.
func (b *Bundle) Use(middleware func(http.Handler) http.Handler, more ...func(http.Handler) http.Handler) {
	b.middlewares = append(b.middlewares, middleware)
	b.middlewares = append(b.middlewares, more...)
}

// With adds new middleware(s) to the Group and returns a new Group with the updated middleware stack.
func (b *Bundle) With(middleware func(http.Handler) http.Handler, more ...func(http.Handler) http.Handler) *Bundle {
	newMiddlewares := make([]func(http.Handler) http.Handler, len(b.middlewares)+len(more))
	copy(newMiddlewares, b.middlewares)
	newMiddlewares = append(newMiddlewares, middleware)
	newMiddlewares = append(newMiddlewares, more...)

	return &Bundle{
		mux:         b.mux,
		basePath:    b.basePath,
		middlewares: newMiddlewares,
	}
}

// Matches non-space characters, spaces, then anything, i.e. "GET /path/to/resource"
var reGo122 = regexp.MustCompile(`^(\S*)\s+(.*)$`)

// Handle adds a new route to the Group's mux, applying all middlewares to the handler.
func (b *Bundle) Handle(path string, handler http.HandlerFunc) {
	wrap := func(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
		if len(mws) == 0 {
			return h
		}
		res := h
		for i := len(mws) - 1; i >= 0; i-- {
			res = mws[i](res)
		}
		return res
	}

	if b.basePath != "" {
		matches := reGo122.FindStringSubmatch(path)
		if len(matches) > 2 { // path in the form "GET /path/to/resource"
			path = matches[1] + " " + b.basePath + matches[2]
		} else { // path is just "/path/to/resource"
			path = b.basePath + path
		}
	}

	b.mux.HandleFunc(path, wrap(handler, b.middlewares...).ServeHTTP)
}

// Route allows for configuring the Group inside the configureFn function.
func (b *Bundle) Route(configureFn func(*Bundle)) {
	configureFn(b)
}
