// Package routegroup provides a way to group routes and apply middleware to them. Works with the standard library's http.ServeMux.
package routegroup

import "net/http"

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
	b.mux.HandleFunc(b.basePath+path, wrap(handler, b.middlewares...).ServeHTTP)
}

// Set allows for configuring the Group.
func (b *Bundle) Set(configureFn func(*Bundle)) {
	configureFn(b)
}
