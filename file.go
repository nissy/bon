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
	absRoot  string // 絶対パスをキャッシュ
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
	// 絶対パスを事前に計算してキャッシュ
	absRoot, err := filepath.Abs(root)
	if err != nil {
		// エラーの場合は元のパスを使用
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

	// パスを正規化（先頭のスラッシュを削除）
	requestPath := v[i:]
	if requestPath != "" && requestPath[0] == '/' {
		requestPath = requestPath[1:]
	}
	
	// パストラバーサル攻撃の基本チェック
	// 1. ".." を含むパスを拒否
	// 2. "." で始まるファイル（隠しファイル）へのアクセスを拒否
	// 3. null文字を含むパスを拒否
	if strings.Contains(requestPath, "..") || 
	   strings.Contains(requestPath, "\x00") ||
	   strings.HasPrefix(requestPath, ".") ||
	   strings.Contains(requestPath, "/.") {
		return "", os.ErrPermission
	}
	
	// パスをクリーンアップ
	cleanPath := filepath.Clean(requestPath)
	
	// 絶対パスを構築（filepath.Joinを使用してセキュアに結合）
	fullPath := filepath.Join(fs.root, cleanPath)
	
	// 絶対パスに変換
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	
	// 最終的な安全性チェック：パスがルートディレクトリ内にあることを確認
	// HasPrefixだけでなく、正確に比較
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
		// ディレクトリの場合、index.htmlを探す
		indexPath := filepath.Join(file, fs.dirIndex)
		
		// インデックスファイルの絶対パスを取得して検証
		indexAbsPath, err := filepath.Abs(indexPath)
		if err != nil {
			fs.mux.NotFound(w, r)
			return
		}
		
		// インデックスファイルがルートディレクトリ内にあることを確認
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
