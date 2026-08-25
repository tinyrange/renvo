package main

import (
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	address := flag.String("addr", "127.0.0.1:8000", "listen address")
	directory := flag.String("dir", "sandbox/wasm/browser", "browser asset directory")
	flag.Parse()

	handler, err := newAssetHandler(*directory)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Renvo Web IDE available at http://%s/browser/", *address)
	log.Fatal(http.ListenAndServe(*address, handler))
}

func newAssetHandler(directory string) (http.Handler, error) {
	root, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
		if statErr != nil {
			return nil, statErr
		}
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			response.Header().Set("Allow", "GET, HEAD")
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(filepath.Clean("/"+request.URL.Path), string(filepath.Separator))
		if name == "." || strings.HasSuffix(request.URL.Path, "/") {
			name = filepath.Join(name, "index.html")
		}
		path := filepath.Join(root, name)
		if path != root && !strings.HasPrefix(path, root+string(filepath.Separator)) {
			http.NotFound(response, request)
			return
		}
		info, statErr := os.Stat(path)
		if statErr != nil || info.IsDir() {
			http.NotFound(response, request)
			return
		}

		servedPath := path
		encoding := ""
		if acceptsEncoding(request.Header.Get("Accept-Encoding"), "br") && regularFile(path+".br") {
			servedPath, encoding = path+".br", "br"
		} else if acceptsEncoding(request.Header.Get("Accept-Encoding"), "gzip") && regularFile(path+".gz") {
			servedPath, encoding = path+".gz", "gzip"
		}
		file, openErr := os.Open(servedPath)
		if openErr != nil {
			http.NotFound(response, request)
			return
		}
		defer file.Close()
		servedInfo, statErr := file.Stat()
		if statErr != nil {
			http.Error(response, statErr.Error(), http.StatusInternalServerError)
			return
		}
		contentType := mime.TypeByExtension(filepath.Ext(path))
		if filepath.Ext(path) == ".wasm" {
			contentType = "application/wasm"
		} else if filepath.Ext(path) == ".mjs" {
			contentType = "text/javascript; charset=utf-8"
		}
		if contentType != "" {
			response.Header().Set("Content-Type", contentType)
		}
		if encoding != "" {
			response.Header().Set("Content-Encoding", encoding)
			response.Header().Set("Vary", "Accept-Encoding")
		}
		if request.URL.Query().Has("v") {
			response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			response.Header().Set("Cache-Control", "no-cache")
		}
		http.ServeContent(response, request, filepath.Base(path), servedInfo.ModTime(), file)
	}), nil
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func acceptsEncoding(header, wanted string) bool {
	for _, item := range strings.Split(header, ",") {
		parts := strings.Split(strings.TrimSpace(item), ";")
		if strings.EqualFold(strings.TrimSpace(parts[0]), wanted) {
			for _, parameter := range parts[1:] {
				if strings.TrimSpace(parameter) == "q=0" {
					return false
				}
			}
			return true
		}
	}
	return false
}
