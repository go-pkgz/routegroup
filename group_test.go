package routegroup_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

	t.Run("handle, wrong method -> 405", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodPost, "/test2", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
		}
		if allow := recorder.Header().Get("Allow"); !strings.Contains(allow, http.MethodGet) {
			t.Errorf("expected Allow header to contain GET, got %q", allow)
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

	t.Run("GET /test2 wrong method -> 405", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/test2", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
		}
		if allow := recorder.Header().Get("Allow"); allow != http.MethodPost {
			t.Errorf("expected Allow header to be POST, got %q", allow)
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

	t.Run("GET /with-test-post-only wrong method -> 405", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/with-test-post-only", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		group.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
		}
		if allow := recorder.Header().Get("Allow"); allow != http.MethodPost {
			t.Errorf("expected Allow header to be POST, got %q", allow)
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

	// create subgroup and register route
	sub := group.Group()
	sub.Use(testMiddleware)
	sub.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test handler"))
	})

	// calling Use on the same subgroup after routes should panic
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on Use after routes registration on the same bundle")
		}
	}()

	sub.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Top-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})
}

// Test that calling Use after routes are registered on the same bundle panics,
// and that calling Use on a parent after child routes is allowed.
func TestUseAfterRoutesPanicsAndParentAllowed(t *testing.T) {
	t.Run("root: Use after route panics", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		router.HandleFunc("/r", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic on root.Use after routes registration on the same bundle")
			}
		}()
		router.Use(testMiddleware)
	})

	t.Run("parent: Use after child routes is allowed", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		child := router.Group()
		child.HandleFunc("/child", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})

		// parent hasn't registered any routes yet; calling Use should not panic
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Parent", "true")
				next.ServeHTTP(w, r)
			})
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/child", http.NoBody)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status %d", rec.Code)
		}
		if hv := rec.Header().Get("X-Parent"); hv != "true" {
			t.Fatalf("expected global parent middleware to apply, got %q", hv)
		}
	})
}

// DisableNotFoundHandler semantics are removed; global middlewares always apply.

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

	t.Run("POST / wrong method -> 405", func(t *testing.T) {
		resp, err := http.Post(testServer.URL+"/", "application/json", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
		}
		if string(body) != "Method Not Allowed\n" {
			t.Errorf("Expected body 'Method Not Allowed', got '%s'", string(body))
		}
		if allow := resp.Header.Get("Allow"); !strings.Contains(allow, http.MethodGet) {
			t.Errorf("expected Allow header to contain GET, got %q", allow)
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
		{http.MethodPut, "/test", "Method Not Allowed\n", http.StatusMethodNotAllowed},
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

	// create a mounted group at /api/v1/users
	usersGroup := router.Mount("/api/v1/users")

	// add middleware to the group to test middleware invocation
	usersGroup.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Users-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	})

	// register handler for the root of the mounted group using "/"
	usersGroup.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("users root"))
	})

	// also add a child route for comparison
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

		// check middleware was applied
		middlewareHeader := resp.Header.Get("X-Users-Middleware")
		if middlewareHeader != "applied" {
			t.Errorf("expected middleware header to be 'applied', got %q", middlewareHeader)
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

		// check middleware was applied
		middlewareHeader := resp.Header.Get("X-Users-Middleware")
		if middlewareHeader != "applied" {
			t.Errorf("expected middleware header to be 'applied', got %q", middlewareHeader)
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

		// check middleware was applied
		middlewareHeader := resp.Header.Get("X-Users-Middleware")
		if middlewareHeader != "applied" {
			t.Errorf("expected middleware header to be 'applied', got %q", middlewareHeader)
		}
	})
}

