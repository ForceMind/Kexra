package mcphttp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// These tests use the real http.Client redirect and request replay machinery.
// Replacing only its default RoundTripper keeps them independent of sockets,
// DNS, TLS certificates, the MCP SDK, and external services. Do not parallelize
// tests that replace the process-wide http.DefaultTransport.
type securityTransport func(*http.Request) (*http.Response, error)

func (f securityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func installSecurityTransport(t *testing.T, fn securityTransport) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = fn
	t.Cleanup(func() { http.DefaultTransport = previous })
}

func securityResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func securityRedirect(req *http.Request, target string) *http.Response {
	resp := securityResponse(req, http.StatusTemporaryRedirect, "")
	resp.Header.Set("Location", target)
	return resp
}

func securityRequest(t *testing.T, req *http.Request) rpcRequest {
	t.Helper()
	var frame rpcRequest
	if req.Body == nil {
		t.Error("RPC request has no body")
		return frame
	}
	defer req.Body.Close()
	if err := json.NewDecoder(req.Body).Decode(&frame); err != nil {
		t.Error("RPC request body is not valid JSON")
	}
	return frame
}

func securityRPCResponse(req *http.Request, frame rpcRequest) *http.Response {
	if frame.ID == 0 {
		return securityResponse(req, http.StatusAccepted, "")
	}
	result := json.RawMessage(`{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"test","version":"1"}}`)
	if frame.Method == "tools/call" {
		result = json.RawMessage(`{"content":[{"type":"text","text":"ok"}]}`)
	}
	data, _ := json.Marshal(rpcResponse{JSONRPC: "2.0", ID: frame.ID, Result: result})
	resp := securityResponse(req, http.StatusOK, string(data))
	resp.Header.Set("Content-Type", "application/json")
	if frame.Method == "initialize" {
		resp.Header.Set("Mcp-Session-Id", "fake-session-for-tests")
	}
	return resp
}

func securityContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func securityWriteSSE(ctx context.Context, writer *io.PipeWriter, data string) error {
	written := make(chan error, 1)
	go func() {
		_, err := io.WriteString(writer, data)
		written <- err
	}()
	select {
	case err := <-written:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func securityHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer fake-token-for-tests",
		"X-Api-Key":     "fake-api-key-for-tests",
	}
}

func checkSecurityHeaders(t *testing.T, req *http.Request, session bool) {
	t.Helper()
	for name, expected := range securityHeaders() {
		if req.Header.Get(name) != expected {
			t.Errorf("request lost configured %s header", name)
		}
	}
	if session && req.Header.Get("Mcp-Session-Id") != "fake-session-for-tests" {
		t.Error("request lost MCP session header")
	}
}

