package routegroup_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-pkgz/routegroup"
)

func TestMiddlewareCanAccessPathValues(t *testing.T) {
	// test path value accessibility in middlewares
	// EXPECTED: root/global middlewares can't see PathValue (runs before routing)
	// EXPECTED: mounted group middlewares CAN see PathValue (applied at registration)
	tests := []struct {
		name           string
		setupFunc      func() *routegroup.Bundle
		requestPath    string
		expectedID     string
		expectedUser   string
	}{
		{
			name: "root middleware cannot access path params (expected)",
			setupFunc: func() *routegroup.Bundle {
				rtr := routegroup.New(http.NewServeMux())
				
				// root middleware runs BEFORE mux.ServeHTTP sets path values
				rtr.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						// PathValue is empty here - this is EXPECTED
						id := r.PathValue("id")
						w.Header().Set("X-Root-Middleware-ID", id) // will be empty
						
						// but Pattern IS available (our fix from #24)
						w.Header().Set("X-Root-Pattern", r.Pattern)
						next.ServeHTTP(w, r)
					})
				})
				
				rtr.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
					// handler CAN access path values
					w.Header().Set("X-Handler-ID", r.PathValue("id"))
					w.WriteHeader(http.StatusOK)
				})
				
				return rtr
			},
			requestPath: "/users/123",
			expectedID:  "", // empty in root middleware is EXPECTED
		},
		{
			name: "mounted group with path params",
			setupFunc: func() *routegroup.Bundle {
				rtr := routegroup.New(http.NewServeMux())
				
				api := rtr.Mount("/api")
				api.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						// path values should be accessible in mounted group middleware
						id := r.PathValue("id")
						user := r.PathValue("user")
						if id != "" {
							w.Header().Set("X-Middleware-ID", id)
						}
						if user != "" {
							w.Header().Set("X-Middleware-User", user)
						}
						next.ServeHTTP(w, r)
					})
				})
				
				api.HandleFunc("GET /users/{user}/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
				
				return rtr
			},
			requestPath:  "/api/users/john/posts/456",
			expectedID:   "456",
			expectedUser: "john",
		},
		{
			name: "nested mounted groups with params",
			setupFunc: func() *routegroup.Bundle {
				rtr := routegroup.New(http.NewServeMux())
				
				v1 := rtr.Mount("/v1")
				users := v1.Mount("/users")
				
				users.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						id := r.PathValue("id")
						action := r.PathValue("action")
						if id != "" {
							w.Header().Set("X-Middleware-ID", id)
						}
						if action != "" {
							w.Header().Set("X-Middleware-Action", action)
						}
						next.ServeHTTP(w, r)
					})
				})
				
				users.HandleFunc("GET /{id}/{action}", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
				
				return rtr
			},
			requestPath: "/v1/users/789/edit",
			expectedID:  "789",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle := tt.setupFunc()
			
			req, err := http.NewRequest(http.MethodGet, tt.requestPath, http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			
			rec := httptest.NewRecorder()
			bundle.ServeHTTP(rec, req)
			
			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rec.Code)
			}
			
			// verify path value accessibility based on middleware type
			if tt.name == "root middleware cannot access path params (expected)" {
				// verify root middleware can't see path values
				if got := rec.Header().Get("X-Root-Middleware-ID"); got != "" {
					t.Errorf("root middleware should not see path values, got %q", got)
				}
				// but handler can
				if got := rec.Header().Get("X-Handler-ID"); got != "123" {
					t.Errorf("handler should see path value, got %q", got)
				}
				// and Pattern is available (from our fix)
				if got := rec.Header().Get("X-Root-Pattern"); got != "GET /users/{id}" {
					t.Errorf("root middleware should see pattern, got %q", got)
				}
			} else {
				// mounted group middlewares CAN see path values
				if tt.expectedID != "" {
					if got := rec.Header().Get("X-Middleware-ID"); got != tt.expectedID {
						t.Errorf("middleware ID = %q, want %q", got, tt.expectedID)
					}
				}
				
				if tt.expectedUser != "" {
					if got := rec.Header().Get("X-Middleware-User"); got != tt.expectedUser {
						t.Errorf("middleware User = %q, want %q", got, tt.expectedUser)
					}
				}
			}
		})
	}
}

