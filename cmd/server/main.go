package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go-php/server"

	"github.com/google/uuid"
)

// -------------------------------------------------------------------------------
// Static file routing
// -------------------------------------------------------------------------------

type StaticRule struct {
	Prefix string // URL prefix e.g. "/assets/"
	Dir    string // relative to project root, e.g. "public/assets"
}

// tryServeStatic tries to serve from one of the static rules.
// Returns true if it served a file, false if PHP should handle it.
func tryServeStatic(w http.ResponseWriter, r *http.Request, projectRoot string, rules []StaticRule) bool {
	// only serve static for GET/HEAD
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	path := r.URL.Path

	for _, rule := range rules {
		if !strings.HasPrefix(path, rule.Prefix) {
			continue
		}

		// strip prefix from URL path
		relPath := strings.TrimPrefix(path, rule.Prefix)
		relPath = filepath.Clean(relPath)

		// build full filesystem path
		baseDir := filepath.Join(projectRoot, rule.Dir)
		fullPath := filepath.Join(baseDir, relPath)

		// ensure fullPath stays under baseDir (no ../../ escape)
		if !strings.HasPrefix(fullPath, baseDir) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return true
		}

		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			// no file here, let PHP decide or next rule try
			continue
		}

		http.ServeFile(w, r, fullPath)
		return true
	}

	return false
}

// -------------------------------------------------------------------------------
// BuildPayload: Converts HTTP request → bridge format
// -------------------------------------------------------------------------------

func BuildPayload(r *http.Request) *server.RequestPayload {
	headers := map[string]string{}
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	bodyBytes, _ := io.ReadAll(r.Body)

	return &server.RequestPayload{
		ID:      uuid.NewString(),
		Method:  r.Method,
		Path:    r.URL.RequestURI(),
		Headers: headers,
		Body:    string(bodyBytes),
	}
}

// -------------------------------------------------------------------------------
// getProjectRoot: finds directory of go.mod
// -------------------------------------------------------------------------------

func getProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}

	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// hit filesystem root
			return wd
		}

		dir = parent
	}
}

// -------------------------------
// MAIN
// -------------------------------

func main() {
	projectRoot := getProjectRoot()

	// configure static routes (similar to nginx locations)
	staticRules := []StaticRule{
		{Prefix: "/assets/", Dir: "public/assets"},
		{Prefix: "/build/", Dir: "public/build"},
		{Prefix: "/css/", Dir: "public/css"},
		{Prefix: "/js/", Dir: "public/js"},
		{Prefix: "/images/", Dir: "public/images"},
		{Prefix: "/img/", Dir: "public/img"},
	}

	// Create multipools: 4 fast workers, 2 slow workers
	srv, err := server.NewServer(4, 2)
	if err != nil {
		log.Fatal("Failed creating worker pools:", err)
	}

	// optional: enable hot reload if env is set
	devHot := os.Getenv("GO_PHP_HOT_RELOAD") == "1"
	if devHot {
		if err := srv.EnableHotReload(projectRoot); err != nil {
			log.Println("hot reload disabled:", err)
		} else {
			log.Println("hot reload enabled (GO_PHP_HOT_RELOAD=1)")
		}
	}

	log.Println("BareMetalPHP App Server starting on :8080")
	log.Println("Fast workers: 4 | Slow workers: 2")
	log.Println("Static rules:")
	for _, rule := range staticRules {
		log.Printf("  %s -> %s\n", rule.Prefix, filepath.Join(projectRoot, rule.Dir))
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1) Static-first for known asset prefixes
		if tryServeStatic(w, r, projectRoot, staticRules) {
			return
		}

		// 2) Everything else goes to PHP first
		payload := BuildPayload(r)

		resp, err := srv.Dispatch(payload)
		if err != nil {
			log.Println("Worker error:", err)
			http.Error(w, "Worker error: "+err.Error(), 500)
			return
		}

		// 3) Optional: PHP 404 → last-chance static fallback
		if resp.Status == http.StatusNotFound {
			if tryServeStatic(w, r, projectRoot, staticRules) {
				return
			}
		}

		// 4) Write PHP response
		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}

		status := resp.Status
		if status == 0 {
			status = 200
		}
		w.WriteHeader(status)

		_, _ = w.Write([]byte(resp.Body))
	})

	// Start the HTTP server
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("HTTP Server failed:", err)
	}
}