func TestClientRejectsUnsafeRedirectsBeforeReplay(t *testing.T) {
	targets := []struct{ name, target string }{
		{"other_host", "https://sink.example/sink"},
		{"other_port", "https://trusted.example:444/sink"},
		{"https_downgrade", "http://trusted.example/sink"},
		{"userinfo", "https://test-user@trusted.example/sink"},
		{"fragment", "https://trusted.example/sink#fragment"},
		{"non_http", "ftp://trusted.example/sink"},
	}
	for _, phase := range []string{"initialize", "call", "close"} {
		for _, target := range targets {
			t.Run(phase+"/"+target.name, func(t *testing.T) {
				attack := phase == "initialize"
				sinkRequests, sinkCredentials, sinkSessions, sinkBodies := 0, 0, 0, 0
				installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
					if req.URL.Path == "/sink" {
						sinkRequests++
						if req.Header.Get("X-Api-Key") != "" || req.Header.Get("Authorization") != "" {
							sinkCredentials++
						}
						if req.Header.Get("Mcp-Session-Id") != "" {
							sinkSessions++
						}
						if req.Method == http.MethodDelete {
							return securityResponse(req, http.StatusNoContent, ""), nil
						}
						frame := securityRequest(t, req)
						if frame.Method != "" {
							sinkBodies++
						}
						return securityRPCResponse(req, frame), nil
					}
					if attack {
						return securityRedirect(req, target.target), nil
					}
					if req.Method == http.MethodDelete {
						return securityResponse(req, http.StatusNoContent, ""), nil
					}
					return securityRPCResponse(req, securityRequest(t, req)), nil
				})
				client, err := New(securityContext(t), "test", "https://trusted.example/rpc", securityHeaders(), false)
				if phase == "initialize" {
					if err == nil {
						t.Error("initialize accepted an unsafe redirect")
					}
				} else {
					if err != nil {
						t.Fatal("normal initialization failed")
					}
					attack = true
					if phase == "call" {
						_, err = client.Call(securityContext(t), "echo", map[string]string{"input": "fake-private-input"})
						if err == nil {
							t.Error("Call accepted an unsafe redirect")
						}
					} else {
						_ = client.Close() // Close deliberately reports errors best-effort.
					}
				}
				if sinkRequests != 0 || sinkCredentials != 0 || sinkSessions != 0 || sinkBodies != 0 {
					t.Errorf("unsafe sink received requests=%d credential-bearing=%d session-bearing=%d bodies=%d", sinkRequests, sinkCredentials, sinkSessions, sinkBodies)
				}
			})
		}
	}
}

func TestClientSameOriginRedirectPreservesRPC(t *testing.T) {
	methods := make(map[string]int)
	redirects := 0
	installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
		checkSecurityHeaders(t, req, methods["initialize"] > 0)
		if req.URL.Path == "/rpc" {
			redirects++
			return securityRedirect(req, "https://TRUSTED.EXAMPLE:443/nested/rpc"), nil
		}
		if req.URL.Path != "/nested/rpc" {
			t.Error("same-origin redirect reached an unexpected path")
		}
		if req.Method == http.MethodDelete {
			methods["DELETE"]++
			return securityResponse(req, http.StatusNoContent, ""), nil
		}
		if req.Method != http.MethodPost {
			t.Error("307 redirect changed the RPC HTTP method")
		}
		frame := securityRequest(t, req)
		methods[frame.Method]++
		if frame.Method == "tools/call" {
			params, _ := json.Marshal(frame.Params)
			if !strings.Contains(string(params), "fake-private-input") {
				t.Error("307 redirect lost tool arguments")
			}
		}
		return securityRPCResponse(req, frame), nil
	})
	client, err := New(securityContext(t), "test", "https://trusted.example/rpc", securityHeaders(), false)
	if err != nil {
		t.Fatal("same-origin initialization failed")
	}
	got, err := client.Call(securityContext(t), "echo", map[string]string{"input": "fake-private-input"})
	if err != nil || got != "ok" {
		t.Fatal("same-origin Call did not return its tool result")
	}
	_ = client.Close()
	for _, method := range []string{"initialize", "notifications/initialized", "tools/call", "DELETE"} {
		if methods[method] != 1 {
			t.Errorf("RPC method %s completed %d times, want 1", method, methods[method])
		}
	}
	if redirects != 4 {
		t.Errorf("followed %d same-origin RPC redirects, want 4", redirects)
	}
}

func TestClientRedirectLimit(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy_%t", legacy), func(t *testing.T) {
			requests := 0
			installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
				requests++
				return securityRedirect(req, fmt.Sprintf("/redirect/%d", requests)), nil
			})
			var err error
			if legacy {
				_, err = NewSSE(securityContext(t), "test", "https://trusted.example/sse", securityHeaders(), false)
			} else {
				_, err = New(securityContext(t), "test", "https://trusted.example/rpc", securityHeaders(), false)
			}
			if err == nil {
				t.Error("unbounded redirects were accepted")
			}
			if requests > 11 {
				t.Errorf("issued %d requests, exceeding ten redirects", requests)
			}
		})
	}
}

