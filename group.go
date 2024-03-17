// Package routegroup provides a way to group routes and applies middleware to them.
// Works with the standard library's http.ServeMux.
package routegroup

import (
	"net/http"
	"regexp"
	"sync"
)

// Bundle represents a group of routes with associated middleware.
type Bundle struct {
	mux            *http.ServeMux                    // the underlying mux to register the routes to
	basePath       string                            // base path for the group
	middlewares    []func(http.Handler) http.Handler // middlewares stack
	rootRegistered struct {
		once                       sync.Once // used to register a not found handler for the root path if no / route is registered
		set                        bool      // true if the root path is registered in the mux
		disableRootNotFoundHandler bool      // if true, the not found handler for the root path is not registered automatically
	}
}

// New creates a new Group.
func New(mux *http.ServeMux) *Bundle {
	b := &Bundle{mux: mux}
	return b
}

// Mount creates a new group with a specified base path.
func Mount(mux *http.ServeMux, basePath string) *Bundle {
	b := &Bundle{mux: mux, basePath: basePath}
	return b
}

// ServeHTTP implements the http.Handler interface
func (b *Bundle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.rootRegistered.once.Do(func() {
		if !b.rootRegistered.set && !b.rootRegistered.disableRootNotFoundHandler {
			// register a not found handler for the root path if no / route is registered
			// this is needed to be able to use middleware on all routes, for example logging
			notFoundHandler := http.NotFoundHandler()
			b.register("/", notFoundHandler.ServeHTTP)
		}
	})
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
		basePath:    b.basePath + basePath,
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
	newMiddlewares := make([]func(http.Handler) http.Handler, len(b.middlewares), len(b.middlewares)+len(more)+1)
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
func (b *Bundle) Handle(pattern string, handler http.Handler) {
	b.register(pattern, handler.ServeHTTP)
}

// HandleFunc registers the handler function for the given pattern to the Group's mux.
// The handler is wrapped with the Group's middlewares.
func (b *Bundle) HandleFunc(pattern string, handler http.HandlerFunc) {
	b.register(pattern, handler)
}

// Handler returns the handler and the pattern that matches the request.
// It always returns a non-nil handler, see http.ServeMux.Handler documentation for details.
func (b *Bundle) Handler(r *http.Request) (h http.Handler, pattern string) {
	return b.mux.Handler(r)
}

// DisableNotFoundHandler disables the automatic registration of a not found handler for the root path.
func (b *Bundle) DisableNotFoundHandler() { b.rootRegistered.disableRootNotFoundHandler = true }

// Matches non-space characters, spaces, then anything, i.e. "GET /path/to/resource"
var reGo122 = regexp.MustCompile(`^(\S*)\s+(.*)$`)

func (b *Bundle) register(pattern string, handler http.HandlerFunc) {
	matches := reGo122.FindStringSubmatch(pattern)
	if len(matches) > 2 { // path in the form "GET /path/to/resource"
		pattern = matches[1] + " " + b.basePath + matches[2]
	} else { // path is just "/path/to/resource"
		pattern = b.basePath + pattern
	}

	if pattern == "/" || b.basePath+pattern == "/" {
		b.rootRegistered.set = true
	}
	b.mux.HandleFunc(pattern, b.wrapMiddleware(handler).ServeHTTP)
}

// Route allows for configuring the Group inside the configureFn function.
func (b *Bundle) Route(configureFn func(*Bundle)) { configureFn(b) }

// wrapMiddleware applies the registered middlewares to a handler.
func (b *Bundle) wrapMiddleware(handler http.Handler) http.Handler {
	for i := range b.middlewares {
		handler = b.middlewares[len(b.middlewares)-1-i](handler)
	}
	return handler
}

// Wrap directly wraps the handler with the provided middleware(s).
func Wrap(handler http.Handler, mw1 func(http.Handler) http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return mw1(handler)
}
