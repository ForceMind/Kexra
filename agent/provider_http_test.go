package agent

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type providerPolicyTransport func(*http.Request) (*http.Response, error)

func (f providerPolicyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func providerPolicyResponse(r *http.Request, status int) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header),
		Body: io.NopCloser(strings.NewReader("{}")), Request: r}
}

func TestProviderRedirectRejectsCredentialReplay(t *testing.T) {
	for _, tc := range []struct{ name, start, target string }{
		{"host", "https://model.example/rpc", "https://sink.example/sink"},
		{"port", "https://model.example/rpc", "https://model.example:444/sink"},
		{"downgrade", "https://model.example/rpc", "http://model.example/sink"},
		{"userinfo", "https://model.example/rpc", "https://other-user@model.example/sink"},
		{"fragment", "https://model.example/rpc", "https://model.example/sink#fragment"},
		{"unicode", "https://σ.example/rpc", "https://ς.example/sink"},
		{"zone", "http://[fe80::1%25eth0]/rpc", "http://[fe80::1%25ETH0]/sink"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, err := providerHTTPClient("")
			if err != nil {
				t.Fatal(err)
			}
			sink := 0
			client.Transport = providerPolicyTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/sink" {
					sink++
					return providerPolicyResponse(r, 200), nil
				}
				response := providerPolicyResponse(r, 307)
				response.Header.Set("Location", tc.target)
				return response, nil
			})
			req, _ := http.NewRequest("POST", tc.start, strings.NewReader("fake-private-prompt"))
			req.Header.Set("Authorization", "Bearer fake-test-token")
			req.Header.Set("X-Api-Key", "fake-test-key")
			req.Header.Set("X-Session-ID", "fake-test-session")
			resp, err := client.Do(req)
			if resp != nil {
				resp.Body.Close()
			}
			if err == nil || sink != 0 {
				t.Fatalf("unsafe redirect: error=%t sink requests=%d", err != nil, sink)
			}
		})
	}
}

func TestProviderSameOriginPreservesHeadersAndBody(t *testing.T) {
	for _, tc := range []struct{ name, start, target string }{
		{"https", "https://model.example/rpc", "https://MODEL.EXAMPLE:443/complete"},
		{"local_http", "http://127.0.0.1:8000/rpc", "/complete"},
		{"private_http", "http://10.0.0.1:8000/rpc", "/complete"},
		{"unicode_exact", "https://σ.example/rpc", "/complete"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, err := providerHTTPClient("")
			if err != nil {
				t.Fatal(err)
			}
			requests := 0
			client.Transport = providerPolicyTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != "fake-prompt" || r.Method != "POST" {
					t.Error("lost request body/method")
				}
				for _, name := range []string{"Authorization", "X-Api-Key", "X-Session-ID"} {
					if r.Header.Get(name) == "" {
						t.Errorf("lost %s", name)
					}
				}
				if r.URL.Path == "/rpc" {
					resp := providerPolicyResponse(r, 307)
					resp.Header.Set("Location", tc.target)
					return resp, nil
				}
				return providerPolicyResponse(r, 200), nil
			})
			req, _ := http.NewRequest("POST", tc.start, strings.NewReader("fake-prompt"))
			for _, name := range []string{"Authorization", "X-Api-Key", "X-Session-ID"} {
				req.Header.Set(name, "fake-header")
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if requests != 2 {
				t.Fatalf("requests=%d, want 2", requests)
			}
		})
	}
}

func TestProviderRedirectLimit(t *testing.T) {
	client, err := providerHTTPClient("")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	client.Transport = providerPolicyTransport(func(r *http.Request) (*http.Response, error) {
		count++
		resp := providerPolicyResponse(r, 307)
		resp.Header.Set("Location", "/loop")
		return resp, nil
	})
	req, _ := http.NewRequest("POST", "https://model.example/rpc", strings.NewReader("{}"))
	resp, err := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil || count > 10 {
		t.Fatalf("unbounded redirects: error=%t requests=%d", err != nil, count)
	}
}

func TestProviderProxyConfigurationAndRedaction(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://unused.example:9999")
	client, err := providerHTTPClient("")
	if err != nil || client.Transport.(*http.Transport).Proxy != nil {
		t.Fatal("empty proxy must remain direct")
	}
	for _, scheme := range []string{"http", "https", "socks5"} {
		t.Run(scheme, func(t *testing.T) {
			c, err := providerHTTPClient(scheme + "://fake-user:fake-password@proxy.example:8080")
			if err != nil {
				t.Fatal("valid proxy rejected")
			}
			transport := c.Transport.(*http.Transport)
			req, _ := http.NewRequest("POST", "https://model.example/rpc", nil)
			proxy, err := transport.Proxy(req)
			if err != nil || proxy.Scheme != scheme || proxy.Host != "proxy.example:8080" || proxy.User == nil {
				t.Fatal("proxy configuration changed")
			}
		})
	}
	for _, raw := range []string{
		"http://fake-user:fixture-secret@proxy.example:bad",
		"ftp://fake-user:fixture-secret@proxy.example:21",
		"fake-user:fixture-secret@proxy.example",
		"http://",
	} {
		t.Run("invalid_"+raw[:4], func(t *testing.T) {
			_, err := providerHTTPClient(raw)
			if err == nil {
				t.Fatal("invalid proxy accepted")
			}
			if strings.Contains(err.Error(), "fixture-secret") || strings.Contains(err.Error(), "fake-user") {
				t.Fatal("proxy validation error disclosed fixture credentials")
			}
		})
	}
}
