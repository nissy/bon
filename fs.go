package bon

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type fileServer struct {
	mux      *Mux
	class    int
	root     string
	dirIndex string
}

func (m *Mux) newFileServer(pattern, root string) *fileServer {
	return &fileServer{
		mux:      m,
		root:     root,
		class:    strings.Count(pattern, "/"),
		dirIndex: "index.html",
	}
}

func (fs *fileServer) resolveRequestFile(v string) string {
	var s, i int
	for ; i < len(v); i++ {
		if v[i] == '/' {
			s++
			if fs.class == s {
				break
			}
		}
	}

	return path.Join(fs.root, v[i:])
}

func (fs *fileServer) content(w http.ResponseWriter, r *http.Request) {
	file := fs.resolveRequestFile(r.URL.Path)
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
