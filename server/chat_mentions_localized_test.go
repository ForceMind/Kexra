package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestChatMentionErrorsLocalized(t *testing.T) {
	var tooMany strings.Builder
	for i := 1; i <= 11; i++ {
		fmt.Fprintf(&tooMany, "@[漏洞#%d] ", i)
	}

	server := &Server{m: &Manager{}}
	for _, tc := range []struct {
		name, message, want string
		status              int
	}{
		{"invalid-id", "@[漏洞#0]", "引用 ID 无效，请重新选择", http.StatusBadRequest},
		{"too-many", tooMany.String(), "每条消息最多引用 10 条记录", http.StatusBadRequest},
		{"data-unavailable", "@[漏洞#1]", "暂时无法读取引用记录", http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			if _, ok := server.prepareChatMentionMessage(rec, tc.message); ok {
				t.Fatal("invalid reference accepted")
			}
			if got := decodeErrorField(t, rec.Body.Bytes()); rec.Code != tc.status || got != tc.want {
				t.Fatalf("status=%d error=%q, want %d %q", rec.Code, got, tc.status, tc.want)
			}
		})
	}

	for _, tc := range []struct{ name, query string }{
		{"invalid-kind", "kind=unsupported"},
		{"query-too-long", "q=" + url.QueryEscape(strings.Repeat("字", 201))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			server.searchChatMentions(rec, httptest.NewRequest(http.MethodGet, "/api/chat/mentions?"+tc.query, nil))
			const want = "引用类型无效或搜索词超过 200 个字符"
			if got := decodeErrorField(t, rec.Body.Bytes()); rec.Code != http.StatusBadRequest || got != want {
				t.Fatalf("status=%d error=%q, want 400 %q", rec.Code, got, want)
			}
		})
	}

	const wantMissing = "引用的漏洞 #7 不存在或类型不匹配，请移除后重新选择"
	if got := fmt.Sprintf(errChatMentionNotFoundFmt, chatMentionKindLabel["finding"], 7); got != wantMissing {
		t.Fatalf("missing record error=%q, want %q", got, wantMissing)
	}
	if errChatMentionTooLarge != "引用内容过大，请减少引用记录后重试" {
		t.Fatalf("oversized reference error=%q", errChatMentionTooLarge)
	}
	for label, kind := range chatMentionKinds {
		if got := chatMentionKindLabel[kind]; got != label {
			t.Errorf("%s label=%q, want %q", kind, got, label)
		}
	}
}
