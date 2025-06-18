package bon

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type fileServer struct {
	mux      *Mux
	depth    int
	root     string
	absRoot  string // Cache absolute path
	dirIndex string
}

func contentsHandle(r Router, pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	p := resolvePattern(pattern)
	for _, v := range []string{p, p + "*"} {
		r.Handle(http.MethodGet, v, handlerFunc, middlewares...)
		r.Handle(http.MethodHead, v, handlerFunc, middlewares...)
	}
}

func (m *Mux) newFileServer(pattern, root string) *fileServer {
	// Pre-calculate and cache absolute path
	absRoot, err := filepath.Abs(root)
	if err != nil {
		// Use original path on error
		absRoot = root
	}
	
	return &fileServer{
		mux:      m,
		root:     root,
		absRoot:  absRoot,
		depth:    strings.Count(resolvePattern(pattern), "/"),
		dirIndex: "index.html",
	}
}

func (fs *fileServer) resolveFilePath(v string) (string, error) {
	var s, i int
	for ; i < len(v); i++ {
		if v[i] == '/' {
			s++
			if fs.depth == s {
				break
			}
		}
	}

	// Normalize path - remove leading slash
	requestPath := v[i:]
	if requestPath != "" && requestPath[0] == '/' {
		requestPath = requestPath[1:]
	}
	
	// Basic path traversal attack checks:
	// 1. Reject paths containing ".."
	// 2. Reject access to hidden files (starting with ".")
	// 3. Reject paths containing null characters
	if strings.Contains(requestPath, "..") || 
	   strings.Contains(requestPath, "\x00") ||
	   strings.HasPrefix(requestPath, ".") ||
	   strings.Contains(requestPath, "/.") {
		return "", os.ErrPermission
	}
	
	// Clean up the path
	cleanPath := filepath.Clean(requestPath)
	
	// Build absolute path securely using filepath.Join
	fullPath := filepath.Join(fs.root, cleanPath)
	
	// Convert to absolute path
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	
	// Final safety check: ensure path is within root directory
	// Check precisely, not just with HasPrefix
	if !strings.HasPrefix(absPath, fs.absRoot) || 
	   (len(absPath) > len(fs.absRoot) && absPath[len(fs.absRoot)] != filepath.Separator) {
		return "", os.ErrPermission
	}
	
	return absPath, nil
}

func (fs *fileServer) contents(w http.ResponseWriter, r *http.Request) {
	file, err := fs.resolveFilePath(r.URL.Path)
	if err != nil {
		if err == os.ErrPermission {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		fs.mux.NotFound(w, r)
		return
	}
	
	f, err := os.Open(file)
	if err != nil {
		if os.IsPermission(err) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		fs.mux.NotFound(w, r)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		fs.mux.NotFound(w, r)
		return
	}
	
	if fi.IsDir() {
		// For directories, look for index.html
		indexPath := filepath.Join(file, fs.dirIndex)
		
		// Get and validate absolute path of index file
		indexAbsPath, err := filepath.Abs(indexPath)
		if err != nil {
			fs.mux.NotFound(w, r)
			return
		}
		
		// Ensure index file is within root directory
		if !strings.HasPrefix(indexAbsPath, fs.absRoot) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		
		indexFile, err := os.Open(indexPath)
		if err != nil {
			if os.IsPermission(err) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			fs.mux.NotFound(w, r)
			return
		}
		defer indexFile.Close()
		
		indexFi, err := indexFile.Stat()
		if err != nil {
			fs.mux.NotFound(w, r)
			return
		}
		
		http.ServeContent(w, r, indexFi.Name(), indexFi.ModTime(), indexFile)
		return
	}

	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}
