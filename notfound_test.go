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

func TestNotFoundHandlerOnMountedGroup(t *testing.T) {
	// test that NotFoundHandler sets handler on root when called on mounted group
	root := routegroup.New(http.NewServeMux())
	mounted := root.Mount("/api")

	// set NotFoundHandler on mounted group - should set it on root
	mounted.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Custom 404 from mounted", http.StatusNotFound)
	})

	// add a route to the mounted group
	mounted.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test"))
	})

	testServer := httptest.NewServer(root)
	defer testServer.Close()

	// test that custom 404 works for non-matching routes
	resp, err := http.Get(testServer.URL + "/unknown")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	if string(body) != "Custom 404 from mounted\n" {
		t.Errorf("got body %q, want %q", string(body), "Custom 404 from mounted\n")
	}
}

func TestStatusRecorderWith200Default(t *testing.T) {
	// test that statusRecorder correctly identifies handlers that return 200 without explicit WriteHeader
	group := routegroup.New(http.NewServeMux())
	
	// set custom NotFound handler
	group.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Custom 404", http.StatusNotFound)
	})
	
	// register a handler that returns 200 without calling WriteHeader explicitly
	// this is common practice - handlers often just call Write() which implicitly sets 200
	group.HandleFunc("GET /implicit200", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("success")) // no WriteHeader call, should be 200
	})
	
	testServer := httptest.NewServer(group)
	defer testServer.Close()
	
	// test that the handler works correctly and isn't mistaken for a 404
	resp, err := http.Get(testServer.URL + "/implicit200")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	// this should return 200 with "success" body
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	
	if string(body) != "success" {
		t.Errorf("expected body 'success', got %q", string(body))
	}
	
	// verify it didn't trigger the custom 404 handler
	if string(body) == "Custom 404\n" {
		t.Error("custom 404 handler was incorrectly triggered for a valid 200 response")
	}
}

func TestCustomNotFoundVsMethodNotAllowed(t *testing.T) {
	// test demonstrates issue #27 - custom NotFound handler should not override 405 Method Not Allowed
	t.Run("without custom NotFound handler", func(t *testing.T) {
		group := routegroup.New(http.NewServeMux())

		// register a route for GET method only
		group.HandleFunc("GET /api/resource", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("GET response"))
		})

		testServer := httptest.NewServer(group)
		defer testServer.Close()

		// test POST to the same path - should return 405
		req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/api/resource", http.NoBody)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("without custom NotFound: status=%d, body=%s", resp.StatusCode, body)

		// this should return 405 Method Not Allowed
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d (Method Not Allowed), got %d", http.StatusMethodNotAllowed, resp.StatusCode)
		}

		// test that Allow header is present
		allowHeader := resp.Header.Get("Allow")
		if allowHeader == "" {
			t.Error("expected Allow header to be present for 405 response")
		} else {
			t.Logf("Allow header: %s", allowHeader)
		}
	})

	t.Run("with custom NotFound handler", func(t *testing.T) {
		group := routegroup.New(http.NewServeMux())

		// set custom NotFound handler
		group.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "Custom 404: Not Found", http.StatusNotFound)
		})

		// register a route for GET method only
		group.HandleFunc("GET /api/resource", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("GET response"))
		})

		testServer := httptest.NewServer(group)
		defer testServer.Close()

		// test POST to the same path - should still return 405, not 404
		req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/api/resource", http.NoBody)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("with custom NotFound: status=%d, body=%s", resp.StatusCode, body)

		// this should return 405 Method Not Allowed, but might return 404 if the issue exists
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d (Method Not Allowed), got %d - custom NotFound handler incorrectly overrides 405",
				http.StatusMethodNotAllowed, resp.StatusCode)
		}

		// test that Allow header is present
		allowHeader := resp.Header.Get("Allow")
		if allowHeader == "" && resp.StatusCode == http.StatusMethodNotAllowed {
			t.Error("expected Allow header to be present for 405 response")
		} else if allowHeader != "" {
			t.Logf("Allow header: %s", allowHeader)
		}

		// verify the body is not the custom 404 message when it should be 405
		if resp.StatusCode == http.StatusNotFound && string(body) == "Custom 404: Not Found\n" {
			t.Error("custom NotFound handler was incorrectly called for a method mismatch (should be 405)")
		}
	})

	// additional test case: verify that actual 404 still uses custom handler
	t.Run("verify actual 404 uses custom handler", func(t *testing.T) {
		group := routegroup.New(http.NewServeMux())

		// set custom NotFound handler
		group.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "Custom 404: Not Found", http.StatusNotFound)
		})

		// register a route for GET method
		group.HandleFunc("GET /api/resource", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("GET response"))
		})

		testServer := httptest.NewServer(group)
		defer testServer.Close()

		// test a completely non-existent path - should use custom 404
		resp, err := http.Get(testServer.URL + "/api/nonexistent")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("actual 404: status=%d, body=%s", resp.StatusCode, body)

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
		}

		// verify custom 404 message is used
		if string(body) != "Custom 404: Not Found\n" {
			t.Errorf("expected custom 404 body, got %q", string(body))
		}
	})
}
