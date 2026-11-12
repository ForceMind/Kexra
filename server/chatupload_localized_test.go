package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatUploadErrorConstantsLocalized(t *testing.T) {
	cases := []struct {
		label, got, want string
	}{
		{"scope", errChatUploadScopeInvalid, "scope 必须为 task、session 或 staging"},
		{"id", errChatUploadBadID, "id 无效"},
		{"task", errChatUploadTaskNotFound, "任务不存在"},
		{"deleting", errChatUploadTaskDeleting, "任务正在删除，无法上传附件"},
		{"mkdir", errChatUploadMkdir, "无法创建目录： "},
		{"parse", errChatUploadParse, "上传处理失败或超过大小限制： "},
		{"file", errChatUploadNoFile, "没有可上传的文件（表单字段 file）"},
		{"save", errChatUploadSaveFailed, "保存失败： "},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			if c.got != c.want {
				t.Fatalf("message=%q, want %q", c.got, c.want)
			}
		})
	}
}

func TestChatUploadHandlerResponsesLocalized(t *testing.T) {
	server := &Server{m: &Manager{dir: t.TempDir(), tasks: map[string]*Task{"t1": {ID: "t1"}}}, engine: &Engine{}}
	server.engine.deleting.Store("t1", true)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("other", "x"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	missingFile := httptest.NewRequest("POST", "/api/chat/upload?scope=session&id=s1", &body)
	missingFile.Header.Set("Content-Type", writer.FormDataContentType())
	cases := []struct {
		name    string
		request *http.Request
		status  int
		want    string
	}{
		{"scope-invalid", httptest.NewRequest("POST", "/api/chat/upload?scope=bogus&id=x", nil), 400, "scope 必须为 task、session 或 staging"},
		{"bad-id", httptest.NewRequest("POST", "/api/chat/upload?scope=session&id=../bad", nil), 400, "id 无效"},
		{"missing-task", httptest.NewRequest("POST", "/api/chat/upload?scope=task&id=missing", nil), 404, "任务不存在"},
		{"task-deleting", httptest.NewRequest("POST", "/api/chat/upload?scope=task&id=t1", nil), 409, "任务正在删除，无法上传附件"},
		{"no-file", missingFile, 400, "没有可上传的文件（表单字段 file）"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.chatUpload(recorder, c.request)
			var response struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != c.status || response.Error != c.want {
				t.Fatalf("status=%d error=%q, want %d %q", recorder.Code, response.Error, c.status, c.want)
			}
		})
	}
	t.Run("parse-error", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		server.chatUpload(recorder, httptest.NewRequest("POST", "/api/chat/upload?scope=session&id=s1", strings.NewReader("invalid")))
		var response struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if recorder.Code != 400 || !strings.HasPrefix(response.Error, "上传处理失败或超过大小限制： ") {
			t.Fatalf("status=%d error=%q", recorder.Code, response.Error)
		}
	})
}