func TestClientsRejectUnsafeInitialURLBeforeRequest(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		for _, address := range []struct{ name, value string }{
			{"userinfo", "https://test-user@trusted.example/rpc"},
			{"fragment", "https://trusted.example/rpc#fragment"},
			{"non_http", "ftp://trusted.example/rpc"},
		} {
			t.Run(fmt.Sprintf("legacy_%t/%s", legacy, address.name), func(t *testing.T) {
				requests := 0
				installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
					requests++
					if req.Method == http.MethodGet {
						return securityResponse(req, http.StatusOK, ""), nil
					}
					return securityRPCResponse(req, securityRequest(t, req)), nil
				})
				var client *Client
				var err error
				if legacy {
					client, err = NewSSE(securityContext(t), "test", address.value, securityHeaders(), false)
				} else {
					client, err = New(securityContext(t), "test", address.value, securityHeaders(), false)
				}
				if client != nil {
					_ = client.Close()
				}
				if err == nil || requests != 0 {
					t.Errorf("unsafe initial URL issued %d requests or was accepted", requests)
				}
			})
		}
	}
}

func TestReadSSEEndpointRequiresCompleteEndpointEvent(t *testing.T) {
	tests := []struct{ name, stream, want string }{
		{"relative", "event: endpoint\ndata: ../message?session=test\n\n", "https://trusted.example/message?session=test"},
		{"absolute_default_port", "event:endpoint\ndata: https://TRUSTED.EXAMPLE:443/message\n\n", "https://TRUSTED.EXAMPLE:443/message"},
		{"crlf", ": heartbeat\r\nevent: endpoint\r\ndata: /message\r\n\r\n", "https://trusted.example/message"},
		{"field_order", "data: /message\nevent: endpoint\n\n", "https://trusted.example/message"},
		{"ignore_message", "event: message\ndata: https://sink.example/log\n\nevent: endpoint\ndata: /message\n\n", "https://trusted.example/message"},
		{"ignore_unnamed_data", "data: https://sink.example/log\n\nevent: endpoint\ndata: /message\n\n", "https://trusted.example/message"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			trailing := "event: message\ndata: following-rpc-frame\n\n"
			reader := bufio.NewReader(strings.NewReader(test.stream + trailing))
			got, err := readSSEEndpoint(reader, "https://trusted.example/nested/sse")
			if err != nil || got != test.want {
				t.Error("did not resolve the announced endpoint from the complete endpoint event")
				return
			}
			remainder, err := io.ReadAll(reader)
			if err != nil || string(remainder) != trailing {
				t.Error("endpoint parser did not preserve the following RPC frame")
			}
		})
	}
}

func TestReadSSEEndpointRejectsUnsafeOrIncompleteEvent(t *testing.T) {
	tests := []struct{ name, stream string }{
		{"other_host", "event: endpoint\ndata: https://sink.example/message\n\n"},
		{"other_port", "event: endpoint\ndata: https://trusted.example:444/message\n\n"},
		{"network_path", "event: endpoint\ndata: //sink.example/message\n\n"},
		{"https_downgrade", "event: endpoint\ndata: http://trusted.example/message\n\n"},
		{"userinfo", "event: endpoint\ndata: https://test-user@trusted.example/message\n\n"},
		{"fragment", "event: endpoint\ndata: /message#fragment\n\n"},
		{"non_http", "event: endpoint\ndata: ftp://trusted.example/message\n\n"},
		{"opaque_uri", "event: endpoint\ndata: https:message\n\n"},
		{"missing_frame_boundary", "event: endpoint\ndata: /message\n"},
		{"unterminated_line", "event: endpoint\ndata: /message"},
		{"message_only", "event: message\ndata: /message\n\n"},
		{"empty_endpoint", "event: endpoint\n\ndata: /message\n\n"},
		{"multiline_url", "event: endpoint\ndata: /message\ndata: /second\n\n"},
		{"oversized_data", "event: endpoint\ndata: /" + strings.Repeat("x", 64*1024) + "\n\n"},
		{"oversized_frame", "event: endpoint\ndata: /message\n" + strings.Repeat(": padding padding padding\n", 3000) + "\n"},
		{"oversized_crlf_frame", "event: endpoint\r\ndata: /message\r\n" + strings.Repeat(":\r\n", 22000) + "\r\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := readSSEEndpoint(bufio.NewReader(strings.NewReader(test.stream)), "https://trusted.example/sse")
			if err == nil {
				t.Error("accepted an unsafe, incomplete, or oversized endpoint frame")
			}
		})
	}
}