func TestHandleRoot(t *testing.T) {
	// create client that doesn't follow redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // don't follow redirects
		},
	}

	t.Run("HandleRoot with middleware", func(t *testing.T) {
		group := routegroup.New(http.NewServeMux())
		group.Mount("/api").Route(func(apiGroup *routegroup.Bundle) {
			apiGroup.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Middleware", "applied")
					next.ServeHTTP(w, r)
				})
			})
			apiGroup.HandleRoot("GET", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte("api root")); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))
			apiGroup.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte("test")); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})
		})
		group.Mount("/api-2").Route(func(apiGroup *routegroup.Bundle) {
			apiGroup.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Middleware", "applied")
					next.ServeHTTP(w, r)
				})
			})
			apiGroup.HandleRootFunc("GET", func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte("api root")); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})
			apiGroup.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte("test")); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})
		})

		ts := httptest.NewServer(group)
		defer ts.Close()

		apis := []string{"/api", "/api-2"}
		for _, api := range apis {
			// test direct access to registered root /api - should NOT redirect and middleware should be applied
			resp, err := client.Get(ts.URL + api)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			if resp.Header.Get("X-Middleware") != "applied" {
				t.Errorf("middleware not applied")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			if string(body) != "api root" {
				t.Errorf("expected 'api root', got '%s'", body)
			}
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Errorf("failed to close response body: %v", closeErr)
			}

			// test access to /api/test
			resp, err = client.Get(ts.URL + api + "/test")
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			body, err = io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			if string(body) != "test" {
				t.Errorf("expected 'test', got '%s'", body)
			}
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Errorf("failed to close response body: %v", closeErr)
			}

			// test POST request to /api
			req, err := http.NewRequest(http.MethodPost, ts.URL+api, http.NoBody)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			resp, err = client.Do(req)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405, got %d", resp.StatusCode)
			}
			if allow := resp.Header.Get("Allow"); !strings.Contains(allow, http.MethodGet) {
				t.Errorf("expected Allow header to contain GET, got %q", allow)
			}
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Errorf("failed to close response body: %v", closeErr)
			}
		}
	})

	t.Run("HandleRoot without method", func(t *testing.T) {
		group := routegroup.New(http.NewServeMux())
		group.Mount("/data").Route(func(dataGroup *routegroup.Bundle) {
			dataGroup.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Middleware", "applied")
					next.ServeHTTP(w, r)
				})
			})
			// register without specifying a method (empty string)
			dataGroup.HandleRoot("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte("data root")); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))
		})
		group.Mount("/data-2").Route(func(dataGroup *routegroup.Bundle) {
			dataGroup.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Middleware", "applied")
					next.ServeHTTP(w, r)
				})
			})
			// register without specifying a method (empty string)
			dataGroup.HandleRootFunc("", func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte("data root")); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})
		})

		ts := httptest.NewServer(group)
		defer ts.Close()

		paths := []string{"/data", "/data-2"}

		for _, path := range paths {
			// test GET request
			resp, err := client.Get(ts.URL + path)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			if resp.Header.Get("X-Middleware") != "applied" {
				t.Errorf("middleware not applied")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			if string(body) != "data root" {
				t.Errorf("expected 'data root', got '%s'", body)
			}
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Errorf("failed to close response body: %v", closeErr)
			}

			// test POST request - should also work since no method was specified
			req, err := http.NewRequest(http.MethodPost, ts.URL+path, http.NoBody)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			resp, err = client.Do(req)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			if resp.Header.Get("X-Middleware") != "applied" {
				t.Errorf("middleware not applied")
			}
			body, err = io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			if string(body) != "data root" {
				t.Errorf("expected 'data root', got '%s'", body)
			}
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Errorf("failed to close response body: %v", closeErr)
			}
		}
	})

	t.Run("HandleRoot with empty base path", func(t *testing.T) {
		// create a group with empty base path
		group := routegroup.New(http.NewServeMux())
		group.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware", "applied")
				next.ServeHTTP(w, r)
			})
		})

		// handle the root path (empty base path)
		group.HandleRoot("GET", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("root")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		ts := httptest.NewServer(group)
		defer ts.Close()

		// test GET request to root
		resp, err := client.Get(ts.URL + "/")
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if resp.Header.Get("X-Middleware") != "applied" {
			t.Errorf("middleware not applied")
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}
		if string(body) != "root" {
			t.Errorf("expected 'root', got '%s'", body)
		}
	})

	t.Run("HandleRootFunc with empty base path", func(t *testing.T) {
		// create a group with empty base path
		group := routegroup.New(http.NewServeMux())
		group.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware", "applied")
				next.ServeHTTP(w, r)
			})
		})

		// handle the root path (empty base path)
		group.HandleRootFunc("GET", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("root")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		})

		ts := httptest.NewServer(group)
		defer ts.Close()

		// test GET request to root
		resp, err := client.Get(ts.URL + "/")
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if resp.Header.Get("X-Middleware") != "applied" {
			t.Errorf("middleware not applied")
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}
		if string(body) != "root" {
			t.Errorf("expected 'root', got '%s'", body)
		}
	})

	t.Run("handle with trailing slash", func(t *testing.T) {
		group := routegroup.New(http.NewServeMux())
		apiGroup := group.Mount("/api")
		apiGroup.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("api root")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		})

		ts := httptest.NewServer(group)
		defer ts.Close()

		resp, err := client.Get(ts.URL + "/api")
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// verify trailing slash approach causes redirect
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Errorf("expected redirect status 301, got %d", resp.StatusCode)
		}

		location := resp.Header.Get("Location")
		if location != "/api/" {
			t.Errorf("expected redirect to '/api/', got '%s'", location)
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

// This example shows how to use HandleRoot to handle the root path of a mounted group without trailing slash
func ExampleBundle_HandleRoot() {
	group := routegroup.New(http.NewServeMux())

	// create API group
	apiGroup := group.Mount("/api")

	// apply middleware
	apiGroup.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-API", "true")
			next.ServeHTTP(w, r)
		})
	})

	// handle root path (responds to /api without redirect)
	apiGroup.HandleRoot("GET", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "API root")
	}))

	// regular routes
	apiGroup.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "List of users")
	})
}

