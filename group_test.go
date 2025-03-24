package routegroup_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/go-pkgz/routegroup"
)

// testMiddleware is simple middleware for testing purposes.
func testMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Test middleware")
		w.Header().Add("X-Test-Middleware", "true")
		next.ServeHTTP(w, r)
	})
}

func TestGroupMiddleware(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.Use(testMiddleware)

	group.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/test", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	group.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
	if header := recorder.Header().Get("X-Test-Middleware"); header != "true" {
		t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
	}
}
func TestGroupHandle(t *testing.T) {
	group := routegroup.New(http.NewServeMux())

	group.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test handler"))
	})
	group.Handle("GET /test2", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test2 handler"))
	}))

	t.Run("handler function", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if body := recorder.Body.String(); body != "test handler" {
			t.Errorf("Expected body 'test handler', got '%s'", body)
		}
	})

	t.Run("handle, wrong method", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodPost, "/test2", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, recorder.Code)
		}
	})

	t.Run("handler", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/test2", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if body := recorder.Body.String(); body != "test2 handler" {
			t.Errorf("Expected body 'test2 handler', got '%s'", body)
		}
	})
}

func TestBundleHandler(t *testing.T) {
	group := routegroup.New(http.NewServeMux())

	group.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("handler returns correct pattern and handler", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		handler, pattern := group.Handler(request)

		if handler == nil {
			t.Error("Expected handler to be not nil")
		}
		if pattern != "/test" {
			t.Errorf("Expected pattern '/test', got '%s'", pattern)
		}
	})

	t.Run("handler returns not-nil and empty pattern for non-existing route", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "/non-existing", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		handler, pattern := group.Handler(request)

		if handler == nil {
			t.Error("Expected handler to be not nil")
		}
		if pattern != "" {
			t.Errorf("Expected empty pattern, got '%s'", pattern)
		}
	})
}

func TestGroupRoute(t *testing.T) {
	group := routegroup.New(http.NewServeMux())

	group.Route(func(g *routegroup.Bundle) {
		g.Use(testMiddleware)
		g.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		g.HandleFunc("POST /test2", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	t.Run("GET /test", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if header := recorder.Header().Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("POST /test2", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodPost, "/test2", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if header := recorder.Header().Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /test2", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/test2", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, recorder.Code)
		}
		if header := recorder.Header().Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})
}

func TestGroupWithMiddleware(t *testing.T) {
	group := routegroup.New(http.NewServeMux())

	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Original-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	newGroup := group.With(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-New-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	newGroup.HandleFunc("/with-test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	newGroup.HandleFunc("POST /with-test-post-only", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("GET /with-test", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/with-test", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if header := recorder.Header().Get("X-Original-Middleware"); header != "true" {
			t.Errorf("Expected header X-Original-Middleware to be 'true', got '%s'", header)
		}
		if header := recorder.Header().Get("X-New-Middleware"); header != "true" {
			t.Errorf("Expected header X-New-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("POST /with-test", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodPost, "/with-test", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if header := recorder.Header().Get("X-Original-Middleware"); header != "true" {
			t.Errorf("Expected header X-Original-Middleware to be 'true', got '%s'", header)
		}
		if header := recorder.Header().Get("X-New-Middleware"); header != "true" {
			t.Errorf("Expected header X-New-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /not-found", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/not-found", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, recorder.Code)
		}
		if header := recorder.Header().Get("X-Original-Middleware"); header != "true" {
			t.Errorf("Expected header X-Original-Middleware to be 'true', got '%s'", header)
		}
		if header := recorder.Header().Get("X-New-Middleware"); header != "" {
			t.Errorf("Expected header X-New-Middleware to be not set, got '%s'", header)
		}
	})

	t.Run("POST /with-test-post-only", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodPost, "/with-test-post-only", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if header := recorder.Header().Get("X-Original-Middleware"); header != "true" {
			t.Errorf("Expected header X-Original-Middleware to be 'true', got '%s'", header)
		}
		if header := recorder.Header().Get("X-New-Middleware"); header != "true" {
			t.Errorf("Expected header X-New-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /with-test-post-only", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/with-test-post-only", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, recorder.Code)
		}
		if header := recorder.Header().Get("X-Original-Middleware"); header != "true" {
			t.Errorf("Expected header X-Original-Middleware to be 'true', got '%s'", header)
		}
		if header := recorder.Header().Get("X-New-Middleware"); header != "" {
			t.Errorf("Expected header X-New-Middleware to be not set, got '%s'", header)
		}
	})
}

func TestGroupWithMiddlewareAndTopLevelAfter(t *testing.T) {
	group := routegroup.New(http.NewServeMux())

	group.Group().Route(func(g *routegroup.Bundle) {
		g.Use(testMiddleware)
		g.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("test handler"))
		})
	})

	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Top-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	group.HandleFunc("/top", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("top handler"))
	})

	t.Run("GET /top", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/top", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if header := recorder.Header().Get("X-Top-Middleware"); header != "true" {
			t.Errorf("Expected header X-Top-Middleware to be 'true', got '%s'", header)
		}
		if header := recorder.Header().Get("X-Test-Middleware"); header != "" {
			t.Errorf("Expected header X-Test-Middleware not to be set, got '%s'", header)
		}
	})

	t.Run("GET /test", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if header := recorder.Header().Get("X-Top-Middleware"); header == "true" {
			t.Errorf("Expected header X-Top-Middleware not to be set, got '%s'", header)
		}
		if header := recorder.Header().Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})
}

