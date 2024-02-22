package routegroup_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-pkgz/routegroup"
)

// testMiddleware is simple middleware for testing purposes.
func testMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Test-Middleware", "true")
		next.ServeHTTP(w, r)
	})
}

func TestGroupMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	group := routegroup.New(mux)

	// apply middleware to the group
	group.Use(testMiddleware)

	// add a test handler
	group.Handle("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// make a request to the test handler
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
	mux.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "true", recorder.Header().Get("X-Test-Middleware"))
}

func TestGroupHandle(t *testing.T) {
	mux := http.NewServeMux()
	group := routegroup.New(mux)

	// add a test handler
	group.Handle("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// make a request to the test handler
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
	mux.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestGroupSet(t *testing.T) {
	mux := http.NewServeMux()
	group := routegroup.New(mux)

	// configure the group using Set
	group.Set(func(g *routegroup.Bundle) {
		g.Use(testMiddleware)
		g.Handle("/test", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	// make a request to the test handler
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
	mux.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "true", recorder.Header().Get("X-Test-Middleware"))
}

func TestGroupWithMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	group := routegroup.New(mux)

	// original group middleware
	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Original-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	// new group with additional middleware
	newGroup := group.With(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-New-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	// add a test handler to the new group
	newGroup.Handle("/with-test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// make a request to the test handler
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodGet, "/with-test", http.NoBody)
	mux.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "true", recorder.Header().Get("X-Original-Middleware"))
	assert.Equal(t, "true", recorder.Header().Get("X-New-Middleware"))
}

func TestMount(t *testing.T) {
	mux := http.NewServeMux()
	basePath := "/api"
	group := routegroup.Mount(mux, basePath)

	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Mounted-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	// add a test handler
	group.Handle("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// make a request to the mounted handler
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodGet, basePath+"/test", http.NoBody)
	mux.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "true", recorder.Header().Get("X-Mounted-Middleware"))
}

func TestHTTPServerWithBasePathAndMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	group := routegroup.Mount(mux, "/api")

	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Test-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	})

	group.Handle("/test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test handler"))
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/api/test")
	assert.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, "test handler", string(body))
	assert.Equal(t, "applied", resp.Header.Get("X-Test-Middleware"))
}

func TestHTTPServerMethodAndPathHandling(t *testing.T) {
	mux := http.NewServeMux()
	group := routegroup.New(mux)

	group.Use(testMiddleware)

	group.Handle("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("GET method handler"))
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/test")
	assert.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, "GET method handler", string(body))
	assert.Equal(t, "true", resp.Header.Get("X-Test-Middleware"))
}

func ExampleNew() {
	mux := http.NewServeMux()
	group := routegroup.New(mux)

	// apply middleware to the group
	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Test-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	// add test handlers
	group.Handle("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	group.Handle("POST /test2", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// start the server
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}

func ExampleMount() {
	mux := http.NewServeMux()
	group := routegroup.Mount(mux, "/api")

	// apply middleware to the group
	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Test-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	// add test handlers
	group.Handle("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	group.Handle("POST /test2", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// start the server
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}

func ExampleBundle_Set() {
	mux := http.NewServeMux()
	group := routegroup.New(mux)

	// configure the group using Set
	group.Set(func(g *routegroup.Bundle) {
		// apply middleware to the group
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Test-Middleware", "true")
				next.ServeHTTP(w, r)
			})
		})
		// add test handlers
		g.Handle("GET /test", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		g.Handle("POST /test2", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	// start the server
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
