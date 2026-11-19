package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ForceMind/Kexra/db"
)

func TestTaskTemplateErrorConstantsLocalized(t *testing.T) {
	for _, tc := range []struct{ name, got, want string }{
		{"invalid-rule", errTaskTemplateRuleInvalid, "拦截或放行规则无效，"},
		{"empty-patch", errTaskTemplateNoFields, "至少提供 name、description、goal、category_id 或 intercept_rules 中的一项"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("error=%q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestTaskTemplateResponsesLocalized(t *testing.T) {
	t.Run("large-body", func(t *testing.T) {
		body := `{"name":"` + strings.Repeat("a", maxTaskTemplateRequestBytes+1) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/task-templates", strings.NewReader(body))
		rec := httptest.NewRecorder()
		var decoded taskTemplateRequest
		if _, ok := decodeTaskTemplateRequest(rec, req, &decoded); ok {
			t.Fatal("oversized template accepted")
		}
		assertTaskLocalizedResponse(t, rec, http.StatusRequestEntityTooLarge, "请求体过大")
	})

	t.Run("long-name", func(t *testing.T) {
		name := strings.Repeat("字", db.MaxTaskTemplateNameRunes+1)
		err := validateTaskTemplateRequest(taskTemplateRequest{Name: &name})
		if err == nil || err.Error() != "name 最多可输入 120 个字符" {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("name-conflict", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeTaskTemplateErr(rec, db.ErrTaskTemplateNameConflict)
		assertTaskLocalizedResponse(t, rec, http.StatusConflict, "模板名称已存在")
	})
}
