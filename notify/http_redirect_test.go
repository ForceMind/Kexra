package notify

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type notifyRedirectTransport func(*http.Request) (*http.Response, error)

func (f notifyRedirectTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func withNotifyRedirectTransport(t *testing.T, transport http.RoundTripper) {
	t.Helper()
	previous := httpClient
	client := *previous
	client.Transport = transport
	httpClient = &client
	t.Cleanup(func() { httpClient = previous })
}

func TestNotifyRedirectRejectsHTTPSDowngrade(t *testing.T) {
	const secret = "fake-notify-redirect-secret"
	var sent atomic.Int32
	withNotifyRedirectTransport(t, notifyRedirectTransport(func(r *http.Request) (*http.Response, error) {
		sent.Add(1)
		if r.URL.Scheme == "http" {
			t.Errorf("明文目标收到了凭据头：Authorization=%q X-Api-Key=%q", r.Header.Get("Authorization"), r.Header.Get("X-Api-Key"))
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
		}
		return &http.Response{StatusCode: 307, Header: http.Header{"Location": {"http://notify.example/hook?key=" + secret}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	}))
	_, err := doJSON(context.Background(), http.MethodPost, "https://notify.example/hook?key="+secret,
		map[string]string{"Authorization": "Bearer " + secret, "X-Api-Key": secret}, map[string]string{"detail": secret})
	if err == nil {
		t.Fatal("HTTPS 降级重定向应被拒绝")
	}
	if got := sent.Load(); got != 1 {
		t.Fatalf("降级目标不应收到请求，实际发送了 %d 次", got)
	}
	assertNoSecret(t, err.Error(), secret)
}

// 两个真实本地接收端分别处理 TLS 与明文 HTTP。拨号钩子把同一 URL Host
// 的两种协议路由到这两个 listener，避免依赖机器上已有的双协议代理。
func TestNotifyRedirectRejectsHTTPSDowngradeOnWire(t *testing.T) {
	const secret = "fake-notify-wire-secret"
	var cleartextHits atomic.Int32
	var secureHits atomic.Int32
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleartextHits.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer plain.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secureHits.Add(1)
		w.Header().Set("Location", "http://"+r.Host+"/hook?key="+secret)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer secure.Close()
	tlsConfig := secure.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	dialer := &net.Dialer{Timeout: time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, plain.Listener.Addr().String())
		},
		DialTLSContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, secure.Listener.Addr().String())
			if err != nil {
				return nil, err
			}
			cfg := tlsConfig.Clone()
			cfg.ServerName = "example.com" // httptest 的固定证书覆盖此名称。
			tlsConn := tls.Client(conn, cfg)
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				_ = conn.Close()
				return nil, err
			}
			return tlsConn, nil
		},
	}
	defer transport.CloseIdleConnections()
	withNotifyRedirectTransport(t, transport)
	_, err := doJSON(context.Background(), http.MethodPost, secure.URL+"/hook",
		map[string]string{"Authorization": "Bearer " + secret, "X-Api-Key": secret}, map[string]string{"detail": secret})
	if err == nil || !strings.Contains(err.Error(), "禁止将 HTTPS") {
		t.Fatalf("HTTPS 降级重定向应被策略拒绝，得到 %v", err)
	}
	if got := secureHits.Load(); got != 1 {
		t.Fatalf("TLS 接收端应先收到一次请求，得到 %d", got)
	}
	if got := cleartextHits.Load(); got != 0 {
		t.Fatalf("明文接收端收到了 %d 次请求", got)
	}
	assertNoSecret(t, err.Error(), secret)
}

