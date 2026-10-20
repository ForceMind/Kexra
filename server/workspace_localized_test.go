package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"unicode"
)

// TestWorkspaceErrorConstantsLocalized checks default messages only, not raw evidence or errors.
func TestWorkspaceErrorConstantsLocalized(t *testing.T) {
	for label, msg := range map[string]string{
		"illegalPath":      errWsIllegalPath,
		"pathNotFound":     errWsPathNotFound,
		"notDir":           errWsNotDir,
		"fileNotFound":     errWsFileNotFound,
		"isDir":            errWsIsDir,
		"targetIsDir":      errWsTargetIsDir,
		"cannotDeleteRoot": errWsCannotDeleteRoot,
		"uploadDirMissing": errWsUploadDirMissing,
		"uploadParse":      errWsUploadParse,
		"noUploadFile":     errWsNoUploadFile,
	} {
		assertWorkspaceChineseError(t, label, msg)
	}
}

// wsErrBody pulls the {"error": "..."} string writeErr produces.
func wsErrBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("응답 JSON 파싱 실패: %v (본문 %q)", err, rec.Body.String())
	}
	return out.Error
}

// TestWorkspaceHandlerResponsesLocalized verifies actual Chinese HTTP error bodies without a DB.
func TestWorkspaceHandlerResponsesLocalized(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := &Server{m: &Manager{dir: dir}}

	cases := []struct {
		name    string
		handler http.HandlerFunc
		req     *http.Request
		code    int
		want    string
	}{
		{"read-missing", s.wsRead, httptest.NewRequest(http.MethodGet, "/api/workspace/read?path=nope.txt", nil), 404, "文件不存在"},
		{"read-dir", s.wsRead, httptest.NewRequest(http.MethodGet, "/api/workspace/read?path=sub", nil), 400, "该路径是目录，无法作为文件读取"},
		{"list-missing", s.wsList, httptest.NewRequest(http.MethodGet, "/api/workspace/list?path=nope", nil), 404, "路径不存在"},
		{"list-not-dir", s.wsList, nil, 400, "不是目录"}, // req built below (needs a real file)
		{"delete-root", s.wsDelete, httptest.NewRequest(http.MethodDelete, "/api/workspace/delete?path=", nil), 400, "不能删除工作空间根目录"},
	}

	// list-not-dir needs an actual file to point at.
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases[3].req = httptest.NewRequest(http.MethodGet, "/api/workspace/list?path=a.txt", nil)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c.handler(rec, c.req)
			if rec.Code != c.code {
				t.Fatalf("status = %d, want %d (본문 %q)", rec.Code, c.code, rec.Body.String())
			}
			got := wsErrBody(t, rec)
			if got != c.want {
				t.Fatalf("error = %q, want %q", got, c.want)
			}
			assertWorkspaceChineseError(t, c.name, got)
		})
	}
}

func assertWorkspaceChineseError(t *testing.T, label, message string) {
	t.Helper()
	hasHan := false
	for _, r := range message {
		if unicode.Is(unicode.Hangul, r) {
			t.Fatalf("%s still contains a legacy default Korean message: %q", label, message)
		}
		hasHan = hasHan || unicode.Is(unicode.Han, r)
	}
	if !hasHan {
		t.Fatalf("%s has no Chinese display text: %q", label, message)
	}
}

func TestWorkspaceEvidenceTextIsNotTranslated(t *testing.T) {
	dir := t.TempDir()
	name, content := "원본-证据.txt", "원본 증거 / raw payload: 사용자 원문"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Server{m: &Manager{dir: dir}}
	rec := httptest.NewRecorder()
	s.wsRead(rec, httptest.NewRequest(http.MethodGet, "/api/workspace/read?path="+name, nil))
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["content"] != content || body["path"] != name {
		t.Fatalf("evidence or filename changed: %v", body)
	}
}