func TestDisableNotFoundHandler(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.DisableNotFoundHandler()

	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Original-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	newGroup := group.With(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-New-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	newGroup.HandleFunc("/with-test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("GET /with-test", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/with-test", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if header := recorder.Header().Get("X-Original-Middleware"); header != "true" {
			t.Errorf("Expected header X-Original-Middleware to be 'true', got '%s'", header)
		}
		if header := recorder.Header().Get("X-New-Middleware"); header != "true" {
			t.Errorf("Expected header X-New-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /not-found", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/not-found", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, recorder.Code)
		}
		if header := recorder.Header().Get("X-Original-Middleware"); header != "" {
			t.Errorf("Expected header X-Original-Middleware to be not set, got '%s'", header)
		}
		if header := recorder.Header().Get("X-New-Middleware"); header != "" {
			t.Errorf("Expected header X-New-Middleware to be not set, got '%s'", header)
		}
	})
}

func TestGroupWithMoreMiddleware(t *testing.T) {
	group := routegroup.New(http.NewServeMux())

	newGroup := group.With(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-New-Middleware", "true")
				next.ServeHTTP(w, r)
			})
		},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-More-Middleware", "true")
				next.ServeHTTP(w, r)
			})
		},
	)

	newGroup.HandleFunc("/with-test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/with-test", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	group.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
	if header := recorder.Header().Get("X-New-Middleware"); header != "true" {
		t.Errorf("Expected header X-New-Middleware to be 'true', got '%s'", header)
	}
	if header := recorder.Header().Get("X-More-Middleware"); header != "true" {
		t.Errorf("Expected header X-More-Middleware to be 'true', got '%s'", header)
	}
}
func TestMount(t *testing.T) {
	basePath := "/api"
	group := routegroup.Mount(http.NewServeMux(), basePath)

	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Mounted-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	group.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, basePath+"/test", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	group.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
	if header := recorder.Header().Get("X-Mounted-Middleware"); header != "true" {
		t.Errorf("Expected header X-Mounted-Middleware to be 'true', got '%s'", header)
	}
}

func TestHTTPServerWithBasePathAndMiddleware(t *testing.T) {
	group := routegroup.Mount(http.NewServeMux(), "/api")

	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Test-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	})

	group.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test handler"))
	})

	testServer := httptest.NewServer(group)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/api/test")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "test handler" {
		t.Errorf("Expected body 'test handler', got '%s'", string(body))
	}
	if header := resp.Header.Get("X-Test-Middleware"); header != "applied" {
		t.Errorf("Expected header X-Test-Middleware to be 'applied', got '%s'", header)
	}
}

