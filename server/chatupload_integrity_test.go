package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestChatUploadPreservesDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	uploads := filepath.Join(dir, "sessions", "s1", "uploads")
	if err := os.MkdirAll(uploads, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.bin", filepath.Join(uploads, "原文-원본.bin")); err != nil {
		t.Fatal(err)
	}
	content := []byte("원본 증거\x00\xff\x01")
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "原文-원본.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/api/chat/upload?scope=session&id=s1", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	server := &Server{m: &Manager{dir: dir}}
	server.chatUpload(recorder, request)
	var response struct {
		Attachments []chatAttachment `json:"attachments"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != 200 || len(response.Attachments) != 1 {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body)
	}
	attachment := response.Attachments[0]
	if attachment.Name != "原文-원본-1.bin" || attachment.Path != "uploads/原文-원본-1.bin" || attachment.Abs != filepath.Join(uploads, attachment.Name) || attachment.Size != int64(len(content)) {
		t.Fatalf("unexpected manifest: %+v", attachment)
	}
	stored, err := os.ReadFile(attachment.Abs)
	if err != nil || !bytes.Equal(stored, content) {
		t.Fatalf("evidence changed: %q, %v", stored, err)
	}
	if _, err := os.Stat(filepath.Join(uploads, "target.bin")); !os.IsNotExist(err) {
		t.Fatalf("symlink target was written: %v", err)
	}
	if target, err := os.Readlink(filepath.Join(uploads, "原文-원본.bin")); err != nil || target != "target.bin" {
		t.Fatalf("symlink changed: %q, %v", target, err)
	}
}
