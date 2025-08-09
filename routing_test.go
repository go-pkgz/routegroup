package routegroup_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-pkgz/routegroup"
)

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