func TestMiddlewareAbortChain(t *testing.T) {
	// test that middleware can stop the chain by not calling next.ServeHTTP()
	// this is critical for auth/security middleware
	
	t.Run("auth middleware aborts on unauthorized", func(t *testing.T) {
		handlerCalled := false
		middleware2Called := false
		
		rtr := routegroup.New(http.NewServeMux())
		
		// first middleware - auth check that aborts
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Auth-Checked", "true")
				
				if r.Header.Get("Authorization") == "" {
					// abort chain - don't call next
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte("unauthorized"))
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		
		// second middleware - should not be called if first aborts
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				middleware2Called = true
				w.Header().Set("X-Middleware2", "called")
				next.ServeHTTP(w, r)
			})
		})
		
		rtr.HandleFunc("GET /protected", func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("protected content"))
		})
		
		// test unauthorized request
		req, _ := http.NewRequest(http.MethodGet, "/protected", http.NoBody)
		rec := httptest.NewRecorder()
		
		rtr.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
		if rec.Body.String() != "unauthorized" {
			t.Errorf("expected 'unauthorized', got %q", rec.Body.String())
		}
		if handlerCalled {
			t.Error("handler should not be called when middleware aborts")
		}
		if middleware2Called {
			t.Error("second middleware should not be called when first aborts")
		}
		if rec.Header().Get("X-Auth-Checked") != "true" {
			t.Error("first middleware should have run")
		}
		if rec.Header().Get("X-Middleware2") != "" {
			t.Error("second middleware should not have set header")
		}
	})
	
	t.Run("middleware abort with mounted groups", func(t *testing.T) {
		handlerCalled := false
		
		rtr := routegroup.New(http.NewServeMux())
		
		// root middleware - always passes
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Root", "passed")
				next.ServeHTTP(w, r)
			})
		})
		
		api := rtr.Mount("/api")
		
		// api middleware - aborts on missing API key
		api.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-API-Check", "true")
				
				if r.Header.Get("X-API-Key") == "" {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("API key required"))
					return // abort
				}
				next.ServeHTTP(w, r)
			})
		})
		
		api.HandleFunc("GET /data", func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("data"))
		})
		
		// test without API key
		req, _ := http.NewRequest(http.MethodGet, "/api/data", http.NoBody)
		rec := httptest.NewRecorder()
		
		rtr.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
		if rec.Body.String() != "API key required" {
			t.Errorf("expected 'API key required', got %q", rec.Body.String())
		}
		if handlerCalled {
			t.Error("handler should not be called when middleware aborts")
		}
		// root middleware should have run
		if rec.Header().Get("X-Root") != "passed" {
			t.Error("root middleware should have run")
		}
		// api middleware should have run and aborted
		if rec.Header().Get("X-API-Check") != "true" {
			t.Error("API middleware should have run")
		}
	})
	
	t.Run("middleware chain continues with authorization", func(t *testing.T) {
		handlerCalled := false
		
		rtr := routegroup.New(http.NewServeMux())
		
		// auth middleware that passes with correct header
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Auth-Checked", "true")
				
				if r.Header.Get("Authorization") == "" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		
		// second middleware
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware2", "called")
				next.ServeHTTP(w, r)
			})
		})
		
		rtr.HandleFunc("GET /protected", func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("protected content"))
		})
		
		// test authorized request
		req, _ := http.NewRequest(http.MethodGet, "/protected", http.NoBody)
		req.Header.Set("Authorization", "Bearer token")
		rec := httptest.NewRecorder()
		
		rtr.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Body.String() != "protected content" {
			t.Errorf("expected 'protected content', got %q", rec.Body.String())
		}
		if !handlerCalled {
			t.Error("handler should be called with authorization")
		}
		if rec.Header().Get("X-Auth-Checked") != "true" {
			t.Error("auth middleware should have run")
		}
		if rec.Header().Get("X-Middleware2") != "called" {
			t.Error("second middleware should have run")
		}
	})
}

