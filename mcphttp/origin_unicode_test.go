package mcphttp

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMCPOriginUnicodeHostsAndIPv6Zones(t *testing.T) {
	for _, test := range []struct {
		name, base, target string
		want               bool
	}{
		{"distinct_sigma_domains", "https://σ.example/rpc", "https://ς.example/message", false},
		{"identical_unicode_bytes", "https://σ.example/rpc", "https://σ.example:443/message", true},
		{"ascii_case", "https://Trusted.Example/rpc", "https://TRUSTED.EXAMPLE:443/message", true},
		{"unicode_host_ascii_case", "https://σ.Example/rpc", "https://σ.EXAMPLE/message", true},
		{"distinct_zone_case", "https://[fe80::1%25eth0]/rpc", "https://[fe80::1%25ETH0]/message", false},
		{"identical_zone", "https://[FE80::1%25eth0]/rpc", "https://[fe80::1%25eth0]:443/message", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			base, err := parseMCPURL(test.base)
			if err != nil {
				t.Fatal(err)
			}
			target, err := parseMCPURL(test.target)
			if err != nil {
				t.Fatal(err)
			}
			if got := sameMCPOrigin(base, target); got != test.want {
				t.Errorf("sameMCPOrigin() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestClientUnicodeLookalikeRedirectCannotReachSink(t *testing.T) {
	// Unicode simple case folding equates σ and ς, but they identify distinct
	// IDNA domains. Exercise the actual http.Client redirect/replay policy.
	for _, phase := range []string{"initialize", "call", "close"} {
		t.Run(phase, func(t *testing.T) {
			attack := phase == "initialize"
			sinkRequests := 0
			installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/sink" {
					sinkRequests++
					if req.Method == http.MethodDelete {
						return securityResponse(req, http.StatusNoContent, ""), nil
					}
					return securityRPCResponse(req, securityRequest(t, req)), nil
				}
				if attack {
					return securityRedirect(req, "https://ς.example/sink"), nil
				}
				if req.Method == http.MethodDelete {
					return securityResponse(req, http.StatusNoContent, ""), nil
				}
				return securityRPCResponse(req, securityRequest(t, req)), nil
			})
			client, err := New(securityContext(t), "test", "https://σ.example/rpc", securityHeaders(), false)
			if phase == "initialize" {
				if err == nil {
					t.Error("initialize accepted a distinct Unicode domain")
				}
			} else {
				if err != nil {
					t.Fatal("normal initialization failed")
				}
				attack = true
				if phase == "call" {
					_, err = client.Call(securityContext(t), "echo", map[string]string{"input": "fake-private-input"})
					if err == nil {
						t.Error("Call accepted a distinct Unicode domain")
					}
				} else {
					_ = client.Close()
				}
			}
			if sinkRequests != 0 {
				t.Errorf("distinct Unicode domain received %d requests", sinkRequests)
			}
		})
	}
}

func TestLegacySSEUnicodeLookalikeCannotReceiveRequests(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		name := "advertised_endpoint"
		if redirect {
			name = "get_redirect"
		}
		t.Run(name, func(t *testing.T) {
			sinkRequests, posts := 0, 0
			installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/sse" {
					sinkRequests++
				}
				if req.Method == http.MethodPost {
					posts++
					return securityResponse(req, http.StatusAccepted, ""), nil
				}
				if redirect && req.URL.Path == "/sse" {
					return securityRedirect(req, "https://ς.example/sink"), nil
				}
				return securityResponse(req, http.StatusOK, "event: endpoint\ndata: https://ς.example/message\n\n"), nil
			})
			client, err := NewSSE(securityContext(t), "test", "https://σ.example/sse", securityHeaders(), false)
			if client != nil {
				_ = client.Close()
			}
			if err == nil {
				t.Error("legacy SSE accepted a distinct Unicode domain")
			}
			// A finite stream can return EOF after an unsafe initialization POST.
			// Check the destination requests as well as the constructor error.
			if sinkRequests != 0 || posts != 0 {
				t.Errorf("distinct Unicode domain received requests=%d posts=%d", sinkRequests, posts)
			}
		})
	}
}