func TestNotifyRedirectPolicyContracts(t *testing.T) {
	for _, tc := range []struct {
		name, source, target string
		blocked              bool
	}{
		{"HTTPS 同源", "https://notify.example/hook", "https://notify.example/next", false},
		{"HTTP 自建目标", "http://notify.example/hook", "http://notify.example/next", false},
		{"HTTP 升级", "http://notify.example/hook", "https://notify.example/next", false},
		{"HTTPS 降级", "https://notify.example/hook", "http://notify.example/next", true},
		{"跨主机", "https://notify.example/hook", "https://elsewhere.example/next", true},
		{"跨端口", "https://notify.example:443/hook", "https://notify.example:8443/next", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first, _ := http.NewRequest(http.MethodPost, tc.source, nil)
			next, _ := http.NewRequest(http.MethodPost, tc.target, nil)
			err := httpClient.CheckRedirect(next, []*http.Request{first})
			if (err != nil) != tc.blocked {
				t.Fatalf("重定向被拒绝=%v，期望 %v，错误=%v", err != nil, tc.blocked, err)
			}
		})
	}
	first, _ := http.NewRequest(http.MethodPost, "http://notify.example/hook", nil)
	secure, _ := http.NewRequest(http.MethodPost, "https://notify.example/secure", nil)
	cleartext, _ := http.NewRequest(http.MethodPost, "http://notify.example/next", nil)
	if err := httpClient.CheckRedirect(cleartext, []*http.Request{first, secure}); err == nil {
		t.Fatal("从 HTTP 升级后也不得再降级到明文")
	}
}

func TestNotifyRedirectLimitAndErrorRedaction(t *testing.T) {
	const secret = "fake-notify-limit-secret"
	var sent atomic.Int32
	withNotifyRedirectTransport(t, notifyRedirectTransport(func(r *http.Request) (*http.Response, error) {
		sent.Add(1)
		return &http.Response{StatusCode: 307, Header: http.Header{"Location": {"/next?key=" + secret}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	}))
	_, err := doJSON(context.Background(), http.MethodPost, "https://notify.example/hook?key="+secret,
		map[string]string{"Authorization": "Bearer " + secret}, nil)
	if err == nil || !strings.Contains(err.Error(), "重定向次数过多") {
		t.Fatalf("应保留五次请求的跳转限制，得到 %v", err)
	}
	if got := sent.Load(); got != 5 {
		t.Fatalf("应仅发送五次请求，得到 %d", got)
	}
	assertNoSecret(t, err.Error(), secret)
}

func TestNotifyRedirectMalformedLocationDoesNotLeakCredentials(t *testing.T) {
	const secret = "fake-notify-malformed-location-secret"
	for _, location := range []string{
		"https://notify.example:invalid/hook?key=" + secret,
		"HTTPS://notify.example:invalid/hook?key=" + secret,
		"/hook%zz?key=" + secret,
	} {
		withNotifyRedirectTransport(t, notifyRedirectTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 307, Header: http.Header{"Location": {location}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		}))
		_, err := doJSON(context.Background(), http.MethodPost, "https://notify.example/hook", nil, nil)
		if err == nil || !strings.Contains(err.Error(), "Location") {
			t.Fatalf("应保留非法 Location 的诊断信息，得到 %v", err)
		}
		assertNoSecret(t, err.Error(), secret)
	}
}

func TestNotifyRedirectPreservesSameOriginCredentials(t *testing.T) {
	const secret = "fake-notify-same-origin-secret"
	var sent atomic.Int32
	withNotifyRedirectTransport(t, notifyRedirectTransport(func(r *http.Request) (*http.Response, error) {
		sent.Add(1)
		if r.URL.Path == "/hook" {
			return &http.Response{StatusCode: 307, Header: http.Header{"Location": {"/next"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		}
		if r.Header.Get("Authorization") != "Bearer "+secret || r.Header.Get("X-Api-Key") != secret {
			t.Error("同源跳转应保留用户配置的凭据头")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(body), secret) {
			t.Errorf("同源 307 应保留 JSON 请求体：%q，错误=%v", body, err)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Request: r}, nil
	}))
	_, err := doJSON(context.Background(), http.MethodPost, "https://notify.example/hook",
		map[string]string{"Authorization": "Bearer " + secret, "X-Api-Key": secret}, map[string]string{"detail": secret})
	if err != nil || sent.Load() != 2 {
		t.Fatalf("同源跳转应正常投递，发送次数=%d，错误=%v", sent.Load(), err)
	}
}
