package selfupdate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 真实原生子进程调用公开 Bootstrap/Settle，覆盖 Windows .exe 与 Unix。
// fixture 仅导入本地 selfupdate（标准库依赖），不启动数据库、端口或网络。
func TestNativeBootstrapLifecycle(t *testing.T) {
	t.Setenv(smokeEnv, "")
	previous := buildUpdateFixture(t, "fixture-previous")
	candidate := buildUpdateFixture(t, "fixture-candidate")
	for _, outcome := range []string{"settle-and-manual-rollback", "crash-and-auto-rollback"} {
		t.Run(outcome, func(t *testing.T) {
			p := testPaths(t)
			copyFixture(t, previous, p.Current)
			copyFixture(t, candidate, p.New)
			sum, err := fileSHA256(p.New)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p.Sum, []byte(sum), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := saveRollbackMarkerOrDiscard(p, "1.0.0", "v1.1.0"); err != nil {
				t.Fatal(err)
			}
			data := filepath.Join(p.Dir, "synthetic-user-config")
			if err := os.WriteFile(data, []byte("preserve-test-config"), 0o600); err != nil {
				t.Fatal(err)
			}
			// 第一次由旧程序发现暂存件，换装后请求重启。
			runFixture(t, p.Current, "-crash", ExitRestart)
			if outcome == "settle-and-manual-rollback" {
				runFixture(t, p.Current, "-settle", 0)
				if _, err := os.Stat(p.Marker); !os.IsNotExist(err) {
					t.Fatalf("稳定启动未清除标记：%v", err)
				}
				if got := runFixture(t, p.Current, "-version", 0); strings.TrimSpace(got) != "fixture-candidate" {
					t.Fatalf("未启动新版：%s", got)
				}
				if err := manualRollback(p); err != nil {
					t.Fatal(err)
				}
			} else {
				for range maxAttempts {
					runFixture(t, p.Current, "-crash", 1)
				}
				runFixture(t, p.Current, "-crash", ExitRestart)
			}
			if got := runFixture(t, p.Current, "-version", 0); strings.TrimSpace(got) != "fixture-previous" {
				t.Fatalf("没有恢复原生旧程序：%s", got)
			}
			if got := readAll(t, data); got != "preserve-test-config" {
				t.Fatalf("更新回滚影响用户配置：%q", got)
			}
		})
	}
}

func buildUpdateFixture(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	code := fmt.Sprintf(`package main
import (
    "fmt"
    "os"
    "github.com/ForceMind/Kexra/selfupdate"
)
func main() {
    arg := ""
    if len(os.Args) > 1 { arg = os.Args[1] }
    if arg == "-version" { fmt.Println(%q); return }
    action, _ := selfupdate.Bootstrap()
    if action == selfupdate.Restart { os.Exit(selfupdate.ExitRestart) }
    if arg == "-h" { return }
    if arg == "-crash" { os.Exit(1) }
    if arg == "-settle" { selfupdate.Settle() }
}
`, version)
	if err := os.WriteFile(source, []byte(code), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "fixture.exe")
	cmd := exec.Command("go", "build", "-o", bin, source)
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOSUMDB=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("编译本地测试二进制失败：%v\n%s", err, out)
	}
	return bin
}

func copyFixture(t *testing.T, from, to string) {
	t.Helper()
	content, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, content, 0o755); err != nil {
		t.Fatal(err)
	}
}

func runFixture(t *testing.T, bin, arg string, wantExit int) string {
	t.Helper()
	out, err := exec.Command(bin, arg).CombinedOutput()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("运行测试程序失败：%v\n%s", err, out)
		}
		code = exitErr.ExitCode()
	}
	if code != wantExit {
		t.Fatalf("原生程序 %s 的退出码=%d，期望 %d\n%s", arg, code, wantExit, out)
	}
	return string(out)
}