func TestHTTPServerWithRoot(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.Use(testMiddleware)
	group.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test handler"))
	})
	group.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("root handler"))
	})
	testServer := httptest.NewServer(group)
	defer testServer.Close()

	t.Run("GET /test", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/test")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "test handler" {
			t.Errorf("Expected body 'test handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "root handler" {
			t.Errorf("Expected body 'root handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("/", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "root handler" {
			t.Errorf("Expected body 'root handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /unknown-path", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/unknown-path")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
		if string(body) != "404 page not found\n" {
			t.Errorf("Expected body '404 page not found\n', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})
}

func TestHTTPServerWithRoot122(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.Use(testMiddleware)
	group.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test handler"))
	})
	group.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("root handler"))
	})
	testServer := httptest.NewServer(group)
	defer testServer.Close()

	t.Run("GET /test", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/test")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "test handler" {
			t.Errorf("Expected body 'test handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "root handler" {
			t.Errorf("Expected body 'root handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("POST /", func(t *testing.T) {
		resp, err := http.Post(testServer.URL+"/", "application/json", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
		if string(body) != "404 page not found\n" {
			t.Errorf("Expected body '404 page not found\n', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /unknown-path", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/unknown-path")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
		if string(body) != "404 page not found\n" {
			t.Errorf("Expected body '404 page not found\n', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})
}

func TestRootAndCatchAll(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.Use(testMiddleware)
	group.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("root handler"))
	})
	group.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("custom not found handler"))
	})

	testServer := httptest.NewServer(group)
	defer testServer.Close()

	t.Run("GET /", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
		if string(body) != "root handler" {
			t.Errorf("Expected body 'root handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /unknown-path", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/unknown-path")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
		if string(body) != "custom not found handler" {
			t.Errorf("Expected body 'custom not found handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})
}

func TestHTTPServerWithBasePathNoMiddleware(t *testing.T) {
	group := routegroup.Mount(http.NewServeMux(), "/api")
	group.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test handler"))
	})

	testServer := httptest.NewServer(group)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/api/test")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "test handler" {
		t.Errorf("Expected body 'test handler', got '%s'", string(body))
	}
}

func TestHTTPServerMethodAndPathHandling(t *testing.T) {
	group := routegroup.Mount(http.NewServeMux(), "/api")

	group.Use(testMiddleware)

	group.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("GET test method handler"))
	})

	group.HandleFunc("/test2", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test2 method handler"))
	})

	testServer := httptest.NewServer(group)
	defer testServer.Close()

	t.Run("handle with verb", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/test")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "GET test method handler" {
			t.Errorf("Expected body 'GET test method handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("handle without verb", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/test2")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "test2 method handler" {
			t.Errorf("Expected body 'test2 method handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})
}

func TestHTTPServerWrap(t *testing.T) {
	mw1 := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-MW1", "1")
			h.ServeHTTP(w, r)
		})
	}
	mw2 := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-MW2", "2")
			h.ServeHTTP(w, r)
		})
	}

	handlers := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test handler"))
	})

	ts := httptest.NewServer(routegroup.Wrap(handlers, mw1, mw2))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if header := resp.Header.Get("X-MW1"); header != "1" {
		t.Errorf("Expected header X-MW1 to be '1', got '%s'", header)
	}
	if header := resp.Header.Get("X-MW2"); header != "2" {
		t.Errorf("Expected header X-MW2 to be '2', got '%s'", header)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "test handler" {
		t.Errorf("Expected body 'test handler', got '%s'", string(body))
	}
}

func TestHTTPServerWithDerived(t *testing.T) {
	// create a new bundle with default middleware
	bundle := routegroup.New(http.NewServeMux())
	bundle.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found handler"))
	})
	bundle.Use(testMiddleware)

	// mount a group with additional middleware on /api
	group1 := bundle.Mount("/api")
	group1.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-API-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	})

	group1.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("GET test method handler"))
	})
	group1.HandleFunc("POST /", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("POST api / method handler"))
	})

	// add another group with middleware
	bundle.Group().Route(func(g *routegroup.Bundle) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Blah-Middleware", "true")
				next.ServeHTTP(w, r)
			})
		})
		g.HandleFunc("GET /blah/blah", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("GET blah method handler"))
		})
	})

	// mount the bundle on /auth under /api
	group1.Mount("/auth").Route(func(g *routegroup.Bundle) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Auth-Middleware", "true")
				next.ServeHTTP(w, r)
			})
		})
		g.HandleFunc("GET /auth-test", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("GET auth-test method handler"))
		})
		g.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("auth GET / method handler"))
		})
	})

	testServer := httptest.NewServer(bundle)
	defer testServer.Close()

	t.Run("GET /api/test", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/test")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "GET test method handler" {
			t.Errorf("Expected body 'GET test method handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /blah/blah", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/blah/blah")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "GET blah method handler" {
			t.Errorf("Expected body 'GET blah method handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Blah-Middleware"); header != "true" {
			t.Errorf("Expected header X-Blah-Middleware to be 'true', got '%s'", header)
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /api/auth/auth-test", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/auth/auth-test")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "GET auth-test method handler" {
			t.Errorf("Expected body 'GET auth-test method handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Auth-Middleware"); header != "true" {
			t.Errorf("Expected header X-Auth-Middleware to be 'true', got '%s'", header)
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /api/auth/", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/auth/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "auth GET / method handler" {
			t.Errorf("Expected body 'GET auth-test method handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Auth-Middleware"); header != "true" {
			t.Errorf("Expected header X-Auth-Middleware to be 'true', got '%s'", header)
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("POST /api/", func(t *testing.T) {
		resp, err := http.Post(testServer.URL+"/api/", "application/json", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
		if string(body) != "POST api / method handler" {
			t.Errorf("Expected body 'GET auth-test method handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Auth-Middleware"); header != "" {
			t.Errorf("Expected header X-Auth-Middleware to be empty, got '%s'", header)
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("POST /api/not-found", func(t *testing.T) {
		resp, err := http.Post(testServer.URL+"/api/not-found", "application/json", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
		if string(body) != "not found handler" {
			t.Errorf("Expected body '404 page not found', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Auth-Middleware"); header != "" {
			t.Errorf("Expected header X-Auth-Middleware to be empty, got '%s'", header)
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /api/", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
		if string(body) != "not found handler" {
			t.Errorf("Expected body '404 page not found', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Auth-Middleware"); header != "" {
			t.Errorf("Expected header X-Auth-Middleware to be empty, got '%s'", header)
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /not-found", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/not-found")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})
}

func TestHTTPServerWithCustomNotFound(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.Use(testMiddleware)
	group.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Custom 404: Page not found!", http.StatusNotFound)
	})

	apiGroup := group.Mount("/api")
	apiGroup.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test handler"))
	})

	testServer := httptest.NewServer(group)
	defer testServer.Close()

	t.Run("GET /api/test", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/test")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "test handler" {
			t.Errorf("Expected body 'test handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /api/not-found", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/not-found")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("body: %s", body)

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
		}

		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
		if string(body) != "Custom 404: Page not found!\n" {
			t.Errorf("Expected body 'Custom 404: Page not found!', got '%s'", string(body))
		}
	})

	t.Run("GET /not-found", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/not-found")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("body: %s", body)

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
		if string(body) != "Custom 404: Page not found!\n" {
			t.Errorf("Expected body 'Custom 404: Page not found!', got '%s'", string(body))
		}
	})
}

func TestHTTPServerWithCustomNotFoundNon404Status(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.Use(testMiddleware)
	group.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("Custom 404: Page not found!\n"))
	})

	apiGroup := group.Mount("/api")
	apiGroup.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test handler"))
	})

	testServer := httptest.NewServer(group)
	defer testServer.Close()

	t.Run("GET /api/test", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/test")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
		if string(body) != "test handler" {
			t.Errorf("Expected body 'test handler', got '%s'", string(body))
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("GET /api/not-found", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/api/not-found")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("body: %s", body)

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, resp.StatusCode)
		}
		if header := resp.Header.Get("X-Test-Middleware"); header != "true" {
			t.Errorf("Expected header X-Test-Middleware to be 'true', got '%s'", header)
		}
		if string(body) != "Custom 404: Page not found!\n" {
			t.Errorf("Expected body 'Custom 404: Page not found!', got '%s'", string(body))
		}
	})
}

func TestCustomNotFoundHandlerChange(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "First handler", http.StatusNotFound)
	})

	group.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Second handler", http.StatusNotFound)
	})

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/not-found", http.NoBody)
	group.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("got %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec.Body.String() != "Second handler\n" {
		t.Errorf("got %q, want %q", rec.Body.String(), "Second handler\n")
	}
}

