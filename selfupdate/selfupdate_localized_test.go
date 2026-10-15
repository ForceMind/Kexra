package selfupdate

// 升级模块的回归测试：确保页面实际展示的错误、进度和启动通知为简体中文。
// 不修改 API 字段、阶段值、下载来源、校验和或旧版本回滚协议。

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
}

func hasHanzi(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

func assertChineseText(t *testing.T, label, msg string) {
	t.Helper()
	if !hasHanzi(msg) {
		t.Errorf("%s：缺少中文提示：%q", label, msg)
	}
	if hasHangul(msg) {
		t.Errorf("%s：用户可见提示仍包含韩文：%q", label, msg)
	}
}

func assertChineseError(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s：预期出现错误，但实际为 nil", label)
	}
	assertChineseText(t, label, err.Error())
}

// 使用假的 HTTP Transport 验证 GitHub 接口错误，不产生真实网络请求。
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubClient(status int, body string) *http.Client {
	return &http.Client{Transport: rtFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
}

func TestCheckURLErrorsLocalized(t *testing.T) {
	nonHTTPS := checkURL(mustParse(t, "http://api.github.com/x"))
	assertChineseError(t, "非 HTTPS 地址", nonHTTPS)
	if !strings.Contains(nonHTTPS.Error(), "HTTPS") {
		t.Errorf("非 HTTPS 提示必须包含 HTTPS：%q", nonHTTPS.Error())
	}
	nonGitHub := checkURL(mustParse(t, "https://evil.example.com/x"))
	assertChineseError(t, "非 GitHub 域名", nonGitHub)
	if !strings.Contains(nonGitHub.Error(), "GitHub") {
		t.Errorf("非法域名提示必须包含 GitHub：%q", nonGitHub.Error())
	}
}

func TestFetchLatestErrorsLocalized(t *testing.T) {
	ctx := context.Background()
	_, errRate := FetchLatest(ctx, stubClient(http.StatusForbidden, ""))
	assertChineseError(t, "请求限流", errRate)
	if !strings.Contains(errRate.Error(), "60") {
		t.Errorf("限流提示必须保留未认证配额信息：%q", errRate.Error())
	}
	_, errNF := FetchLatest(ctx, stubClient(http.StatusNotFound, ""))
	assertChineseError(t, "尚无正式发布", errNF)
	_, errStatus := FetchLatest(ctx, stubClient(http.StatusInternalServerError, ""))
	assertChineseError(t, "HTTP 状态错误", errStatus)
	if !strings.Contains(errStatus.Error(), "GitHub") {
		t.Errorf("HTTP 错误提示必须包含 GitHub：%q", errStatus.Error())
	}
	_, errJSON := FetchLatest(ctx, stubClient(http.StatusOK, "{not json"))
	assertChineseError(t, "Release JSON 解析失败", errJSON)
	_, errTag := FetchLatest(ctx, stubClient(http.StatusOK, "{}"))
	assertChineseError(t, "缺少 tag 字段", errTag)
	if !strings.Contains(errTag.Error(), "tag") {
		t.Errorf("缺失字段提示必须包含 tag：%q", errTag.Error())
	}
}

func TestStageMissingAssetLocalized(t *testing.T) {
	rel := &Release{TagName: "v9.9.9"}
	err := Stage(context.Background(), &http.Client{}, rel, "0.0.1", nil)
	assertChineseError(t, "当前平台安装包不存在", err)
}

func TestFailedStagedVersionNoticeLocalized(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "previous", 0)
	stage(t, p, "new", 0)
	// 生成校验和后篡改二进制，确保不会安装，并反馈中文失败说明。
	fakeBin(t, p.New, "tampered", 0)
	action, state := applyStaged(p)
	if action != Continue || !state.FailedStage {
		t.Fatalf("校验失败应保留旧版：action=%v state=%+v", action, state)
	}
	assertChineseText(t, "暂存校验失败", state.Detail)
}

func TestRollbackMissingBackupLocalized(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "current", 0)
	err := rollback(p)
	assertChineseError(t, "找不到可回滚的备份", err)
}
