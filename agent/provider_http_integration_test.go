package agent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Autumn-27/norma/transcript"
)

// Full-module regression: keeps the real quota/capture/session wrapper in the
// path. Unlike the pure-stdlib factory tests, this requires the pinned SDK.
func TestQuotaClientKeepsRedirectPolicyWithSessionWrapper(t *testing.T) {
	client, err := quotaAwareHTTPClient("", "X-Session-ID")
	if err != nil {
		t.Fatal(err)
	}
	wrapper, ok := client.Transport.(quotaAwareTransport)
	if !ok {
		t.Fatal("quota instrumentation lost")
	}
	sink := 0
	wrapper.base = providerPolicyTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/sink" {
			sink++
			return providerPolicyResponse(r, 200), nil
		}
		if r.Header.Get("X-Session-ID") != "fixture-session" {
			t.Error("session header lost")
		}
		response := providerPolicyResponse(r, 307)
		response.Header.Set("Location", "https://sink.example/sink")
		return response, nil
	})
	client.Transport = wrapper
	req, _ := http.NewRequestWithContext(transcript.WithSessionID(context.Background(), "fixture-session"),
		"POST", "https://model.example/rpc", strings.NewReader("fixture-prompt"))
	req.Header.Set("X-Api-Key", "fixture-key")
	response, err := client.Do(req)
	if response != nil {
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
	}
	if err == nil || sink != 0 {
		t.Fatalf("wrapped client forwarded credentials: error=%t sink=%d", err != nil, sink)
	}
}
