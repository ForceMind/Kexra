package server

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func chatUploadHeader(t *testing.T, name string, content []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = request.MultipartForm.RemoveAll() })
	return request.MultipartForm.File["file"][0]
}

func TestChatUploadStorageConcurrentNames(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	const count = 24
	headers := make([]*multipart.FileHeader, count)
	for i := range headers {
		headers[i] = chatUploadHeader(t, "证据.bin", []byte(fmt.Sprintf("payload-%d", i)))
	}
	type result struct {
		path string
		err  error
		id   int
	}
	results := make(chan result, count)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i, header := range headers {
		workers.Add(1)
		go func(i int, header *multipart.FileHeader) {
			defer workers.Done()
			<-start
			path, err := saveUniqueRootUpload(root, header, ".", "证据.bin")
			results <- result{path, err, i}
		}(i, header)
	}
	close(start)
	workers.Wait()
	close(results)
	seen := make(map[string]bool)
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if seen[result.path] {
			t.Fatalf("filename reused: %s", result.path)
		}
		seen[result.path] = true
		content, err := os.ReadFile(filepath.Join(dir, result.path))
		if err != nil || string(content) != fmt.Sprintf("payload-%d", result.id) {
			t.Fatalf("upload overwritten: path=%s content=%q error=%v", result.path, content, err)
		}
	}
}

func TestChatUploadStoragePreservesExistingEntries(t *testing.T) {
	for _, kind := range []string{"file", "directory", "dangling-symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			occupied := filepath.Join(dir, "report.txt")
			switch kind {
			case "file":
				if err := os.WriteFile(occupied, []byte("original"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(occupied, 0o700); err != nil {
					t.Fatal(err)
				}
			case "dangling-symlink":
				if err := os.Symlink("target.txt", occupied); err != nil {
					t.Fatal(err)
				}
			}
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			path, err := saveUniqueRootUpload(root, chatUploadHeader(t, "report.txt", []byte("new")), ".", "report.txt")
			if err != nil || path != "report-1.txt" {
				t.Fatalf("path=%q error=%v", path, err)
			}
			if kind == "file" {
				content, err := os.ReadFile(occupied)
				if err != nil || string(content) != "original" {
					t.Fatalf("existing file changed: %q, %v", content, err)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "target.txt")); !os.IsNotExist(err) {
				t.Fatalf("symlink target created: %v", err)
			}
		})
	}
}

func TestChatUploadStorageRemovesIncompleteFile(t *testing.T) {
	header := chatUploadHeader(t, "report.txt", []byte("complete content"))
	source, err := header.Open()
	if err != nil {
		t.Fatal(err)
	}
	diskFile, ok := source.(*os.File)
	if !ok {
		_ = source.Close()
		t.Fatal("expected a spilled multipart file")
	}
	name := diskFile.Name()
	_ = source.Close()
	if err := os.Truncate(name, 3); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	path, err := saveUniqueRootUpload(root, header, ".", "report.txt")
	if path != "" || err != io.ErrUnexpectedEOF {
		t.Fatalf("path=%q error=%v", path, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("incomplete file remains: %v, %v", entries, err)
	}
	if _, err := saveUniqueRootUpload(root, chatUploadHeader(t, "report.txt", []byte("retry")), ".", "report.txt"); err != nil {
		t.Fatal(err)
	}
}

func TestChatUploadStorageEmptyFile(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, want := range []string{"empty", "empty-1"} {
		path, err := saveUniqueRootUpload(root, chatUploadHeader(t, "empty", nil), ".", "empty")
		if err != nil || path != want {
			t.Fatalf("path=%q error=%v, want %q", path, err, want)
		}
		info, err := root.Stat(path)
		if err != nil || info.Size() != 0 {
			t.Fatalf("empty upload changed: info=%v error=%v", info, err)
		}
	}
}
