package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 升级提示直接显示在设置页面。断言默认简体中文，不改变 HTTP 状态码和 SSE 结构。
func assertUpdateChinese(t *testing.T, label, message string) {
	t.Helper()
	hasChinese := false
	for _, r := range message {
		if r >= 0x4E00 && r <= 0x9FFF {
			hasChinese = true
		}
		if r >= 0xAC00 && r <= 0xD7A3 {
			t.Errorf("%s：提示包含韩文：%q", label, message)
		}
	}
	if !hasChinese {
		t.Errorf("%s：提示缺少中文：%q", label, message)
	}
}

func TestUpdateMessageConstantsLocalized(t *testing.T) {
	assertUpdateChinese(t, "updateMsgPreparing", updateMsgPreparing)
	assertUpdateChinese(t, "updateMsgFailed", updateMsgFailed)
	assertUpdateChinese(t, "updateMsgStaged", updateMsgStaged)
	assertUpdateChinese(t, "updateErrInProgress", updateErrInProgress)
	assertUpdateChinese(t, "updateErrRollbackInProgress", updateErrRollbackInProgress)

	notRelease := fmt.Sprintf(updateErrNotReleaseFmt, "v0.0.0-dev")
	assertUpdateChinese(t, "updateErrNotReleaseFmt", notRelease)
	if !strings.Contains(notRelease, "v0.0.0-dev") {
		t.Fatalf("updateErrNotReleaseFmt：版本占位符丢失：%q", notRelease)
	}
	latest := fmt.Sprintf(updateErrAlreadyLatestFmt, "v1.2.3")
	assertUpdateChinese(t, "updateErrAlreadyLatestFmt", latest)
	if !strings.Contains(latest, "v1.2.3") {
		t.Fatalf("updateErrAlreadyLatestFmt：版本占位符丢失：%q", latest)
	}
}

// 通过 updateHub 实际路径检查 SSE 进度消息，不需要数据库或外部网络。
func TestUpdateProgressMessagesLocalized(t *testing.T) {
	h := &updateHub{subs: map[chan updateProgress]struct{}{}}
	if !h.begin("v1.2.3") {
		t.Fatal("首次调用 begin 应成功")
	}
	cur, running := h.snapshot()
	if !running || cur.Message != updateMsgPreparing {
		t.Fatalf("准备中消息与状态不符：running=%v message=%q", running, cur.Message)
	}
	assertUpdateChinese(t, "开始更新", cur.Message)
	if h.begin("v1.2.4") {
		t.Fatal("更新进行期间不得再次开始")
	}

	h.finish(errors.New("下载失败"))
	cur, running = h.snapshot()
	if running || cur.Message != updateMsgFailed {
		t.Fatalf("失败消息与状态不符：running=%v message=%q", running, cur.Message)
	}
	assertUpdateChinese(t, "更新失败", cur.Message)
	assertUpdateChinese(t, "错误内容", cur.Error)

	h.finish(nil)
	cur, _ = h.snapshot()
	if cur.Message != updateMsgStaged {
		t.Fatalf("暂存完成消息不符：%q", cur.Message)
	}
	assertUpdateChinese(t, "暂存成功", cur.Message)
}

// 验证更新期间拒绝回滚仍返回 409，且错误正文已经中文化。
func TestUpdateRollbackInProgressLocalized(t *testing.T) {
	updHub.mu.Lock()
	prev := updHub.running
	updHub.running = true
	updHub.mu.Unlock()
	defer func() {
		updHub.mu.Lock()
		updHub.running = prev
		updHub.mu.Unlock()
	}()

	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/update/rollback", nil)
	rec := httptest.NewRecorder()
	s.updateRollback(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("期望 409，实际 %d（正文 %s）", rec.Code, rec.Body.Bytes())
	}
	msg := decodeErrorField(t, rec.Body.Bytes())
	assertUpdateChinese(t, "回滚被拒绝", msg)
	if msg != updateErrRollbackInProgress {
		t.Fatalf("回滚拒绝正文不一致：%q", msg)
	}
}