func TestWithMethodMiddlewareCounting(t *testing.T) {
	// test that With() properly tracks middleware count to avoid double execution
	// this is critical to prevent issues like #24
	
	t.Run("With() creates new bundle with correct middleware count", func(t *testing.T) {
		callCounts := make(map[string]int)
		
		rtr := routegroup.New(http.NewServeMux())
		
		// root middleware
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCounts["root"]++
				w.Header().Set("X-MW-Order", "root")
				next.ServeHTTP(w, r)
			})
		})
		
		// create new bundle with additional middleware using With()
		withBundle := rtr.With(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCounts["with"]++
				existing := w.Header().Get("X-MW-Order")
				w.Header().Set("X-MW-Order", existing+",with")
				next.ServeHTTP(w, r)
			})
		})
		
		// register route on the With bundle
		withBundle.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
			callCounts["handler"]++
			w.WriteHeader(http.StatusOK)
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		rec := httptest.NewRecorder()
		
		rtr.ServeHTTP(rec, req)
		
		// verify each middleware called exactly once
		if callCounts["root"] != 1 {
			t.Errorf("root middleware called %d times, expected 1", callCounts["root"])
		}
		if callCounts["with"] != 1 {
			t.Errorf("with middleware called %d times, expected 1", callCounts["with"])
		}
		if callCounts["handler"] != 1 {
			t.Errorf("handler called %d times, expected 1", callCounts["handler"])
		}
		
		// verify order
		if order := rec.Header().Get("X-MW-Order"); order != "root,with" {
			t.Errorf("middleware order = %q, expected 'root,with'", order)
		}
	})
	
	t.Run("multiple With() calls maintain proper count", func(t *testing.T) {
		callCounts := make(map[string]int)
		
		rtr := routegroup.New(http.NewServeMux())
		
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCounts["root"]++
				next.ServeHTTP(w, r)
			})
		})
		
		// first With()
		bundle1 := rtr.With(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCounts["with1"]++
				next.ServeHTTP(w, r)
			})
		})
		
		// second With() on top of first
		bundle2 := bundle1.With(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCounts["with2"]++
				next.ServeHTTP(w, r)
			})
		})
		
		bundle2.HandleFunc("GET /nested", func(w http.ResponseWriter, r *http.Request) {
			callCounts["handler"]++
			w.WriteHeader(http.StatusOK)
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/nested", http.NoBody)
		rec := httptest.NewRecorder()
		
		rtr.ServeHTTP(rec, req)
		
		// all middlewares should be called exactly once
		if callCounts["root"] != 1 {
			t.Errorf("root middleware called %d times, expected 1", callCounts["root"])
		}
		if callCounts["with1"] != 1 {
			t.Errorf("with1 middleware called %d times, expected 1", callCounts["with1"])
		}
		if callCounts["with2"] != 1 {
			t.Errorf("with2 middleware called %d times, expected 1", callCounts["with2"])
		}
		if callCounts["handler"] != 1 {
			t.Errorf("handler called %d times, expected 1", callCounts["handler"])
		}
	})
	
	t.Run("With() on mounted group maintains correct count", func(t *testing.T) {
		callCounts := make(map[string]int)
		
		rtr := routegroup.New(http.NewServeMux())
		
		// root middleware
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCounts["root"]++
				next.ServeHTTP(w, r)
			})
		})
		
		// mount a group
		api := rtr.Mount("/api")
		
		// add middleware to mounted group
		api.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCounts["api"]++
				next.ServeHTTP(w, r)
			})
		})
		
		// use With() on mounted group
		apiWith := api.With(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCounts["api-with"]++
				next.ServeHTTP(w, r)
			})
		})
		
		apiWith.HandleFunc("GET /data", func(w http.ResponseWriter, r *http.Request) {
			callCounts["handler"]++
			w.WriteHeader(http.StatusOK)
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/api/data", http.NoBody)
		rec := httptest.NewRecorder()
		
		rtr.ServeHTTP(rec, req)
		
		// verify no double execution
		if callCounts["root"] != 1 {
			t.Errorf("root middleware called %d times, expected 1", callCounts["root"])
		}
		if callCounts["api"] != 1 {
			t.Errorf("api middleware called %d times, expected 1", callCounts["api"])
		}
		if callCounts["api-with"] != 1 {
			t.Errorf("api-with middleware called %d times, expected 1", callCounts["api-with"])
		}
		if callCounts["handler"] != 1 {
			t.Errorf("handler called %d times, expected 1", callCounts["handler"])
		}
	})
}

