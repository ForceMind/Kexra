package selfupdate

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/iotest"
)

func TestDownloadRejectsOverflowAndRemovesPartial(t *testing.T) {
	for _, tc := range []struct {
		name string
		size int64
		body io.Reader
	}{
		{"overflow", 4, strings.NewReader("too much content")},
		{"truncated", 20, strings.NewReader("short")},
		{"interrupted", 20, io.MultiReader(strings.NewReader("start"), iotest.ErrReader(errors.New("interrupted")))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst := filepath.Join(t.TempDir(), "download.part")
			client := &http.Client{Transport: rtFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(tc.body)}, nil
			})}
			_, err := download(context.Background(), client, Asset{Name: "test.zip", URL: "https://github.com/test.zip", Size: tc.size}, dst, func(Phase, int, string) {})
			if err == nil {
				t.Fatal("异常下载被接受")
			}
			if _, err := os.Stat(dst); !os.IsNotExist(err) {
				t.Fatalf("失败下载未清理：%v", err)
			}
		})
	}
}

func TestDownloadRejectsInvalidSizeBeforeRequest(t *testing.T) {
	client := &http.Client{Transport: rtFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("无效大小不得发起请求")
		return nil, nil
	})}
	for _, size := range []int64{-1, maxArchiveSize + 1} {
		_, err := download(context.Background(), client, Asset{Size: size}, filepath.Join(t.TempDir(), "part"), nil)
		assertChineseError(t, "大小无效", err)
	}
}

func TestDownloadAllowsUnknownSizeWithinCap(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "download.part")
	_, err := download(context.Background(), stubClient(200, "payload"), Asset{URL: "https://github.com/test.zip"}, dst, func(Phase, int, string) {})
	if err != nil || readAll(t, dst) != "payload" {
		t.Fatalf("未知大小但有界的正常响应失败：%v", err)
	}
}

func TestBoundedReadersRejectOverflow(t *testing.T) {
	if n, err := copyBounded(io.Discard, strings.NewReader(strings.Repeat("x", 100)), 8); err == nil || n != 9 {
		t.Fatalf("流式读取超过界限：n=%d err=%v", n, err)
	}
	if _, err := readBounded(strings.NewReader("12345"), 4); err == nil {
		t.Fatal("元数据截断不得当成成功")
	}
	if raw, err := readBounded(strings.NewReader("1234"), 4); err != nil || string(raw) != "1234" {
		t.Fatalf("边界值响应应可读取：%q %v", raw, err)
	}
}

func TestFetchLatestRejectsOversizedOrNonReleaseMetadata(t *testing.T) {
	for _, body := range []string{
		strings.Repeat(" ", maxReleaseSize+1),
		`{"tag_name":"v1.1.0","draft":true}`,
		`{"tag_name":"v1.1.0","prerelease":true}`,
		`{"tag_name":"v1.1.0-rc1"}`,
		`{"tag_name":"v1.1.0"} {"tag_name":"v2.0.0"}`,
	} {
		if _, err := FetchLatest(context.Background(), stubClient(200, body)); err == nil {
			t.Fatalf("异常 Release 元数据被接受（长度 %d）", len(body))
		}
	}
}

func TestFetchSumsRejectsOversizedResponse(t *testing.T) {
	rel := &Release{Assets: []Asset{{Name: sumsAsset, URL: "https://github.com/SHA256SUMS"}}}
	// 有效首行后面附加垃圾，旧实现会静默截断并接受该首行。
	body := strings.Repeat("a", 64) + "  test.zip\n" + strings.Repeat(" ", maxSumsSize)
	if _, err := fetchSums(context.Background(), stubClient(200, body), rel); err == nil {
		t.Fatal("超长校验清单不得被静默截断接受")
	}
}

func TestAmbiguousChecksumsAndAssetsAreRejected(t *testing.T) {
	line := strings.Repeat("a", 64) + "  test.zip\n"
	if _, ok := parseSums(line + line + line)["test.zip"]; ok {
		t.Fatal("重复校验条目不得被接受，即使摘要相同")
	}
	rel := &Release{Assets: []Asset{{Name: "test.zip"}, {Name: "TEST.ZIP"}}}
	if _, ok := rel.FindAsset("test.zip"); ok {
		t.Fatal("大小写等价的重复资产不得随机选择一个")
	}
}

func TestUpdateURLRejectsCredentialsAndCustomPorts(t *testing.T) {
	for _, raw := range []string{"https://user:secret@github.com/a", "https://github.com:8443/a"} {
		if err := checkURL(mustParse(t, raw)); err == nil {
			t.Fatalf("异常更新地址被接受：%s", raw)
		}
	}
	if err := checkURL(mustParse(t, "https://github.com:443/a")); err != nil {
		t.Fatal(err)
	}
}

type zipEntry struct {
	name, body string
	mode       os.FileMode
}

func releaseZIP(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, entry := range entries {
		h := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		h.SetMode(entry.mode)
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestExtractBinaryRejectsAmbiguousOrSpecialEntry(t *testing.T) {
	name := "kexra"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for _, entries := range [][]zipEntry{
		{{"first/" + name, "first", 0o755}, {"second/" + name, "second", 0o755}},
		{{"dir/" + name, "target", os.ModeSymlink | 0o777}},
	} {
		dir := t.TempDir()
		archive, dst := filepath.Join(dir, "release.zip"), filepath.Join(dir, "out")
		if err := os.WriteFile(archive, releaseZIP(t, entries...), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := extractBinary(archive, dst); err == nil {
			t.Fatal("歧义/特殊二进制条目被接受")
		}
		if _, err := os.Stat(dst); !os.IsNotExist(err) {
			t.Fatalf("验证失败前已写入输出：%v", err)
		}
	}
}