func TestDisableNotFoundHandlerAfterRouteRegistration(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("test")); err != nil {
			t.Fatal(err)
		}
	})
	group.DisableNotFoundHandler()

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/not-found", http.NoBody)
	group.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("got %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec.Body.String() != "404 page not found\n" {
		t.Errorf("got %q, want %q", rec.Body.String(), "404 page not found\n")
	}
}

func TestConcurrentRequests(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.HandleFunc("/concurrent", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/concurrent", http.NoBody)
			group.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("got %d, want %d", rec.Code, http.StatusOK)
			}
		}()
	}
	wg.Wait()
}

func TestMethodPatternsWithDifferentMethods(t *testing.T) {
	group := routegroup.New(http.NewServeMux())
	group.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, "GET handler"); err != nil {
			t.Fatal(err)
		}
	})
	group.HandleFunc("POST /test", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, "POST handler"); err != nil {
			t.Fatal(err)
		}
	})

	tests := []struct {
		method, path, expected string
		code                   int
	}{
		{http.MethodGet, "/test", "GET handler", http.StatusOK},
		{http.MethodPost, "/test", "POST handler", http.StatusOK},
		{http.MethodPut, "/test", "404 page not found\n", http.StatusNotFound},
	}

	for _, tt := range tests {
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(tt.method, tt.path, http.NoBody)
		group.ServeHTTP(rec, req)
		if rec.Code != tt.code {
			t.Errorf("got %d, want %d", rec.Code, tt.code)
		}
		if rec.Body.String() != tt.expected {
			t.Errorf("got %q, want %q", rec.Body.String(), tt.expected)
		}
	}
}

func TestMountNested(t *testing.T) {
	bundle := routegroup.New(http.NewServeMux())
	api := bundle.Mount("/api")
	v1 := api.Mount("/v1")
	v1.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("v1 test")); err != nil {
			t.Fatal(err)
		}
	})

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/test", http.NoBody)
	bundle.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "v1 test" {
		t.Errorf("got %q, want %q", rec.Body.String(), "v1 test")
	}
}