func TestResponseWriterInterception(t *testing.T) {
	// test that middlewares can intercept and modify responses
	// this is critical for logging, metrics, response manipulation
	
	t.Run("middleware can capture status code", func(t *testing.T) {
		var capturedStatus int
		
		rtr := routegroup.New(http.NewServeMux())
		
		// status capturing middleware
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// wrap response writer to capture status
				wrapped := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
				next.ServeHTTP(wrapped, r)
				capturedStatus = wrapped.status
			})
		})
		
		rtr.HandleFunc("GET /success", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created"))
		})
		
		rtr.HandleFunc("GET /error", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("error"))
		})
		
		// test success response
		req1, _ := http.NewRequest(http.MethodGet, "/success", http.NoBody)
		rec1 := httptest.NewRecorder()
		rtr.ServeHTTP(rec1, req1)
		
		if capturedStatus != http.StatusCreated {
			t.Errorf("captured status = %d, want %d", capturedStatus, http.StatusCreated)
		}
		if rec1.Code != http.StatusCreated {
			t.Errorf("response status = %d, want %d", rec1.Code, http.StatusCreated)
		}
		
		// test error response
		req2, _ := http.NewRequest(http.MethodGet, "/error", http.NoBody)
		rec2 := httptest.NewRecorder()
		rtr.ServeHTTP(rec2, req2)
		
		if capturedStatus != http.StatusInternalServerError {
			t.Errorf("captured status = %d, want %d", capturedStatus, http.StatusInternalServerError)
		}
	})
	
	t.Run("middleware can modify response body", func(t *testing.T) {
		rtr := routegroup.New(http.NewServeMux())
		
		// response modifying middleware
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// capture response in buffer
				buf := &responseBuffer{ResponseWriter: w, buffer: []byte{}}
				next.ServeHTTP(buf, r)
				
				// modify and write actual response
				modified := append([]byte("PREFIX:"), buf.buffer...)
				_, _ = w.Write(modified)
			})
		})
		
		rtr.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("original"))
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		rec := httptest.NewRecorder()
		rtr.ServeHTTP(rec, req)
		
		if body := rec.Body.String(); body != "PREFIX:original" {
			t.Errorf("body = %q, want %q", body, "PREFIX:original")
		}
	})
	
	t.Run("multiple middlewares can wrap response writer", func(t *testing.T) {
		var statuses []int
		
		rtr := routegroup.New(http.NewServeMux())
		
		// first middleware - captures status
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wrapped := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
				next.ServeHTTP(wrapped, r)
				statuses = append(statuses, wrapped.status)
			})
		})
		
		// second middleware - also captures status
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wrapped := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
				next.ServeHTTP(wrapped, r)
				statuses = append(statuses, wrapped.status)
			})
		})
		
		rtr.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		rec := httptest.NewRecorder()
		
		statuses = nil
		rtr.ServeHTTP(rec, req)
		
		// both middlewares should capture the same status
		if len(statuses) != 2 {
			t.Errorf("expected 2 status captures, got %d", len(statuses))
		}
		if len(statuses) == 2 {
			if statuses[0] != http.StatusAccepted {
				t.Errorf("first middleware captured %d, want %d", statuses[0], http.StatusAccepted)
			}
			if statuses[1] != http.StatusAccepted {
				t.Errorf("second middleware captured %d, want %d", statuses[1], http.StatusAccepted)
			}
		}
	})
}

