package routegroup_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-pkgz/routegroup"
)

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
