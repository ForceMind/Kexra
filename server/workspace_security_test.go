package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceSymlinksCannotEscape(t *testing.T) {
	for _, operation := range []string{"list", "read", "download", "write", "upload", "mkdir", "delete", "file-link-read"} {
		t.Run(operation, func(t *testing.T) {
			base := t.TempDir()
			workspace, private := filepath.Join(base, "workspace"), filepath.Join(base, "private")
			for _, dir := range []string{workspace, private} {
				if err := os.Mkdir(dir, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			key := filepath.Join(private, "jwt.key")
			const original = "test-fixture-secret-not-a-real-key"
			if err := os.WriteFile(key, []byte(original), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("../private", filepath.Join(workspace, "escape")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("../private/jwt.key", filepath.Join(workspace, "file-link")); err != nil {
				t.Fatal(err)
			}
			s := &Server{m: &Manager{dir: workspace}}
			var handler http.HandlerFunc
			var req *http.Request
			switch operation {
			case "list":
				handler, req = s.wsList, httptest.NewRequest("GET", "/api/workspace/list?path=escape", nil)
			case "read":
				handler, req = s.wsRead, httptest.NewRequest("GET", "/api/workspace/read?path=escape/jwt.key", nil)
			case "file-link-read":
				handler, req = s.wsRead, httptest.NewRequest("GET", "/api/workspace/read?path=file-link", nil)
			case "download":
				handler, req = s.wsDownload, httptest.NewRequest("GET", "/api/workspace/download?path=escape/jwt.key", nil)
			case "write":
				body := "{\"path\":\"escape/jwt.key\",\"content\":\"overwritten\"}"
				handler, req = s.wsWrite, httptest.NewRequest("POST", "/api/workspace/write", strings.NewReader(body))
			case "upload":
				var body bytes.Buffer
				form := multipart.NewWriter(&body)
				file, err := form.CreateFormFile("file", "jwt.key")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := file.Write([]byte("overwritten")); err != nil {
					t.Fatal(err)
				}
				if err := form.Close(); err != nil {
					t.Fatal(err)
				}
				handler, req = s.wsUpload, httptest.NewRequest("POST", "/api/workspace/upload?path=escape", &body)
				req.Header.Set("Content-Type", form.FormDataContentType())
			case "mkdir":
				handler, req = s.wsMkdir, httptest.NewRequest("POST", "/api/workspace/mkdir", strings.NewReader("{\"path\":\"escape/new-dir\"}"))
			case "delete":
				handler, req = s.wsDelete, httptest.NewRequest("DELETE", "/api/workspace/delete?path=escape/jwt.key", nil)
			}
			rec := httptest.NewRecorder()
			handler(rec, req)
			got, err := os.ReadFile(key)
			if err != nil || string(got) != original {
				t.Errorf("workspace operation changed private fixture: %q, %v", got, err)
			}
			if _, err := os.Stat(filepath.Join(private, "new-dir")); !os.IsNotExist(err) {
				t.Errorf("mkdir escaped workspace: %v", err)
			}
			if rec.Code < 400 || strings.Contains(rec.Body.String(), original) {
				t.Fatalf("escaped workspace operation: status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestWorkspaceRelativeInternalSymlinkStillWorks(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "real"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real", "page.txt"), []byte("科巡证据原文"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(dir, "alias")); err != nil {
		t.Fatal(err)
	}
	s := &Server{m: &Manager{dir: dir}}
	rec := httptest.NewRecorder()
	s.wsRead(rec, httptest.NewRequest("GET", "/api/workspace/read?path=alias/page.txt", nil))
	if rec.Code != 200 {
		t.Fatalf("internal symlink: %d %s", rec.Code, rec.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["content"] != "科巡证据原文" {
		t.Fatalf("content = %q", result["content"])
	}
}

func TestWorkspaceConfinedFileLifecycle(t *testing.T) {
	dir := t.TempDir()
	s := &Server{m: &Manager{dir: dir}}
	request := func(handler http.HandlerFunc, req *http.Request, code int) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != code {
			t.Fatalf("%s: status=%d want=%d body=%s", req.URL.Path, rec.Code, code, rec.Body.String())
		}
		return rec
	}
	request(s.wsMkdir, httptest.NewRequest("POST", "/api/workspace/mkdir", strings.NewReader("{\"path\":\"sub/nested\"}")), 200)
	request(s.wsWrite, httptest.NewRequest("POST", "/api/workspace/write", strings.NewReader("{\"path\":\"sub/nested/evidence.txt\",\"content\":\"unchanged evidence\"}")), 200)
	request(s.wsList, httptest.NewRequest("GET", "/api/workspace/list?path=sub/nested", nil), 200)
	request(s.wsList, httptest.NewRequest("GET", "/api/workspace/list?path=", nil), 200)
	read := request(s.wsRead, httptest.NewRequest("GET", "/api/workspace/read?path=sub/nested/evidence.txt", nil), 200)
	if !strings.Contains(read.Body.String(), "unchanged evidence") {
		t.Fatalf("content changed: %s", read.Body.String())
	}
	rangeReq := httptest.NewRequest("GET", "/api/workspace/download?path=sub/nested/evidence.txt", nil)
	rangeReq.Header.Set("Range", "bytes=0-8")
	download := request(s.wsDownload, rangeReq, 206)
	if download.Body.String() != "unchanged" {
		t.Fatalf("range download = %q", download.Body.String())
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "原始证据.bin")
	if err != nil {
		t.Fatal(err)
	}
	data := []byte{0, 1, 2, 255, 10}
	if _, err := file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	uploadReq := httptest.NewRequest("POST", "/api/workspace/upload?path=sub", &body)
	uploadReq.Header.Set("Content-Type", form.FormDataContentType())
	request(s.wsUpload, uploadReq, 200)
	stored, err := os.ReadFile(filepath.Join(dir, "sub", "原始证据.bin"))
	if err != nil || !bytes.Equal(stored, data) {
		t.Fatalf("binary evidence changed: %v %v", stored, err)
	}
	listing := request(s.wsList, httptest.NewRequest("GET", "/api/workspace/list?path=sub", nil), 200)
	if !strings.Contains(listing.Body.String(), "原始证据.bin") {
		t.Fatalf("missing upload: %s", listing.Body.String())
	}
	request(s.wsDelete, httptest.NewRequest("DELETE", "/api/workspace/delete?path=sub", nil), 200)
	if _, err := os.Stat(filepath.Join(dir, "sub")); !os.IsNotExist(err) {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestChatUploadsCannotEscapeWorkspace(t *testing.T) {
	for _, scope := range []string{"session", "staging"} {
		t.Run(scope, func(t *testing.T) {
			base := t.TempDir()
			workspace, private := filepath.Join(base, "workspace"), filepath.Join(base, "private")
			sub := "sessions"
			if scope == "staging" {
				sub = "drafts"
			}
			if err := os.MkdirAll(filepath.Join(workspace, sub, "fixture"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(private, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(private, filepath.Join(workspace, sub, "fixture", "uploads")); err != nil {
				t.Fatal(err)
			}
			var body bytes.Buffer
			form := multipart.NewWriter(&body)
			file, err := form.CreateFormFile("file", "new-attachment.txt")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write([]byte("fixture upload")); err != nil {
				t.Fatal(err)
			}
			if err := form.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("POST", "/api/chat/upload?scope="+scope+"&id=fixture", &body)
			req.Header.Set("Content-Type", form.FormDataContentType())
			s := &Server{m: &Manager{dir: workspace}}
			rec := httptest.NewRecorder()
			s.chatUpload(rec, req)
			entries, err := os.ReadDir(private)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 || rec.Code < 400 {
				t.Fatalf("chat upload escaped workspace: status=%d outside-files=%d body=%s", rec.Code, len(entries), rec.Body.String())
			}
		})
	}
}
