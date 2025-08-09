package routegroup_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-pkgz/routegroup"
)

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