// statusRecorder wraps ResponseWriter to capture status code
type statusRecorder struct {
	http.ResponseWriter
	status int
	written bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.written {
		r.status = status
		r.written = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.written {
		r.written = true
		// status remains default (200) if WriteHeader wasn't called
	}
	return r.ResponseWriter.Write(b)
}

// responseBuffer captures response body
type responseBuffer struct {
	http.ResponseWriter
	buffer []byte
}

func (b *responseBuffer) Write(data []byte) (int, error) {
	b.buffer = append(b.buffer, data...)
	return len(data), nil
}

func TestContextPropagation(t *testing.T) {
	// test that context values propagate through middleware chain
	// this is critical for request IDs, user auth, tracing, etc.
	
	type contextKey string
	const (
		requestIDKey contextKey = "requestID"
		userIDKey    contextKey = "userID"
		traceKey     contextKey = "trace"
	)
	
	t.Run("context values propagate through chain", func(t *testing.T) {
		rtr := routegroup.New(http.NewServeMux())
		
		// first middleware - adds request ID
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), requestIDKey, "req-123")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		
		// second middleware - adds user ID and verifies request ID
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// verify request ID from first middleware
				if reqID := r.Context().Value(requestIDKey); reqID != "req-123" {
					t.Errorf("middleware 2: requestID = %v, want 'req-123'", reqID)
				}
				
				ctx := context.WithValue(r.Context(), userIDKey, "user-456")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		
		// handler - verifies both values
		rtr.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Context().Value(requestIDKey)
			userID := r.Context().Value(userIDKey)
			
			if reqID != "req-123" {
				t.Errorf("handler: requestID = %v, want 'req-123'", reqID)
			}
			if userID != "user-456" {
				t.Errorf("handler: userID = %v, want 'user-456'", userID)
			}
			
			w.WriteHeader(http.StatusOK)
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		rec := httptest.NewRecorder()
		rtr.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
	
	t.Run("context cancellation stops chain", func(t *testing.T) {
		handlerCalled := false
		middleware2Called := false
		
		rtr := routegroup.New(http.NewServeMux())
		
		// first middleware - cancels context
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx, cancel := context.WithCancel(r.Context())
				cancel() // immediately cancel
				
				// check if context is done before calling next
				select {
				case <-ctx.Done():
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				default:
					next.ServeHTTP(w, r.WithContext(ctx))
				}
			})
		})
		
		// second middleware - should not be called
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				middleware2Called = true
				next.ServeHTTP(w, r)
			})
		})
		
		rtr.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		rec := httptest.NewRecorder()
		rtr.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if middleware2Called {
			t.Error("middleware 2 should not be called after context cancellation")
		}
		if handlerCalled {
			t.Error("handler should not be called after context cancellation")
		}
	})
	
	t.Run("context values work with mounted groups", func(t *testing.T) {
		rtr := routegroup.New(http.NewServeMux())
		
		// root middleware - adds trace ID
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), traceKey, "trace-root")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		
		api := rtr.Mount("/api")
		
		// api middleware - adds request ID and checks trace
		api.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// verify trace from root
				if trace := r.Context().Value(traceKey); trace != "trace-root" {
					t.Errorf("api middleware: trace = %v, want 'trace-root'", trace)
				}
				
				ctx := context.WithValue(r.Context(), requestIDKey, "req-api")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		
		api.HandleFunc("GET /data", func(w http.ResponseWriter, r *http.Request) {
			trace := r.Context().Value(traceKey)
			reqID := r.Context().Value(requestIDKey)
			
			if trace != "trace-root" {
				t.Errorf("handler: trace = %v, want 'trace-root'", trace)
			}
			if reqID != "req-api" {
				t.Errorf("handler: requestID = %v, want 'req-api'", reqID)
			}
			
			w.WriteHeader(http.StatusOK)
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/api/data", http.NoBody)
		rec := httptest.NewRecorder()
		rtr.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestMiddlewareModifiesRequest(t *testing.T) {
	// test that middlewares can modify request properties
	// this is critical for adding headers, modifying paths, etc.
	
	t.Run("middleware can add and modify headers", func(t *testing.T) {
		var capturedHeaders http.Header
		
		rtr := routegroup.New(http.NewServeMux())
		
		// first middleware - adds header
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.Header.Set("X-Request-ID", "req-123")
				r.Header.Set("X-Custom", "value1")
				next.ServeHTTP(w, r)
			})
		})
		
		// second middleware - modifies header
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// verify first middleware's header
				if reqID := r.Header.Get("X-Request-ID"); reqID != "req-123" {
					t.Errorf("middleware 2: X-Request-ID = %q, want 'req-123'", reqID)
				}
				
				// modify existing header
				r.Header.Set("X-Custom", "value2")
				r.Header.Set("X-Middleware-2", "processed")
				next.ServeHTTP(w, r)
			})
		})
		
		rtr.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
			capturedHeaders = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.Header.Set("X-Original", "client-value")
		rec := httptest.NewRecorder()
		
		rtr.ServeHTTP(rec, req)
		
		// verify headers in handler
		if capturedHeaders.Get("X-Original") != "client-value" {
			t.Errorf("X-Original = %q, want 'client-value'", capturedHeaders.Get("X-Original"))
		}
		if capturedHeaders.Get("X-Request-ID") != "req-123" {
			t.Errorf("X-Request-ID = %q, want 'req-123'", capturedHeaders.Get("X-Request-ID"))
		}
		if capturedHeaders.Get("X-Custom") != "value2" {
			t.Errorf("X-Custom = %q, want 'value2' (should be modified)", capturedHeaders.Get("X-Custom"))
		}
		if capturedHeaders.Get("X-Middleware-2") != "processed" {
			t.Errorf("X-Middleware-2 = %q, want 'processed'", capturedHeaders.Get("X-Middleware-2"))
		}
	})
	
	t.Run("middleware can modify URL path", func(t *testing.T) {
		var capturedPath string
		
		rtr := routegroup.New(http.NewServeMux())
		
		// middleware that modifies URL
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// create a new URL with modified path
				newURL := *r.URL
				newURL.Path = "/modified" + r.URL.Path
				
				// create new request with modified URL
				r2 := r.Clone(r.Context())
				r2.URL = &newURL
				
				next.ServeHTTP(w, r2)
			})
		})
		
		// register handler for modified path
		rtr.HandleFunc("GET /modified/original", func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/original", http.NoBody)
		rec := httptest.NewRecorder()
		
		rtr.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if capturedPath != "/modified/original" {
			t.Errorf("captured path = %q, want '/modified/original'", capturedPath)
		}
	})
	
	t.Run("middleware modifications work with mounted groups", func(t *testing.T) {
		var handlerHeaders http.Header
		
		rtr := routegroup.New(http.NewServeMux())
		
		// root middleware - adds base header
		rtr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.Header.Set("X-Root", "root-value")
				next.ServeHTTP(w, r)
			})
		})
		
		api := rtr.Mount("/api")
		
		// api middleware - adds API header
		api.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// verify root header exists
				if root := r.Header.Get("X-Root"); root != "root-value" {
					t.Errorf("api middleware: X-Root = %q, want 'root-value'", root)
				}
				
				r.Header.Set("X-API", "api-value")
				r.Header.Set("X-API-Version", "v1")
				next.ServeHTTP(w, r)
			})
		})
		
		api.HandleFunc("GET /data", func(w http.ResponseWriter, r *http.Request) {
			handlerHeaders = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
		})
		
		req, _ := http.NewRequest(http.MethodGet, "/api/data", http.NoBody)
		rec := httptest.NewRecorder()
		
		rtr.ServeHTTP(rec, req)
		
		// verify all headers made it to handler
		if handlerHeaders.Get("X-Root") != "root-value" {
			t.Errorf("X-Root = %q, want 'root-value'", handlerHeaders.Get("X-Root"))
		}
		if handlerHeaders.Get("X-API") != "api-value" {
			t.Errorf("X-API = %q, want 'api-value'", handlerHeaders.Get("X-API"))
		}
		if handlerHeaders.Get("X-API-Version") != "v1" {
			t.Errorf("X-API-Version = %q, want 'v1'", handlerHeaders.Get("X-API-Version"))
		}
	})
}

