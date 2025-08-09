package routegroup_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-pkgz/routegroup"
)

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
