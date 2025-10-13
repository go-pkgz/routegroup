package routegroup_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestMountedBundleServeHTTP(t *testing.T) {
	// test ServeHTTP when called directly on a mounted bundle
	root := routegroup.New(http.NewServeMux())
	root.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Root-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	mounted := root.Mount("/api")
	mounted.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("mounted handler"))
	})

	// serve directly from the mounted bundle (not typical usage but should work)
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	mounted.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
	if body := recorder.Body.String(); body != "mounted handler" {
		t.Errorf("Expected body 'mounted handler', got '%s'", body)
	}
	// should still apply root middleware when serving from mounted
	if header := recorder.Header().Get("X-Root-Middleware"); header != "true" {
		t.Errorf("Expected X-Root-Middleware header to be 'true', got '%s'", header)
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
		// with auto-wrapping, middleware is in a sub-group and doesn't apply to 405s
		if header := recorder.Header().Get("X-Test-Middleware"); header != "" {
			t.Errorf("Expected header X-Test-Middleware to be empty for 405 (group middleware), got '%s'", header)
		}
	})
}

func TestGroupRouteAutoWrapping(t *testing.T) {
	// test that calling Route on root bundle auto-creates a group
	router := routegroup.New(http.NewServeMux())

	// add middleware to router first
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Root-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	})

	// calling Route on root should auto-create a group
	router.Route(func(g *routegroup.Bundle) {
		g.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Group-Middleware", "true")
				next.ServeHTTP(w, r)
			})
		})
		g.HandleFunc("/grouped", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("grouped handler"))
		})
	})

	// add another route directly to root
	router.HandleFunc("/root", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("root handler"))
	})

	t.Run("grouped route has both middlewares", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/grouped", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if body := recorder.Body.String(); body != "grouped handler" {
			t.Errorf("Expected body 'grouped handler', got '%s'", body)
		}
		if header := recorder.Header().Get("X-Root-Middleware"); header != "true" {
			t.Errorf("Expected X-Root-Middleware to be 'true', got '%s'", header)
		}
		if header := recorder.Header().Get("X-Group-Middleware"); header != "true" {
			t.Errorf("Expected X-Group-Middleware to be 'true', got '%s'", header)
		}
	})

	t.Run("root route only has root middleware", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/root", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
		if body := recorder.Body.String(); body != "root handler" {
			t.Errorf("Expected body 'root handler', got '%s'", body)
		}
		if header := recorder.Header().Get("X-Root-Middleware"); header != "true" {
			t.Errorf("Expected X-Root-Middleware to be 'true', got '%s'", header)
		}
		if header := recorder.Header().Get("X-Group-Middleware"); header != "" {
			t.Errorf("Expected X-Group-Middleware to be empty, got '%s'", header)
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

	t.Run("root: Use after Route() with auto-wrap panics", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		router.Route(func(b *routegroup.Bundle) {
			b.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {})
		})
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic on root.Use after Route() registered routes")
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
