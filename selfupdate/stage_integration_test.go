package selfupdate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
)

// 通过真实 FetchLatest/stageAt/applyStaged/confirmOrRollback 演练更新状态机。
// HTTP Transport 完全在内存中提供模拟 Release；不会访问 GitHub 或修改测试程序。
func TestSimulatedReleaseUpdateLifecycle(t *testing.T) {
	requireUnix(t)
	for _, outcome := range []string{"settle", "crash", "bad-checksum", "bad-binary"} {
		t.Run(outcome, func(t *testing.T) {
			p := testPaths(t)
			fakeBin(t, p.Current, "previous", 0)
			binary := "#!/bin/sh\necho candidate\nexit 0\n"
			if outcome == "bad-binary" {
				binary = "#!/bin/sh\nexit 1\n"
			}
			archive := releaseZIP(t, zipEntry{"package/kexra", binary, 0o755})
			sum := sha256.Sum256(archive)
			digest := hex.EncodeToString(sum[:])
			if outcome == "bad-checksum" {
				digest = strings.Repeat("0", 64)
			}
			name := AssetName("v1.1.0", runtime.GOOS, runtime.GOARCH)
			release := &Release{TagName: "v1.1.0", Assets: []Asset{
				{Name: name, URL: "https://github.com/package.zip", Size: int64(len(archive))},
				{Name: sumsAsset, URL: "https://github.com/SHA256SUMS"},
			}}
			metadata, err := json.Marshal(release)
			if err != nil {
				t.Fatal(err)
			}
			requests := map[string]int{}
			client := &http.Client{Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
				requests[req.URL.String()]++
				var body []byte
				switch req.URL.String() {
				case latestURL:
					body = metadata
				case "https://github.com/package.zip":
					body = archive
				case "https://github.com/SHA256SUMS":
					body = []byte(fmt.Sprintf("%s  %s\n", digest, name))
				default:
					t.Fatalf("演练中出现未预期网络目标：%s", req.URL)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
			})}
			rel, err := FetchLatest(context.Background(), client)
			if err != nil {
				t.Fatal(err)
			}
			var phases []Phase
			err = stageAt(context.Background(), client, rel, "1.0.0", func(phase Phase, _ int, _ string) {
				phases = append(phases, phase)
			}, p)
			if outcome == "bad-checksum" || outcome == "bad-binary" {
				if err == nil {
					t.Fatal("损坏/不可执行的发布包未被拒绝")
				}
				for _, path := range []string{p.New, p.Sum, p.Marker, p.New + ".zip.part", p.New + ".part"} {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Fatalf("失败后遗留暂存文件 %s：%v", path, err)
					}
				}
				if !strings.Contains(readAll(t, p.Current), "previous") {
					t.Fatal("失败更新修改了原程序")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(requests) != 3 || requests[latestURL] != 1 || requests["https://github.com/package.zip"] != 1 || requests["https://github.com/SHA256SUMS"] != 1 {
				t.Fatalf("模拟发布请求不完整或发生重复下载：%v", requests)
			}
			if phases[len(phases)-1] != PhaseStaged {
				t.Fatalf("未报告暂存完成：%v", phases)
			}
			if action, state := applyStaged(p); action != Restart || !state.Pending {
				t.Fatalf("换装未要求重启：%v %+v", action, state)
			}
			for i := 0; i <= maxAttempts; i++ {
				m, ok := readMarker(p.Marker)
				if !ok {
					t.Fatal("升级保护记录丢失")
				}
				action, state := confirmOrRollback(p, m)
				if outcome == "settle" {
					if action != Continue || !state.Pending {
						t.Fatalf("新版本首次启动状态错误：%v %+v", action, state)
					}
					settle(p)
					if _, err := os.Stat(p.Marker); !os.IsNotExist(err) {
						t.Fatalf("稳定后仍然留有标记：%v", err)
					}
					if !strings.Contains(readAll(t, p.Current), "candidate") || !strings.Contains(readAll(t, p.Old), "previous") {
						t.Fatal("成功更新没有保留当前和上一版本")
					}
					break
				}
				if i == maxAttempts {
					if action != Restart || !state.RolledBack || !strings.Contains(readAll(t, p.Current), "previous") {
						t.Fatalf("连续失败未恢复旧版：%v %+v", action, state)
					}
				} else if action != Continue || !state.Pending {
					t.Fatalf("重试计数状态错误：%v %+v", action, state)
				}
			}
		})
	}
}
