package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ForceMind/Kexra/db"
)

func decodeErrorField(t *testing.T, body []byte) string {
	t.Helper()
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("invalid error response %q: %v", body, err)
	}
	return resp.Error
}

func assertTaskLocalizedResponse(t *testing.T, rec *httptest.ResponseRecorder, status int, want string) {
	t.Helper()
	if got := decodeErrorField(t, rec.Body.Bytes()); rec.Code != status || got != want {
		t.Fatalf("status=%d error=%q, want %d %q", rec.Code, got, status, want)
	}
}

func TestTaskCategoryResponsesLocalized(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{"empty", `{"name":"   "}`, "分类名称不能为空", http.StatusBadRequest},
		{"long-name", `{"name":"` + strings.Repeat("字", db.MaxTaskCategoryNameRunes+1) + `"}`, "分类名称最多可输入 80 个字符", http.StatusBadRequest},
		{"large-body", `{"name":"` + strings.Repeat("a", maxTaskCategoryRequestBytes+1) + `"}`, "请求体过大", http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/task-categories", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			if _, ok := decodeTaskCategoryRequest(rec, req); ok {
				t.Fatal("invalid category accepted")
			}
			assertTaskLocalizedResponse(t, rec, tc.status, tc.want)
		})
	}

	t.Run("invalid-id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if _, ok := parseCategoryIDField(rec, json.RawMessage("0")); ok {
			t.Fatal("invalid category ID accepted")
		}
		assertTaskLocalizedResponse(t, rec, http.StatusBadRequest, "任务分类 ID 无效")
	})

	t.Run("name-conflict", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeTaskCategoryError(rec, db.ErrTaskCategoryNameConflict)
		assertTaskLocalizedResponse(t, rec, http.StatusConflict, "分类名称已存在")
	})

	t.Run("empty-batch", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/category/batch", strings.NewReader(`{"task_ids":[],"category_id":null}`))
		rec := httptest.NewRecorder()
		(&Server{}).updateTasksCategoryBatch(rec, req)
		assertTaskLocalizedResponse(t, rec, http.StatusBadRequest, "task_ids 数量必须在 1 到 100 之间")
	})
}
