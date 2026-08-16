package pages

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// closeNotifyRecorder wraps httptest.ResponseRecorder to implement
// http.CloseNotifier (required by httputil.ReverseProxy; the real gin
// responseWriter supports it, httptest.Recorder does not).
type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
}

func (r *closeNotifyRecorder) CloseNotify() <-chan bool { return make(chan bool) }

// TestWildcardNoPanic verifies that registering the NoRoute wildcard
// alongside specific routes does NOT panic, and routes route correctly.
// HOMEPAGE_URL unset → SPA index fallback path.
func TestWildcardNoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = os.Setenv("HOMEPAGE_URL", "")

	router := gin.New()

	// Simulate other plugins registering specific routes BEFORE pages
	router.GET("/health", func(c *gin.Context) { c.String(200, "ok") })
	router.GET("/api/v1/plans", func(c *gin.Context) { c.String(200, "plans") })
	router.GET("/admin/login", func(c *gin.Context) { c.String(200, "admin") })

	// Now register the pages plugin LAST (as in main.go)
	p := New()
	if err := p.RegisterRoutes(router, nil, nil); err != nil {
		t.Fatalf("RegisterRoutes failed: %v", err)
	}

	assertStatus := func(method, path string, want int) {
		t.Helper()
		w := &closeNotifyRecorder{httptest.NewRecorder()}
		req := httptest.NewRequest(method, path, nil)
		router.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: expected %d, got %d (body=%s)", method, path, want, w.Code, w.Body.String())
		}
	}

	// Specific routes win
	assertStatus("GET", "/health", 200)
	assertStatus("GET", "/api/v1/plans", 200)
	assertStatus("GET", "/admin/login", 200)

	// Unknown API → 404 JSON
	assertStatus("GET", "/api/nothing", 404)

	// Unknown path with HOMEPAGE_URL empty → SPA fallback (index.html missing
	// in test → 404, but no panic and not proxied)
	assertStatus("GET", "/random", 404)

	// /app/dashboard → serve SPA (SPA lives at /app now)
	w := &closeNotifyRecorder{httptest.NewRecorder()}
	req := httptest.NewRequest("GET", "/app/dashboard", nil)
	router.ServeHTTP(w, req)
	if w.Code != 404 {
		// SPA dist not present in test env → index.html 404, but NOT a redirect
		// (confirm it's NOT proxied to homepage — proxy only when path != /app)
		t.Fatalf("expected /app/dashboard 404 (no dist), got %d", w.Code)
	}

	// /spa/dashboard → 301 redirect ke /app
	w = &closeNotifyRecorder{httptest.NewRecorder()}
	req = httptest.NewRequest("GET", "/spa/dashboard", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("expected /spa/dashboard 301, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/app/dashboard" {
		t.Fatalf("expected redirect Location /app/dashboard, got %q", loc)
	}

	t.Log("wildcard NoRoute routing OK — no panic")
}

// TestProxyPath verifies that with HOMEPAGE_URL set, unknown non-Go paths are
// proxied to the homepage target (uses a stub homepage server).
func TestProxyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Stub homepage server
	home := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<h1>landing</h1> path="+r.URL.Path)
	}))
	defer home.Close()

	_ = os.Setenv("HOMEPAGE_URL", home.URL)
	t.Cleanup(func() { _ = os.Setenv("HOMEPAGE_URL", "") })

	router := gin.New()
	router.GET("/health", func(c *gin.Context) { c.String(200, "ok") })

	p := New()
	if err := p.RegisterRoutes(router, nil, nil); err != nil {
		t.Fatalf("RegisterRoutes failed: %v", err)
	}

	// "/" → proxied to homepage
	w := &closeNotifyRecorder{httptest.NewRecorder()}
	req := httptest.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)
	if w.Code != 200 || w.Body.String() != "<h1>landing</h1> path=/" {
		t.Fatalf("expected proxied landing at /, got %d body=%q", w.Code, w.Body.String())
	}

	// "/assets/app.css" → proxied to homepage (homepage assets)
	w = &closeNotifyRecorder{httptest.NewRecorder()}
	req = httptest.NewRequest("GET", "/assets/app.css", nil)
	router.ServeHTTP(w, req)
	if w.Code != 200 || w.Body.String() != "<h1>landing</h1> path=/assets/app.css" {
		t.Fatalf("expected proxied asset, got %d body=%q", w.Code, w.Body.String())
	}

	// "/health" still served by Go
	w = &closeNotifyRecorder{httptest.NewRecorder()}
	req = httptest.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)
	if w.Code != 200 || w.Body.String() != "ok" {
		t.Fatalf("expected /health 200 from Go, got %d body=%q", w.Code, w.Body.String())
	}
}

// TestServeSPA verifies /app static file + index.html fallback serving.
func TestServeSPA(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a fake SPA dist
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, "assets"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, "index.html"), []byte("<html>spa-index</html>"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "assets", "app.js"), []byte("var x=1;"), 0o644)

	// Point spaDir to tmp (spaDir is a field defaulting to the dist const;
	// override in test to a temp dir).
	p := &Plugin{spaDir: tmp, spaIndex: filepath.Join(tmp, "index.html")}

	router := gin.New()
	router.NoRoute(p.wildcardHandler)

	// /app/assets/app.js → static file
	w := &closeNotifyRecorder{httptest.NewRecorder()}
	req := httptest.NewRequest("GET", "/app/assets/app.js", nil)
	router.ServeHTTP(w, req)
	if w.Code != 200 || w.Body.String() != "var x=1;" {
		t.Fatalf("expected static file, got %d body=%q", w.Code, w.Body.String())
	}

	// /app/login (client route) → index.html
	w = &closeNotifyRecorder{httptest.NewRecorder()}
	req = httptest.NewRequest("GET", "/app/login", nil)
	router.ServeHTTP(w, req)
	if w.Code != 200 || w.Body.String() != "<html>spa-index</html>" {
		t.Fatalf("expected index fallback, got %d body=%q", w.Code, w.Body.String())
	}

	// Path traversal guard → 404. Dipanggil langsung via handler (bukan
	// router.ServeHTTP) karena router gin menolak path berisi ".." dengan 400
	// sebelum handler dijalankan — pengujian ini menargetkan guard di dalam
	// serveSPA itu sendiri.
	w = &closeNotifyRecorder{httptest.NewRecorder()}
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/app/../../etc/passwd", nil)
	p.wildcardHandler(c)
	if w.Code != 404 {
		t.Fatalf("expected path traversal 404, got %d", w.Code)
	}

	// "/app/%2e%2e/%2e%2e/etc/passwd" (encoded) juga harus 404.
	w = &closeNotifyRecorder{httptest.NewRecorder()}
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/app/%2e%2e/%2e%2e/etc/passwd", nil)
	p.wildcardHandler(c)
	if w.Code != 404 {
		t.Fatalf("expected encoded path traversal 404, got %d", w.Code)
	}
}