// TestPathParametersWithMount tests path parameter extraction with mounted groups (issue #22)
func TestPathParametersWithMount(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func() *routegroup.Bundle
		requestPath    string
		expectedParam  string
		expectedStatus int
	}{
		{
			name: "path parameters with mount",
			setupFunc: func() *routegroup.Bundle {
				mux := http.NewServeMux()
				bundle := routegroup.Mount(mux, "/api/v0")
				peerGroup := bundle.Mount("/peer")
				peerGroup.HandleFunc("POST /iface/{iface}/multiplenew", func(w http.ResponseWriter, r *http.Request) {
					interfaceID := r.PathValue("iface")
					_, _ = w.Write([]byte("iface=" + interfaceID))
				})
				return bundle
			},
			requestPath:    "/api/v0/peer/iface/test123/multiplenew",
			expectedParam:  "iface=test123",
			expectedStatus: http.StatusOK,
		},
		{
			name: "path parameters with group",
			setupFunc: func() *routegroup.Bundle {
				mux := http.NewServeMux()
				bundle := routegroup.Mount(mux, "/api/v0")
				peerGroup := bundle.Group()
				peerGroup.HandleFunc("POST /peer/iface/{iface}/multiplenew", func(w http.ResponseWriter, r *http.Request) {
					interfaceID := r.PathValue("iface")
					_, _ = w.Write([]byte("iface=" + interfaceID))
				})
				return bundle
			},
			requestPath:    "/api/v0/peer/iface/test123/multiplenew",
			expectedParam:  "iface=test123",
			expectedStatus: http.StatusOK,
		},
		{
			name: "multiple path parameters",
			setupFunc: func() *routegroup.Bundle {
				mux := http.NewServeMux()
				bundle := routegroup.Mount(mux, "/api")
				bundle.HandleFunc("GET /users/{userID}/posts/{postID}", func(w http.ResponseWriter, r *http.Request) {
					userID := r.PathValue("userID")
					postID := r.PathValue("postID")
					_, _ = fmt.Fprintf(w, "user=%s,post=%s", userID, postID)
				})
				return bundle
			},
			requestPath:    "/api/users/alice/posts/42",
			expectedParam:  "user=alice,post=42",
			expectedStatus: http.StatusOK,
		},
		{
			name: "path parameters with middleware",
			setupFunc: func() *routegroup.Bundle {
				mux := http.NewServeMux()
				bundle := routegroup.Mount(mux, "/api")
				bundle.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("X-Middleware", "applied")
						next.ServeHTTP(w, r)
					})
				})
				bundle.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
					itemID := r.PathValue("id")
					_, _ = w.Write([]byte("item=" + itemID))
				})
				return bundle
			},
			requestPath:    "/api/items/xyz",
			expectedParam:  "item=xyz",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle := tt.setupFunc()

			method := "GET"
			if strings.Contains(tt.name, "mount") || strings.Contains(tt.name, "group") {
				method = "POST"
			}

			req := httptest.NewRequest(method, tt.requestPath, http.NoBody)
			rr := httptest.NewRecorder()

			bundle.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if rr.Body.String() != tt.expectedParam {
				t.Errorf("expected %q, got %q", tt.expectedParam, rr.Body.String())
			}
		})
	}
}