func TestStaticFileServer(t *testing.T) {
	dir := t.TempDir()

	// create test file structure
	content := []byte("static file content")
	err := os.WriteFile(filepath.Join(dir, "test.txt"), content, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, "index.html"), []byte("index content"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// create subdirectory
	subDir := filepath.Join(dir, "sub")
	if err = os.Mkdir(subDir, 0o750); err != nil {
		t.Fatal(err)
	}
	subContent := []byte("sub file content")
	err = os.WriteFile(filepath.Join(subDir, "sub.txt"), subContent, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("serve files from root path with HEAD", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		router.HandleFiles("/", http.Dir(dir))

		srv := httptest.NewServer(router)
		defer srv.Close()

		// test GET request
		resp, err := http.Get(srv.URL + "/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET - got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if !bytes.Equal(body, content) {
			t.Errorf("GET - got body %q, want %q", body, content)
		}

		// test HEAD request
		req, err := http.NewRequest(http.MethodHead, srv.URL+"/test.txt", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("HEAD - got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if len(body) != 0 {
			t.Errorf("HEAD - should have no body, got %d bytes", len(body))
		}
		if cl := resp.Header.Get("Content-Length"); cl != fmt.Sprint(len(content)) {
			t.Errorf("HEAD - got Content-Length %s, want %d", cl, len(content))
		}
	})

	t.Run("serve files from /files/ prefix", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		router.HandleFiles("/files", http.Dir(dir))

		srv := httptest.NewServer(router)
		defer srv.Close()

		// test GET request
		resp, err := http.Get(srv.URL + "/files/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET - got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if !bytes.Equal(body, content) {
			t.Errorf("GET - got body %q, want %q", body, content)
		}

		// test HEAD request
		req, err := http.NewRequest(http.MethodHead, srv.URL+"/files/test.txt", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("HEAD - got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if len(body) != 0 {
			t.Errorf("HEAD - should have no body, got %d bytes", len(body))
		}
		if cl := resp.Header.Get("Content-Length"); cl != fmt.Sprint(len(content)) {
			t.Errorf("HEAD - got Content-Length %s, want %d", cl, len(content))
		}
	})

	t.Run("serve files from mounted group", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		assets := router.Mount("/assets")
		assets.HandleFiles("/", http.Dir(dir))

		srv := httptest.NewServer(router)
		defer srv.Close()

		// test GET request
		resp, err := http.Get(srv.URL + "/assets/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET - got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if !bytes.Equal(body, content) {
			t.Errorf("GET - got body %q, want %q", body, content)
		}

		// test HEAD request
		req, err := http.NewRequest(http.MethodHead, srv.URL+"/assets/test.txt", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("HEAD - got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if len(body) != 0 {
			t.Errorf("HEAD - should have no body, got %d bytes", len(body))
		}
		if cl := resp.Header.Get("Content-Length"); cl != fmt.Sprint(len(content)) {
			t.Errorf("HEAD - got Content-Length %s, want %d", cl, len(content))
		}
	})
}

func TestDirectFileServerHandle(t *testing.T) {
	dir := t.TempDir()

	content := []byte("static file content")
	err := os.WriteFile(filepath.Join(dir, "test.txt"), content, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("raw Handle without strip", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		router.Handle("/files/", http.FileServer(http.Dir(dir))) // without StripPrefix!

		srv := httptest.NewServer(router)
		defer srv.Close()

		// test GET request - should fail as we need StripPrefix
		resp, err := http.Get(srv.URL + "/files/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 without StripPrefix, got %d", resp.StatusCode)
		}

		// test HEAD request - should also fail
		req, err := http.NewRequest(http.MethodHead, srv.URL+"/files/test.txt", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("HEAD - expected 404 without StripPrefix, got %d", resp.StatusCode)
		}
	})

	t.Run("Handle with strip prefix", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		router.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(dir))))

		srv := httptest.NewServer(router)
		defer srv.Close()

		// test GET request
		resp, err := http.Get(srv.URL + "/files/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET - got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if !bytes.Equal(body, content) {
			t.Errorf("GET - got body %q, want %q", body, content)
		}

		// test HEAD request
		req, err := http.NewRequest(http.MethodHead, srv.URL+"/files/test.txt", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("HEAD - got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if len(body) != 0 {
			t.Errorf("HEAD - should have no body, got %d bytes", len(body))
		}
		if cl := resp.Header.Get("Content-Length"); cl != fmt.Sprint(len(content)) {
			t.Errorf("HEAD - got Content-Length %s, want %d", cl, len(content))
		}
	})

	t.Run("Handle with mounted group", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		api := router.Mount("/api")
		api.Handle("/static/", http.StripPrefix("/api/static/", http.FileServer(http.Dir(dir))))

		srv := httptest.NewServer(router)
		defer srv.Close()

		// test GET request
		resp, err := http.Get(srv.URL + "/api/static/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET - got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if !bytes.Equal(body, content) {
			t.Errorf("GET - got body %q, want %q", body, content)
		}

		// test HEAD request
		req, err := http.NewRequest(http.MethodHead, srv.URL+"/api/static/test.txt", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("HEAD - got status %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if len(body) != 0 {
			t.Errorf("HEAD - should have no body, got %d bytes", len(body))
		}
		if cl := resp.Header.Get("Content-Length"); cl != fmt.Sprint(len(content)) {
			t.Errorf("HEAD - got Content-Length %s, want %d", cl, len(content))
		}
	})
}

