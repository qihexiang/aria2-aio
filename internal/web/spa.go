package web

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func SPAHandler(devMode bool, devDir string, embeddedFS fs.FS) http.Handler {
	if devMode {
		return devSPAHandler(devDir)
	}
	subFS, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		panic(err)
	}
	return embeddedSPAHandler(subFS)
}

func embeddedSPAHandler(subFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(subFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if isHashedAsset(path) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		f, err := subFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		r.URL.Path = "/"
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	})
}

func devSPAHandler(devDir string) http.Handler {
	absDir, err := filepath.Abs(devDir)
	if err != nil {
		absDir = devDir
	}

	fileServer := http.FileServer(http.FS(os.DirFS(absDir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		fullPath := filepath.Join(absDir, strings.TrimPrefix(path, "/"))

		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func isHashedAsset(path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 && parts[1] == "assets" {
		name := parts[len(parts)-1]
		if strings.Contains(name, "-") && strings.Contains(name, ".") {
			return true
		}
	}
	return false
}