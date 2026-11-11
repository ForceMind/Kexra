package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConversationErrorConstantsLocalized(t *testing.T) {
	for _, tc := range []struct{ name, got, want string }{
		{"request_too_large", convErrRequestTooLarge, "请求体过大"},
		{"agent_key_empty", convErrAgentKeyEmpty, "agent_key 不能为空"},
		{"agent_key_too_long", fmt.Sprintf(convErrAgentKeyTooLong, maxConversationAgentKeyRunes), "agent_key 最多可输入 120 个字符"},
		{"agent_not_found", convErrAgentNotFound, "找不到智能体"},
		{"llm_profile", convErrLLMProfile, "指定的 LLM 配置不存在或尚未设置 API Key"},
		{"title_too_long", fmt.Sprintf(convErrTitleTooLong, maxConversationTitleRunes), "标题最多可输入 200 个字符"},
		{"title_or_pinned", convErrTitleOrPinned, "至少提供 title 或 pinned 中的一项"},
		{"title_empty", convErrTitleEmpty, "标题不能为空"},
		{"bad_conv_id", convErrBadConvID, "会话 ID 无效"},
		{"ids_count", fmt.Sprintf(convErrIDsCount, maxConversationDeleteBatch), "ids 数量必须在 1 到 100 之间"},
		{"message_empty", convErrMessageEmpty, "消息不能为空"},
		{"busy", convErrBusy, "此会话正在处理上一条消息，请稍候"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("error=%q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestDecodeConversationRequestTooLargeLocalized(t *testing.T) {
	body := `{"title":"` + strings.Repeat("a", maxConversationRequestBytes+1024) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(body))
	rec := httptest.NewRecorder()
	var dst struct {
		Title string `json:"title"`
	}
	if decodeConversationRequest(rec, req, &dst) {
		t.Fatal("oversized request accepted")
	}
	const want = "请求体过大"
	if got := decodeErrorField(t, rec.Body.Bytes()); rec.Code != http.StatusRequestEntityTooLarge || got != want {
		t.Fatalf("status=%d error=%q, want 413 %q", rec.Code, got, want)
	}
}

func TestConversationDefaultTitlesLocalized(t *testing.T) {
	if convDefaultTitle != "新对话" || convAttachmentTitle != "附件消息" {
		t.Fatalf("default title=%q attachment title=%q", convDefaultTitle, convAttachmentTitle)
	}
}

func TestIsDefaultConversationTitle(t *testing.T) {
	for _, tc := range []struct {
		name, title string
		want        bool
	}{
		{"empty", "", true},
		{"current", "新对话", true},
		{"legacy", "새 대화", true},
		{"custom", "사용자가 지은 제목", false},
		{"attachments", "附件消息", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDefaultConversationTitle(tc.title); got != tc.want {
				t.Fatalf("isDefaultConversationTitle(%q)=%t, want %t", tc.title, got, tc.want)
			}
		})
	}
}

func TestConversationRetestReasonsLocalized(t *testing.T) {
	for _, tc := range []struct{ name, got, want string }{
		{"failed_to_start", convRetestFailedToStart, "无法启动复测"},
		{"status_read_failed", convRetestStatusReadFailed, "无法读取复测状态，请重新发起复测"},
		{"stopped_or_closed", convRetestStoppedOrClosed, "复测已停止或服务已关闭"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("reason=%q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestTranscriptErrorSummaryLocalized(t *testing.T) {
	for _, tc := range []struct{ name, label, detail, want string }{
		{"chat_turn", "", "connection reset", "(出错，connection reset)"},
		{"main_agent", "主智能体", "원본 오류 / TCP reset", "(主智能体出错，원본 오류 / TCP reset)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := transcriptErrorSummary(tc.label, tc.detail); got != tc.want {
				t.Fatalf("summary=%q, want %q", got, tc.want)
			}
		})
	}
}