func TestFileServerWithMiddleware(t *testing.T) {
	dir := t.TempDir()

	content := []byte("static file content")
	err := os.WriteFile(filepath.Join(dir, "test.txt"), content, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("root path with middleware", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Root-MW", "called")
				next.ServeHTTP(w, r)
			})
		})
		router.HandleFiles("/", http.Dir(dir))

		srv := httptest.NewServer(router)
		defer srv.Close()

		// test GET request
		resp, err := http.Get(srv.URL + "/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if mw := resp.Header.Get("X-Root-MW"); mw != "called" {
			t.Errorf("middleware not called, got header %q", mw)
		}

		// test HEAD request
		req, err := http.NewRequest(http.MethodHead, srv.URL+"/test.txt", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if mw := resp.Header.Get("X-Root-MW"); mw != "called" {
			t.Errorf("middleware not called for HEAD, got header %q", mw)
		}
	})

	t.Run("prefixed path with middleware", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Prefix-MW", "called")
				next.ServeHTTP(w, r)
			})
		})
		router.HandleFiles("/files", http.Dir(dir))

		srv := httptest.NewServer(router)
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/files/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if mw := resp.Header.Get("X-Prefix-MW"); mw != "called" {
			t.Errorf("middleware not called, got header %q", mw)
		}
	})

	t.Run("mounted path with chained middleware", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Root-MW", "called")
				next.ServeHTTP(w, r)
			})
		})

		assets := router.Mount("/assets")
		assets.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Assets-MW", "called")
				next.ServeHTTP(w, r)
			})
		})
		assets.HandleFiles("/", http.Dir(dir))

		srv := httptest.NewServer(router)
		defer srv.Close()

		// test both middleware being called
		resp, err := http.Get(srv.URL + "/assets/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if mw := resp.Header.Get("X-Root-MW"); mw != "called" {
			t.Errorf("root middleware not called, got header %q", mw)
		}
		if mw := resp.Header.Get("X-Assets-MW"); mw != "called" {
			t.Errorf("assets middleware not called, got header %q", mw)
		}

		// test 404 path still triggers middleware
		resp, err = http.Get(srv.URL + "/assets/notfound.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
		if mw := resp.Header.Get("X-Root-MW"); mw != "called" {
			t.Errorf("root middleware not called for 404, got header %q", mw)
		}
		if mw := resp.Header.Get("X-Assets-MW"); mw != "called" {
			t.Errorf("assets middleware not called for 404, got header %q", mw)
		}
	})

	t.Run("direct Handle with middleware", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Direct-MW", "called")
				next.ServeHTTP(w, r)
			})
		})
		router.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(dir))))

		srv := httptest.NewServer(router)
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/files/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if mw := resp.Header.Get("X-Direct-MW"); mw != "called" {
			t.Errorf("middleware not called, got header %q", mw)
		}
	})
}

func TestMixedHandlers(t *testing.T) {
	dir := t.TempDir()

	// create static files
	content := []byte("static file content")
	err := os.WriteFile(filepath.Join(dir, "test.txt"), content, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	router := routegroup.New(http.NewServeMux())

	// setup regular and file handlers in various combinations
	router.HandleFunc("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api info"))
	})
	router.HandleFiles("/public", http.Dir(dir))

	// setup api group with mixed handlers
	api := router.Mount("/v1")
	api.HandleFunc("GET /data", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api data"))
	})
	api.HandleFiles("/static", http.Dir(dir))

	// setup admin group with both types and middleware
	admin := router.Mount("/admin")
	admin.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Admin", "true")
			next.ServeHTTP(w, r)
		})
	})
	admin.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("admin users"))
	})
	admin.HandleFiles("/assets", http.Dir(dir))

	srv := httptest.NewServer(router)
	defer srv.Close()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
		expectedHeader string // for middleware check
	}{
		{"api info endpoint", "/api/info", http.StatusOK, "api info", ""},
		{"public static file", "/public/test.txt", http.StatusOK, "static file content", ""},
		{"v1 api endpoint", "/v1/data", http.StatusOK, "api data", ""},
		{"v1 static file", "/v1/static/test.txt", http.StatusOK, "static file content", ""},
		{"admin endpoint", "/admin/users", http.StatusOK, "admin users", "true"},
		{"admin static file", "/admin/assets/test.txt", http.StatusOK, "static file content", "true"},
		{"non-existent api path", "/api/notfound", http.StatusNotFound, "404 page not found\n", ""},
		{"non-existent static file", "/public/notfound.txt", http.StatusNotFound, "404 page not found\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tt.path)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("got status %d, want %d", resp.StatusCode, tt.expectedStatus)
			}

			if string(body) != tt.expectedBody {
				t.Errorf("got body %q, want %q", string(body), tt.expectedBody)
			}

			if tt.expectedHeader != "" {
				if h := resp.Header.Get("X-Admin"); h != tt.expectedHeader {
					t.Errorf("got X-Admin header %q, want %q", h, tt.expectedHeader)
				}
			}
		})
	}
}

