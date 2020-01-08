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
	return &fileServer{
		mux:      m,
		root:     root,
		depth:    strings.Count(resolvePattern(pattern), "/"),
		dirIndex: "index.html",
	}
}

func (fs *fileServer) resolveFilePath(v string) string {
	var s, i int
	for ; i < len(v); i++ {
		if v[i] == '/' {
			s++
			if fs.depth == s {
				break
			}
		}
	}

	return filepath.Join(fs.root, v[i:])
}

func (fs *fileServer) contents(w http.ResponseWriter, r *http.Request) {
	file := fs.resolveFilePath(r.URL.Path)
	f, err := os.Open(file)
	if err != nil {
		fs.mux.NotFound(w, r)
		return
	}
	defer f.Close()

	fi, _ := f.Stat()
	if fi.IsDir() {
		file = filepath.Join(file, fs.dirIndex)
		f, err = os.Open(file)
		if err != nil {
			fs.mux.NotFound(w, r)
			return
		}
		defer f.Close()
		if fi, err = f.Stat(); err != nil {
			fs.mux.NotFound(w, r)
			return
		}
	}

	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}