// TestRemainderWildcards tests the {path...} remainder wildcard feature
func TestRemainderWildcards(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		requestPath    string
		expectedParam  string
		expectedStatus int
	}{
		{
			name:           "single segment remainder",
			pattern:        "GET /files/{path...}",
			requestPath:    "/files/document.txt",
			expectedParam:  "document.txt",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "multiple segments remainder",
			pattern:        "GET /files/{path...}",
			requestPath:    "/files/docs/2024/report.pdf",
			expectedParam:  "docs/2024/report.pdf",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "remainder with mount",
			pattern:        "GET /static/{filepath...}",
			requestPath:    "/api/static/css/style.css",
			expectedParam:  "css/style.css",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty remainder",
			pattern:        "GET /files/{path...}",
			requestPath:    "/files/",
			expectedParam:  "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()

			if tt.name == "remainder with mount" {
				bundle := routegroup.Mount(mux, "/api")
				bundle.HandleFunc(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
					filepathParam := r.PathValue("filepath")
					_, _ = w.Write([]byte(filepathParam))
				})

				req := httptest.NewRequest("GET", tt.requestPath, http.NoBody)
				rr := httptest.NewRecorder()
				bundle.ServeHTTP(rr, req)

				if rr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
				}
				if rr.Body.String() != tt.expectedParam {
					t.Errorf("expected %q, got %q", tt.expectedParam, rr.Body.String())
				}
			} else {
				bundle := routegroup.New(mux)
				bundle.HandleFunc(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
					path := r.PathValue("path")
					_, _ = w.Write([]byte(path))
				})

				req := httptest.NewRequest("GET", tt.requestPath, http.NoBody)
				rr := httptest.NewRecorder()
				bundle.ServeHTTP(rr, req)

				if rr.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
				}
				if rr.Body.String() != tt.expectedParam {
					t.Errorf("expected %q, got %q", tt.expectedParam, rr.Body.String())
				}
			}
		})
	}
}

