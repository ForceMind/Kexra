package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTaskAssetProvenanceLocalized(t *testing.T) {
	for _, tc := range []struct{ name, got, want string }{
		{"api", taskAssetSourceAPISummary, "通过资产 API 登记"},
		{"task", taskAssetSourceTaskSummary, "根据任务描述或目标初始化"},
		{"company-conflict", errCompanyNameConflict, "企业名称已存在"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("message=%q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestAssetResponsesLocalized(t *testing.T) {
	s := &Server{m: &Manager{tasks: map[string]*Task{"1": {ID: "1"}}}}
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{"scope-conflict", `{"scope":["example.com"],"asset_ids":[1]}`, "不能同时提交 scope 和 asset_ids", http.StatusBadRequest},
		{"large-body", `{"source_summary":"` + strings.Repeat("a", maxTaskAssetRequestBytes+1) + `"}`, "请求体过大", http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/tasks/1/assets", strings.NewReader(tc.body))
			req.SetPathValue("id", "1")
			rec := httptest.NewRecorder()
			s.attachTaskAssets(rec, req)
			assertTaskLocalizedResponse(t, rec, tc.status, tc.want)
		})
	}

	t.Run("large-company-body", func(t *testing.T) {
		body := `{"name":"` + strings.Repeat("a", maxCompanyMutationBodyBytes+1) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/companies", strings.NewReader(body))
		rec := httptest.NewRecorder()
		var dst map[string]any
		if decodeCompanyMutationRequest(rec, req, &dst) {
			t.Fatal("oversized company accepted")
		}
		assertTaskLocalizedResponse(t, rec, http.StatusRequestEntityTooLarge, "请求体过大")
	})
}