func TestRequestPatternAndMiddlewareCallCount(t *testing.T) {
	// regression test for issue #24 - verify that:
	// 1. middlewares are not executed twice
	// 2. Request.Pattern is available in global middlewares

	var callCount map[string]int
	var mu sync.Mutex

	patternLogger := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			callCount[r.URL.Path]++
			patternBefore := r.Pattern
			mu.Unlock()

			next.ServeHTTP(w, r)

			mu.Lock()
			patternAfter := r.Pattern
			mu.Unlock()

			// verify pattern is set before calling next handler
			if patternBefore == "" {
				t.Errorf("pattern should be set before ServeHTTP, got empty for path %s", r.URL.Path)
			}
			// verify pattern remains consistent
			if patternBefore != patternAfter {
				t.Errorf("pattern changed from %q to %q for path %s", patternBefore, patternAfter, r.URL.Path)
			}
		})
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		id := r.PathValue("id")
		if id != "" {
			_, _ = w.Write([]byte("id: " + id))
		}
	}

	t.Run("root group with path params", func(t *testing.T) {
		callCount = make(map[string]int)
		rtr := routegroup.New(http.NewServeMux())
		rtr.Use(patternLogger)
		rtr.HandleFunc("GET /a/{id}", handler)

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/a/123", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}

		rtr.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", recorder.Code)
		}
		if body := recorder.Body.String(); body != "id: 123" {
			t.Errorf("expected 'id: 123', got %q", body)
		}

		// verify middleware was called exactly once
		if count := callCount["/a/123"]; count != 1 {
			t.Errorf("middleware should be called exactly once, but was called %d times", count)
		}
	})

	t.Run("mounted group with path params", func(t *testing.T) {
		callCount = make(map[string]int)
		rtr := routegroup.New(http.NewServeMux())
		rtr.Use(patternLogger)

		bGroup := rtr.Mount("/b")
		bGroup.HandleFunc("GET /{id}", handler)

		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/b/456", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}

		rtr.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", recorder.Code)
		}
		if body := recorder.Body.String(); body != "id: 456" {
			t.Errorf("expected 'id: 456', got %q", body)
		}

		// verify middleware was called exactly once
		if count := callCount["/b/456"]; count != 1 {
			t.Errorf("middleware should be called exactly once for mounted path, but was called %d times", count)
		}
	})

	t.Run("multiple middlewares see pattern", func(t *testing.T) {
		callCount = make(map[string]int)
		var patterns []string
		var mu2 sync.Mutex
		
		middleware1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				callCount["m1"]++
				mu.Unlock()
				
				mu2.Lock()
				patterns = append(patterns, "m1:"+r.Pattern)
				mu2.Unlock()
				
				next.ServeHTTP(w, r)
			})
		}
		
		middleware2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				callCount["m2"]++
				mu.Unlock()
				
				mu2.Lock()
				patterns = append(patterns, "m2:"+r.Pattern)
				mu2.Unlock()
				
				next.ServeHTTP(w, r)
			})
		}
		
		rtr := routegroup.New(http.NewServeMux())
		rtr.Use(middleware1, middleware2)
		rtr.HandleFunc("GET /test/{id}", handler)
		
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/test/789", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		
		rtr.ServeHTTP(recorder, request)
		
		// verify each middleware called once
		if count := callCount["m1"]; count != 1 {
			t.Errorf("middleware1 should be called once, got %d", count)
		}
		if count := callCount["m2"]; count != 1 {
			t.Errorf("middleware2 should be called once, got %d", count)
		}
		
		// verify both middlewares saw the pattern
		if len(patterns) != 2 {
			t.Errorf("expected 2 pattern records, got %d", len(patterns))
		}
		if len(patterns) == 2 {
			if patterns[0] != "m1:GET /test/{id}" {
				t.Errorf("middleware1 pattern = %q, want %q", patterns[0], "m1:GET /test/{id}")
			}
			if patterns[1] != "m2:GET /test/{id}" {
				t.Errorf("middleware2 pattern = %q, want %q", patterns[1], "m2:GET /test/{id}")
			}
		}
	})

	t.Run("route without path params", func(t *testing.T) {
		callCount = make(map[string]int)
		var seenPattern string
		
		checkPattern := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				callCount[r.URL.Path]++
				seenPattern = r.Pattern
				mu.Unlock()
				next.ServeHTTP(w, r)
			})
		}
		
		rtr := routegroup.New(http.NewServeMux())
		rtr.Use(checkPattern)
		rtr.HandleFunc("GET /static", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("static"))
		})
		
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/static", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		
		rtr.ServeHTTP(recorder, request)
		
		if count := callCount["/static"]; count != 1 {
			t.Errorf("middleware should be called once for static route, got %d", count)
		}
		
		if seenPattern != "GET /static" {
			t.Errorf("pattern = %q, want %q", seenPattern, "GET /static")
		}
	})
}