func TestLegacySSERejectsUnsafeEndpointBeforePost(t *testing.T) {
	for _, endpoint := range []struct{ name, value string }{
		{"other_host", "https://sink.example/message"},
		{"other_port", "https://trusted.example:444/message"},
		{"network_path", "//sink.example/message"},
		{"https_downgrade", "http://trusted.example/message"},
		{"userinfo", "https://test-user@trusted.example/message"},
		{"fragment", "/message#fragment"},
		{"non_http", "ftp://trusted.example/message"},
	} {
		t.Run(endpoint.name, func(t *testing.T) {
			posts, credentialPosts, bodyPosts := 0, 0, 0
			installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return securityResponse(req, http.StatusOK, "event: endpoint\ndata: "+endpoint.value+"\n\n"), nil
				}
				posts++
				if req.Header.Get("X-Api-Key") != "" || req.Header.Get("Authorization") != "" {
					credentialPosts++
				}
				if securityRequest(t, req).Method != "" {
					bodyPosts++
				}
				return securityResponse(req, http.StatusAccepted, ""), nil
			})
			client, err := NewSSE(securityContext(t), "test", "https://trusted.example/sse", securityHeaders(), false)
			if client != nil {
				_ = client.Close()
			}
			if err == nil {
				t.Error("legacy SSE accepted an unsafe endpoint")
			}
			// EOF from the finite stream can make old NewSSE return an error even
			// after its initialize POST leaked. Assert the sink, not just err.
			if posts != 0 || credentialPosts != 0 || bodyPosts != 0 {
				t.Errorf("unsafe endpoint received posts=%d credential-bearing=%d bodies=%d", posts, credentialPosts, bodyPosts)
			}
		})
	}
}

func TestLegacySSERejectsUnsafeGETRedirect(t *testing.T) {
	for _, target := range []struct{ name, value string }{
		{"other_host", "https://sink.example/sink"},
		{"other_port", "https://trusted.example:444/sink"},
		{"https_downgrade", "http://trusted.example/sink"},
		{"userinfo", "https://test-user@trusted.example/sink"},
		{"fragment", "https://trusted.example/sink#fragment"},
		{"non_http", "ftp://trusted.example/sink"},
	} {
		t.Run(target.name, func(t *testing.T) {
			sinkRequests := 0
			installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/sse" {
					return securityRedirect(req, target.value), nil
				}
				sinkRequests++
				return securityResponse(req, http.StatusOK, ""), nil
			})
			client, err := NewSSE(securityContext(t), "test", "https://trusted.example/sse", securityHeaders(), false)
			if client != nil {
				_ = client.Close()
			}
			if err == nil || sinkRequests != 0 {
				t.Errorf("unsafe GET redirect reached sink %d times or was accepted", sinkRequests)
			}
		})
	}
}