// TestMiddlewareWithContextAndPathParams tests that path params survive middleware context changes
func TestMiddlewareWithContextAndPathParams(t *testing.T) {
	type contextKey string
	const userKey contextKey = "user"
	type requestIDKey string
	const reqIDKey requestIDKey = "request-id"

	tests := []struct {
		name          string
		setupFunc     func() *routegroup.Bundle
		requestPath   string
		expectedParam string
		expectedUser  string
	}{
		{
			name: "WithContext preserves path params",
			setupFunc: func() *routegroup.Bundle {
				mux := http.NewServeMux()
				bundle := routegroup.New(mux)

				// middleware that adds context value using WithContext
				bundle.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						ctx := context.WithValue(r.Context(), userKey, "alice")
						next.ServeHTTP(w, r.WithContext(ctx))
					})
				})

				bundle.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
					userID := r.PathValue("id")
					user := r.Context().Value(userKey).(string)
					_, _ = fmt.Fprintf(w, "id=%s,user=%s", userID, user)
				})

				return bundle
			},
			requestPath:   "/users/123",
			expectedParam: "id=123,user=alice",
			expectedUser:  "alice",
		},
		{
			name: "Clone preserves path params",
			setupFunc: func() *routegroup.Bundle {
				mux := http.NewServeMux()
				bundle := routegroup.New(mux)

				// middleware that clones request with new context
				bundle.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						ctx := context.WithValue(r.Context(), userKey, "bob")
						newReq := r.Clone(ctx)
						next.ServeHTTP(w, newReq)
					})
				})

				bundle.HandleFunc("GET /items/{itemID}/details", func(w http.ResponseWriter, r *http.Request) {
					itemID := r.PathValue("itemID")
					user := r.Context().Value(userKey).(string)
					_, _ = fmt.Fprintf(w, "item=%s,user=%s", itemID, user)
				})

				return bundle
			},
			requestPath:   "/items/xyz/details",
			expectedParam: "item=xyz,user=bob",
			expectedUser:  "bob",
		},
		{
			name: "Multiple middleware with context changes",
			setupFunc: func() *routegroup.Bundle {
				mux := http.NewServeMux()
				bundle := routegroup.New(mux)

				// first middleware
				bundle.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						ctx := context.WithValue(r.Context(), reqIDKey, "req-123")
						next.ServeHTTP(w, r.WithContext(ctx))
					})
				})

				// second middleware
				bundle.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						ctx := context.WithValue(r.Context(), userKey, "charlie")
						next.ServeHTTP(w, r.WithContext(ctx))
					})
				})

				bundle.HandleFunc("POST /api/{version}/users/{id}", func(w http.ResponseWriter, r *http.Request) {
					version := r.PathValue("version")
					userID := r.PathValue("id")
					user := r.Context().Value(userKey).(string)
					reqID := r.Context().Value(reqIDKey).(string)
					_, _ = fmt.Fprintf(w, "v=%s,id=%s,user=%s,req=%s", version, userID, user, reqID)
				})

				return bundle
			},
			requestPath:   "/api/v2/users/456",
			expectedParam: "v=v2,id=456,user=charlie,req=req-123",
			expectedUser:  "charlie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle := tt.setupFunc()

			method := "GET"
			if strings.Contains(tt.requestPath, "/api/") {
				method = "POST"
			}

			req := httptest.NewRequest(method, tt.requestPath, http.NoBody)
			rr := httptest.NewRecorder()

			bundle.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status OK, got %d", rr.Code)
			}

			if rr.Body.String() != tt.expectedParam {
				t.Errorf("expected %q, got %q", tt.expectedParam, rr.Body.String())
			}
		})
	}
}