// TestRequestIsolation verifies that the original request passed to ServeHTTP
// is not modified, ensuring proper isolation through shallow copy.
func TestRequestIsolation(t *testing.T) {
	t.Run("original request not modified", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		
		var middlewareRequest *http.Request
		
		// middleware that captures the request object
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				middlewareRequest = r
				next.ServeHTTP(w, r)
			})
		})
		
		router.HandleFunc("GET /test/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		
		// create the original request
		originalRequest, err := http.NewRequest(http.MethodGet, "/test/123", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		
		// save original state
		originalPattern := originalRequest.Pattern
		
		// make the request
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, originalRequest)
		
		// verify the original request was not modified
		if originalRequest.Pattern != originalPattern {
			t.Errorf("original request was modified: Pattern changed from %q to %q", 
				originalPattern, originalRequest.Pattern)
		}
		
		// verify middleware received a different request object (shallow copy)
		if middlewareRequest == originalRequest {
			t.Error("middleware received the same request object (expected a copy)")
		}
		
		// verify middleware's request has the pattern set
		if middlewareRequest.Pattern == "" {
			t.Error("middleware's request should have Pattern set")
		}
	})
	
	t.Run("isolation with 404", func(t *testing.T) {
		router := routegroup.New(http.NewServeMux())
		
		router.NotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware", "ran")
				next.ServeHTTP(w, r)
			})
		})
		
		originalRequest, err := http.NewRequest(http.MethodGet, "/non-existent", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		
		originalPattern := originalRequest.Pattern
		
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, originalRequest)
		
		if recorder.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recorder.Code)
		}
		
		// original request should not be modified
		if originalRequest.Pattern != originalPattern {
			t.Errorf("original request was modified: Pattern changed from %q to %q", 
				originalPattern, originalRequest.Pattern)
		}
	})
}