func TestHandleTrailingSlash(t *testing.T) {
	router := routegroup.New(http.NewServeMux())

	t.Run("handler for pattern with trailing slash", func(t *testing.T) {
		router.Handle("/path/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("handler with trailing slash"))
		}))
		router.HandleFunc("GET /path/sub", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("sub handler"))
		})

		srv := httptest.NewServer(router)
		defer srv.Close()

		tests := []struct {
			name       string
			method     string
			path       string
			wantStatus int
			wantBody   string
		}{
			{"GET /path/", http.MethodGet, "/path/", http.StatusOK, "handler with trailing slash"},
			{"POST /path/", http.MethodPost, "/path/", http.StatusOK, "handler with trailing slash"},
			{"GET /path/sub", http.MethodGet, "/path/sub", http.StatusOK, "sub handler"},                   // more specific route wins
			{"POST /path/sub", http.MethodPost, "/path/sub", http.StatusOK, "handler with trailing slash"}, // falls back to /path/
			{"GET /path/anything", http.MethodGet, "/path/anything", http.StatusOK, "handler with trailing slash"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, err := http.NewRequest(tt.method, srv.URL+tt.path, http.NoBody)
				if err != nil {
					t.Fatal(err)
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tt.wantStatus {
					t.Errorf("got status %d, want %d", resp.StatusCode, tt.wantStatus)
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}

				if got := string(body); got != tt.wantBody {
					t.Errorf("got body %q, want %q", got, tt.wantBody)
				}
			})
		}
	})

	t.Run("mounted handler with trailing slash", func(t *testing.T) {
		api := router.Mount("/api")
		api.Handle("/v1/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("api v1"))
		}))
		api.HandleFunc("GET /v1/data", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("api data"))
		})

		srv := httptest.NewServer(router)
		defer srv.Close()

		tests := []struct {
			name       string
			method     string
			path       string
			wantStatus int
			wantBody   string
		}{
			{"GET /api/v1/", http.MethodGet, "/api/v1/", http.StatusOK, "api v1"},
			{"POST /api/v1/", http.MethodPost, "/api/v1/", http.StatusOK, "api v1"},
			{"GET /api/v1/data", http.MethodGet, "/api/v1/data", http.StatusOK, "api data"}, // more specific route wins
			{"POST /api/v1/data", http.MethodPost, "/api/v1/data", http.StatusOK, "api v1"}, // falls back to /v1/
			{"GET /api/v1/anything", http.MethodGet, "/api/v1/anything", http.StatusOK, "api v1"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, err := http.NewRequest(tt.method, srv.URL+tt.path, http.NoBody)
				if err != nil {
					t.Fatal(err)
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tt.wantStatus {
					t.Errorf("got status %d, want %d", resp.StatusCode, tt.wantStatus)
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}

				if got := string(body); got != tt.wantBody {
					t.Errorf("got body %q, want %q", got, tt.wantBody)
				}
			})
		}
	})
}

func TestIssue12StaticAndIndex(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("static content"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	router := routegroup.New(http.NewServeMux())
	router.Route(func(base *routegroup.Bundle) {
		base.Handle("/", http.FileServer(http.Dir(dir)))
		base.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("index page"))
		})
		base.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("login page"))
		})
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	t.Run("serve static file", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if got := string(body); got != "static content" {
			t.Errorf("got %q, want static content", got)
		}
	})

	t.Run("serve index", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if got := string(body); got != "index page" {
			t.Errorf("got %q, want index page", got)
		}
	})

	t.Run("serve login page", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/login")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if got := string(body); got != "login page" {
			t.Errorf("got %q, want login page", got)
		}
	})
}

func TestInvalidPatterns(t *testing.T) {
	group := routegroup.New(http.NewServeMux())

	tests := []struct {
		name      string
		pattern   string
		path      string // actual URL path to test
		wantPanic bool
	}{
		{"empty pattern", "", "/", true},                                                // ServeMux panics on empty pattern
		{"just spaces", "  ", "/", true},                                                // ServeMux panics on spaces-only pattern
		{"spaces in path", "GET /path%20with%20spaces", "/path%20with%20spaces", false}, // encoded spaces work
		{"only method", "GET", "/", true},                                               // ServeMux panics on invalid pattern
		{"just one slash", "/", "/", false},                                             // root path works
		{"missing slash", "GET /path", "/path", false},                                  // normal pattern
		{"double slashes", "GET //path", "/", true},                                     // ServeMux panics on unclean paths
		{"path without slash", "path", "/", true},                                       // must start with /
		{"method path without slash", "GET path", "/", true},                            // must start with /
		{"trailing slash", "GET /path/", "/path/", false},                               // trailing slash is ok
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("handler"))
			}

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic but got none")
					}
				}()
				group.HandleFunc(tt.pattern, handler)
				return
			}

			group.HandleFunc(tt.pattern, handler)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			group.ServeHTTP(rec, req)

			if rec.Code == 0 {
				t.Error("no response code set")
			}
		})
	}
}