// TestURLEncodedPathParams tests that URL-encoded path parameters are properly decoded
func TestURLEncodedPathParams(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		requestPath    string
		expectedParam  string
		expectedStatus int
	}{
		{
			name:           "space encoded as %20",
			pattern:        "GET /users/{name}",
			requestPath:    "/users/John%20Doe",
			expectedParam:  "John Doe",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "slash encoded as %2F",
			pattern:        "GET /files/{filename}",
			requestPath:    "/files/folder%2Ffile.txt",
			expectedParam:  "folder/file.txt",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "special characters",
			pattern:        "GET /search/{query}",
			requestPath:    "/search/hello%3Dworld%26foo%3Dbar",
			expectedParam:  "hello=world&foo=bar",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unicode characters",
			pattern:        "GET /users/{name}",
			requestPath:    "/users/%E4%B8%AD%E6%96%87",
			expectedParam:  "中文",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "plus sign handling",
			pattern:        "GET /api/{version}",
			requestPath:    "/api/v1%2B2",
			expectedParam:  "v1+2",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "percent sign itself",
			pattern:        "GET /discount/{code}",
			requestPath:    "/discount/SAVE%2550",
			expectedParam:  "SAVE%50",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "encoded in remainder wildcard",
			pattern:        "GET /static/{path...}",
			requestPath:    "/static/images%2Flogo%20v2.png",
			expectedParam:  "images/logo v2.png",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			bundle := routegroup.New(mux)

			bundle.HandleFunc(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				var param string
				switch {
				case strings.Contains(tt.pattern, "{path...}"):
					param = r.PathValue("path")
				case strings.Contains(tt.pattern, "{name}"):
					param = r.PathValue("name")
				case strings.Contains(tt.pattern, "{filename}"):
					param = r.PathValue("filename")
				case strings.Contains(tt.pattern, "{query}"):
					param = r.PathValue("query")
				case strings.Contains(tt.pattern, "{version}"):
					param = r.PathValue("version")
				case strings.Contains(tt.pattern, "{code}"):
					param = r.PathValue("code")
				}
				_, _ = w.Write([]byte(param))
			})

			req := httptest.NewRequest("GET", tt.requestPath, http.NoBody)
			rr := httptest.NewRecorder()

			bundle.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if rr.Body.String() != tt.expectedParam {
				t.Errorf("expected %q, got %q", tt.expectedParam, rr.Body.String())
			}
		})
	}
}

// TestMethodlessPathParams tests path parameters without HTTP method prefix
func TestMethodlessPathParams(t *testing.T) {
	mux := http.NewServeMux()
	bundle := routegroup.New(mux)

	// pattern without method prefix - should work for all methods
	bundle.HandleFunc("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		_, _ = w.Write([]byte("item:" + id))
	})

	tests := []struct {
		method string
		path   string
		want   string
	}{
		{"GET", "/items/123", "item:123"},
		{"POST", "/items/456", "item:456"},
		{"PUT", "/items/789", "item:789"},
		{"DELETE", "/items/abc", "item:abc"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			rr := httptest.NewRecorder()

			bundle.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status OK, got %d", rr.Code)
			}
			if rr.Body.String() != tt.want {
				t.Errorf("expected %q, got %q", tt.want, rr.Body.String())
			}
		})
	}
}

// TestRootBundlePathParams tests path parameters on root bundle without Mount
func TestRootBundlePathParams(t *testing.T) {
	mux := http.NewServeMux()
	bundle := routegroup.New(mux)

	bundle.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		_, _ = w.Write([]byte("user:" + id))
	})

	req := httptest.NewRequest("GET", "/users/root123", http.NoBody)
	rr := httptest.NewRecorder()

	bundle.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", rr.Code)
	}
	if got := rr.Body.String(); got != "user:root123" {
		t.Errorf("expected user:root123, got %q", got)
	}
}

// TestHEADWithPathParams tests HEAD requests with path parameters
func TestHEADWithPathParams(t *testing.T) {
	mux := http.NewServeMux()
	bundle := routegroup.New(mux)

	// GET handler should also handle HEAD
	bundle.HandleFunc("GET /api/{version}/status", func(w http.ResponseWriter, r *http.Request) {
		version := r.PathValue("version")
		w.Header().Set("X-Version", version)
		w.Header().Set("Content-Length", "10")
		if r.Method != "HEAD" {
			_, _ = w.Write([]byte("status:ok"))
		}
	})

	// test HEAD request
	req := httptest.NewRequest("HEAD", "/api/v2/status", http.NoBody)
	rr := httptest.NewRecorder()

	bundle.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", rr.Code)
	}
	if got := rr.Header().Get("X-Version"); got != "v2" {
		t.Errorf("expected X-Version header v2, got %q", got)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("HEAD response should have no body, got %d bytes", rr.Body.Len())
	}
}