func TestClientIdenticalUnicodeOriginRedirectPreservesRPC(t *testing.T) {
	methods := make(map[string]int)
	redirects := 0
	installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
		checkSecurityHeaders(t, req, methods["initialize"] > 0)
		if req.URL.Path == "/rpc" {
			redirects++
			return securityRedirect(req, "https://σ.example:443/nested/rpc"), nil
		}
		if req.URL.Path != "/nested/rpc" {
			t.Error("Unicode same-origin redirect reached an unexpected path")
		}
		if req.Method == http.MethodDelete {
			methods["DELETE"]++
			return securityResponse(req, http.StatusNoContent, ""), nil
		}
		frame := securityRequest(t, req)
		methods[frame.Method]++
		return securityRPCResponse(req, frame), nil
	})
	client, err := New(securityContext(t), "test", "https://σ.example/rpc", securityHeaders(), false)
	if err != nil {
		t.Fatal("Unicode same-origin initialization failed")
	}
	got, err := client.Call(securityContext(t), "echo", map[string]string{"input": "fake-private-input"})
	if err != nil || got != "ok" {
		t.Fatal("Unicode same-origin Call did not return its result")
	}
	_ = client.Close()
	for _, method := range []string{"initialize", "notifications/initialized", "tools/call", "DELETE"} {
		if methods[method] != 1 {
			t.Errorf("RPC method %s completed %d times, want 1", method, methods[method])
		}
	}
	if redirects != 4 {
		t.Errorf("followed %d Unicode same-origin redirects, want 4", redirects)
	}
}

func TestLegacySSEIdenticalUnicodeOriginPreservesRPC(t *testing.T) {
	endpoint := "https://σ.example:443/message"
	got, err := readMCPEndpoint(bufio.NewReader(strings.NewReader("event: endpoint\ndata: "+endpoint+"\n\n")), "https://σ.example/sse")
	parsed, parseErr := parseMCPURL(got)
	if err != nil || parseErr != nil || parsed.Hostname() != "σ.example" || parsed.Port() != "443" || parsed.Path != "/message" {
		t.Fatal("identical Unicode endpoint was rejected or changed")
	}
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	methods := make(map[string]int)
	installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
		checkSecurityHeaders(t, req, false)
		if req.Method == http.MethodGet {
			go func() { _, _ = io.WriteString(writer, "event: endpoint\ndata: "+endpoint+"\n\n") }()
			response := securityResponse(req, http.StatusOK, "")
			response.Body = reader
			return response, nil
		}
		if req.URL.Hostname() != "σ.example" || req.URL.Path != "/message" {
			t.Error("Unicode SSE request reached an unexpected destination")
		}
		frame := securityRequest(t, req)
		methods[frame.Method]++
		if frame.ID != 0 {
			response := securityRPCResponse(req, frame)
			data, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if err := securityWriteSSE(req.Context(), writer, "event: message\ndata: "+string(data)+"\n\n"); err != nil {
				return nil, err
			}
		}
		return securityResponse(req, http.StatusAccepted, ""), nil
	})
	client, err := NewSSE(securityContext(t), "test", "https://σ.example/sse", securityHeaders(), false)
	if err != nil {
		t.Fatal("Unicode same-origin SSE initialization failed")
	}
	t.Cleanup(func() { _ = client.Close() })
	got, err = client.Call(securityContext(t), "echo", map[string]string{"input": "fake-private-input"})
	if err != nil || got != "ok" {
		t.Fatal("Unicode same-origin SSE Call did not return its result")
	}
	for _, method := range []string{"initialize", "notifications/initialized", "tools/call"} {
		if methods[method] != 1 {
			t.Errorf("legacy RPC method %s completed %d times, want 1", method, methods[method])
		}
	}
}
