package routegroup_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-pkgz/routegroup"
)

func TestPathParametersWithMount(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupFunc      func() *routegroup.Bundle
		requestPath    string
		expectedParam  string
		expectedStatus int
	}{
		{
			name:   "path parameters with mount",
			method: "POST",
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
			name:   "path parameters with group",
			method: "POST",
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
			name:   "multiple path parameters",
			method: "GET",
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
			name:   "path parameters with middleware",
			method: "GET",
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
			req := httptest.NewRequest(tt.method, tt.requestPath, http.NoBody)
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
		method        string
		setupFunc     func() *routegroup.Bundle
		requestPath   string
		expectedParam string
		expectedUser  string
	}{
		{
			name:   "WithContext preserves path params",
			method: "GET",
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
			name:   "Clone preserves path params",
			method: "GET",
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
			name:   "Multiple middleware with context changes",
			method: "POST",
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
			req := httptest.NewRequest(tt.method, tt.requestPath, http.NoBody)
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
