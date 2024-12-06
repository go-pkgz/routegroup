package routegroup_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
