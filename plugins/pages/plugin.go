package pages

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	"github.com/rolldone/donisgo/internal/plugins"
)

const (
	// spaDir adalah direktori hasil build frontend (Vite production build).
	spaDir = "./sub_app/webapp/dist"
)

// Plugin adalah plugin "bawaan" framework yang mengelola seluruh tampilan
// (pages): ia memegang handler catch-all / wildcard untuk SEMUA path yang
// tidak dikenal.
//
// Keputusan routing (per path):
//   - "/api/*" tak dikenal → 404 JSON (route API asli dari plugin lain menang duluan)
//   - "/app*"            → serve SPA (file statis / index.html untuk client-side routes)
//   - "/spa*"            → redirect ke "/app" (backward compat)
//   - path lain          → proxy homepage (jika HOMEPAGE_URL diset) / index.html SPA (fallback)
//
// Plugin ini HARUS didaftarkan PALING AKHIR di cmd/server/main.go karena ia
// "memiliki" wildcard; semua route spesifik dari plugin lain (api/admin/etc.)
// diregistrasi duluan dan menang. Gin v1.9 tidak mendukung root catch-all
// "/*any" yang berdampingan dengan static routes (panic), jadi dipakai
// router.NoRoute.
type Plugin struct {
	proxy    *httputil.ReverseProxy
	hasProxy bool
	spaDir   string
	spaIndex string
}

// New returns a new plugin instance.
func New() plugins.Plugin { return &Plugin{} }

// ID returns the plugin id.
func (p *Plugin) ID() string { return "pages" }

// RegisterServices is a no-op for this plugin.
func (p *Plugin) RegisterServices(_ plugins.ServiceDeps) error { return nil }

// RegisterMiddleware returns no middleware.
func (p *Plugin) RegisterMiddleware() []plugins.MiddlewareDescriptor { return nil }

// RegisterRoutes registers the wildcard/catch-all handler for all unknown paths.
func (p *Plugin) RegisterRoutes(router *gin.Engine, _, _ *gin.RouterGroup) error {
	p.spaDir = spaDir
	p.spaIndex = filepath.Join(p.spaDir, "index.html")

	// Konfigurasi proxy homepage hanya jika HOMEPAGE_URL diset eksplisit.
	// Jika kosong → tanpa proxy, fallback ke index.html SPA (perilaku lama).
	homepageURL := os.Getenv("HOMEPAGE_URL")
	if homepageURL != "" {
		target, err := url.Parse(homepageURL)
		if err != nil {
			log.Printf("[pages] invalid HOMEPAGE_URL %q: %v — proxy disabled, SPA fallback", homepageURL, err)
		} else {
			proxy := httputil.NewSingleHostReverseProxy(target)
			// Graceful fallback: jika homepage unreachable, serve SPA alih-alih 502.
			proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
				log.Printf("[pages] homepage proxy error — serving SPA fallback")
				http.ServeFile(w, (&http.Request{URL: &url.URL{}}), p.spaIndex)
			}
			p.proxy = proxy
			p.hasProxy = true
			log.Printf("[pages] homepage proxy → %s", target.String())
		}
	}

	// Wildcard / catch-all handler — menangkap SEMUA path tak dikenal.
	router.NoRoute(p.wildcardHandler)

	return nil
}

// wildcardHandler routes unknown paths.
func (p *Plugin) wildcardHandler(c *gin.Context) {
	path := c.Request.URL.Path

	switch {
	case strings.HasPrefix(path, "/app"):
		// Serve SPA: file statis jika ada, selain itu index.html (client-side route).
		p.serveSPA(c, path)

	case strings.HasPrefix(path, "/api"):
		// Route API tak dikenal → 404 JSON (bukan homepage/SPA).
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})

	case strings.HasPrefix(path, "/spa"):
		// Backward-compat redirect: /spa/* → /app/* (SPA pindah ke /app).
		newPath := "/app" + strings.TrimPrefix(path, "/spa")
		if newPath == "/app" {
			newPath = "/app/"
		}
		c.Redirect(http.StatusMovedPermanently, newPath)

	case p.hasProxy:
		// Path lain → homepage (landing page proxy).
		p.proxy.ServeHTTP(c.Writer, c.Request)

	default:
		// Tanpa homepage → fallback ke index.html SPA.
		c.File(p.spaIndex)
	}
}

// serveSPA serves a static file from the SPA dist, or index.html for
// client-side routes. Uses a raw-segment check + filepath.Clean + prefix
// check to prevent path traversal.
func (p *Plugin) serveSPA(c *gin.Context, path string) {
	rel := strings.TrimPrefix(path, "/app")
	if rel == "" {
		rel = "/"
	}

	// Path traversal guard — cek SEGMEN ".." pada path mentah SEBELUM
	// filepath.Clean (karena Clean me-resolve ".." ke root dan bisa
	// menghilangkan jejaknya, mis. "/../../etc/passwd" → "/etc/passwd").
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".." {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
	}

	clean := filepath.Clean(rel)
	// Guard tambahan: hasil Clean tidak boleh keluar dari spaDir.
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/..") {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	full := filepath.Join(p.spaDir, clean)
	if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
		c.File(full)
		return
	}

	// Tidak ada file yang cocok → client-side route → index.html
	c.File(p.spaIndex)
}

// Seed is a no-op for this plugin.
func (p *Plugin) Seed() error { return nil }

// ConsoleCommands returns no console commands.
func (p *Plugin) ConsoleCommands() []*cobra.Command { return nil }