func TestLegacySSERejectsUnsafePOSTRedirect(t *testing.T) {
	for _, phase := range []string{"initialize", "call"} {
		for _, target := range []struct{ name, value string }{
			{"other_host", "https://sink.example/sink"},
			{"other_port", "https://trusted.example:444/sink"},
			{"https_downgrade", "http://trusted.example/sink"},
			{"userinfo", "https://test-user@trusted.example/sink"},
			{"fragment", "https://trusted.example/sink#fragment"},
			{"non_http", "ftp://trusted.example/sink"},
		} {
			t.Run(phase+"/"+target.name, func(t *testing.T) {
				reader, writer := io.Pipe()
				t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
				attack := phase == "initialize"
				sinkRequests, sinkCredentials, sinkBodies := 0, 0, 0
				installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
					if req.Method == http.MethodGet {
						go func() { _, _ = io.WriteString(writer, "event: endpoint\ndata: /message\n\n") }()
						resp := securityResponse(req, http.StatusOK, "")
						resp.Body = reader
						return resp, nil
					}
					if req.URL.Path == "/message" && attack {
						return securityRedirect(req, target.value), nil
					}
					frame := securityRequest(t, req)
					if req.URL.Path == "/sink" {
						sinkRequests++
						if req.Header.Get("X-Api-Key") != "" || req.Header.Get("Authorization") != "" {
							sinkCredentials++
						}
						if frame.Method != "" {
							sinkBodies++
						}
					}
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
				client, err := NewSSE(securityContext(t), "test", "https://trusted.example/sse", securityHeaders(), false)
				if client != nil {
					t.Cleanup(func() { _ = client.Close() })
				}
				if phase == "initialize" {
					if err == nil {
						t.Error("legacy SSE initialize accepted an unsafe POST redirect")
					}
				} else {
					if err != nil {
						t.Fatal("normal legacy SSE initialization failed")
					}
					attack = true
					_, err = client.Call(securityContext(t), "echo", map[string]string{"input": "fake-private-input"})
					if err == nil {
						t.Error("legacy SSE Call accepted an unsafe POST redirect")
					}
				}
				if sinkRequests != 0 || sinkCredentials != 0 || sinkBodies != 0 {
					t.Errorf("unsafe POST redirect sink received requests=%d credential-bearing=%d bodies=%d", sinkRequests, sinkCredentials, sinkBodies)
				}
			})
		}
	}
}

func TestLegacySSESameOriginRedirectUsesFinalResponseBase(t *testing.T) {
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	methods := make(map[string]int)
	getRedirects := 0
	installSecurityTransport(t, func(req *http.Request) (*http.Response, error) {
		checkSecurityHeaders(t, req, false)
		if req.Method == http.MethodGet {
			if req.URL.Path == "/sse" {
				getRedirects++
				return securityRedirect(req, "https://TRUSTED.EXAMPLE:443/nested/events"), nil
			}
			if req.URL.Path != "/nested/events" {
				t.Error("GET redirect reached an unexpected path")
			}
			go func() { _, _ = io.WriteString(writer, "event: endpoint\ndata: message?session=test\n\n") }()
			resp := securityResponse(req, http.StatusOK, "")
			resp.Body = reader
			return resp, nil
		}
		if req.URL.Path != "/nested/message" || req.URL.RawQuery != "session=test" {
			t.Error("SSE endpoint was not resolved against the final redirected GET URL")
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
	client, err := NewSSE(securityContext(t), "test", "https://trusted.example/sse", securityHeaders(), false)
	if err != nil {
		t.Fatal("same-origin legacy SSE initialization failed")
	}
	t.Cleanup(func() { _ = client.Close() })
	got, err := client.Call(securityContext(t), "echo", map[string]string{"input": "fake-private-input"})
	if err != nil || got != "ok" {
		t.Fatal("same-origin legacy SSE Call did not return its tool result")
	}
	for _, method := range []string{"initialize", "notifications/initialized", "tools/call"} {
		if methods[method] != 1 {
			t.Errorf("legacy RPC method %s completed %d times, want 1", method, methods[method])
		}
	}
	if getRedirects != 1 {
		t.Errorf("GET redirected %d times, want 1", getRedirects)
	}
}