func TestMiddlewareOrder(t *testing.T) {
	var order []string

	mkMiddleware := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "before "+name)
				next.ServeHTTP(w, r)
				order = append(order, "after "+name)
			})
		}
	}

	group := routegroup.New(http.NewServeMux())
	group.Use(mkMiddleware("root"))

	api := group.Mount("/api")
	api.Use(mkMiddleware("api"))

	users := api.With(mkMiddleware("users"))
	users.HandleFunc("/action", func(w http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
		_, _ = w.Write([]byte("ok"))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/action", http.NoBody)
	group.ServeHTTP(rec, req)

	expected := []string{
		"before root",
		"before api",
		"before users",
		"handler",
		"after users",
		"after api",
		"after root",
	}

	if !reflect.DeepEqual(order, expected) {
		t.Errorf("wrong middleware execution order\nwant: %v\ngot:  %v", expected, order)
	}
}

func TestMountPointMethodConflicts(t *testing.T) {
	group := routegroup.New(http.NewServeMux())

	// register handler for /api directly
	group.HandleFunc("GET /api", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("api root"))
	})

	// mount a group at /api
	api := group.Mount("/api")
	api.HandleFunc("/users", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("users"))
	})

	srv := httptest.NewServer(group)
	defer srv.Close()

	t.Run("get /api root", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if string(body) != "api root" {
			t.Errorf("expected 'api root', got %q", string(body))
		}
	})

	t.Run("get /api/users", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/users")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if string(body) != "users" {
			t.Errorf("expected 'users', got %q", string(body))
		}
	})
}

func TestDeepNestedMounts(t *testing.T) {
	var callOrder []string
	mkMiddleware := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callOrder = append(callOrder, "before "+name)
				next.ServeHTTP(w, r)
				callOrder = append(callOrder, "after "+name)
			})
		}
	}

	group := routegroup.New(http.NewServeMux())
	group.Use(mkMiddleware("root"))

	v1 := group.Mount("/v1")
	v1.Use(mkMiddleware("v1"))

	api := v1.Mount("/api")
	api.Use(mkMiddleware("api"))

	users := api.Mount("/users")
	users.Use(mkMiddleware("users"))

	users.HandleFunc("/list", func(w http.ResponseWriter, _ *http.Request) {
		callOrder = append(callOrder, "handler")
		_, _ = w.Write([]byte("users list"))
	})

	srv := httptest.NewServer(group)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/api/users/list")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if string(body) != "users list" {
		t.Errorf("expected 'users list', got %q", string(body))
	}

	expected := []string{
		"before root",
		"before v1",
		"before api",
		"before users",
		"handler",
		"after users",
		"after api",
		"after v1",
		"after root",
	}

	if !reflect.DeepEqual(callOrder, expected) {
		t.Errorf("middleware execution order mismatch\nwant: %v\ngot:  %v", expected, callOrder)
	}
}

// TestSubgroupRootPathMatching tests that a subgroup with a root path pattern (/)
// properly matches requests to the exact path without a trailing slash.
func TestSubgroupRootPathMatching(t *testing.T) {
	mux := http.NewServeMux()
	router := routegroup.New(mux)

	// Create a mounted group at /api/v1/users
	usersGroup := router.Mount("/api/v1/users")

	// Register handler for the root of the mounted group using "/"
	usersGroup.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("users root"))
	})

	// Also add a child route for comparison
	usersGroup.HandleFunc("GET /list", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("users list"))
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	t.Run("exact match without trailing slash", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/users")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if string(body) != "users root" {
			t.Errorf("expected 'users root', got %q", string(body))
		}
	})

	t.Run("with trailing slash", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/users/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if string(body) != "users root" {
			t.Errorf("expected 'users root', got %q", string(body))
		}
	})

	t.Run("child route", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/users/list")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if string(body) != "users list" {
			t.Errorf("expected 'users list', got %q", string(body))
		}
	})
}

func ExampleNew() {
	group := routegroup.New(http.NewServeMux())

	// apply middleware to the group
	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Mounted-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	// add test handlers
	group.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	group.HandleFunc("POST /test2", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// start the server
	if err := http.ListenAndServe(":8080", group); err != nil {
		panic(err)
	}
}

func ExampleMount() {
	group := routegroup.Mount(http.NewServeMux(), "/api")

	// apply middleware to the group
	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Test-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	// add test handlers
	group.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	group.HandleFunc("POST /test2", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// start the server
	if err := http.ListenAndServe(":8080", group); err != nil {
		panic(err)
	}
}

func ExampleBundle_Route() {
	group := routegroup.New(http.NewServeMux())

	// configure the group using Set
	group.Route(func(g *routegroup.Bundle) {
		// apply middleware to the group
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Test-Middleware", "true")
				next.ServeHTTP(w, r)
			})
		})
		// add test handlers
		g.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		g.HandleFunc("POST /test2", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	// start the server
	if err := http.ListenAndServe(":8080", group); err != nil {
		panic(err)
	}
}
