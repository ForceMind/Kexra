package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/ForceMind/Kexra/agent"
)

func TestShutdownContextPreservesNamedCause(t *testing.T) {
	signalCtx, signalCancel := context.WithCancel(context.Background())
	ctx, shutdown := shutdownContext(signalCtx)
	defer shutdown(nil)
	signalCancel()
	<-ctx.Done()
	if code, _, _, ok := agent.AbortReason(ctx); !ok || code != "shutdown" {
		t.Fatalf("code=%q ok=%v, want shutdown", code, ok)
	}
}

// TestPrintBannerLocalized ensures the Kexra CLI shows our own product name
// and Simplified Chinese text without requiring Korean.
func TestPrintBannerLocalized(t *testing.T) {
	out := captureStdout(t, func() { printBanner(":8787") })
	for _, want := range []string{"KEXRA", "科巡", "多智能体安全评估", "版本", "监听地址", ":8787"} {
		if !strings.Contains(out, want) {
			t.Errorf("banner missing %q: %q", want, out)
		}
	}
	if strings.ContainsFunc(out, func(r rune) bool { return unicode.Is(unicode.Hangul, r) }) {
		t.Errorf("banner unexpectedly contains Korean: %q", out)
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns whatever
// fn wrote. printBanner uses fmt.Print*, which resolves os.Stdout at call time,
// so swapping it here captures the banner.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}
