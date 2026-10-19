package selfupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadMarkerRejectsInvalidUpgradeState(t *testing.T) {
	for _, raw := range []string{
		`{}`, `null`,
		`{"from":"dev","to":"1.1.0"}`,
		`{"from":"1.0.0","to":"dev"}`,
		`{"from":"1.0.0","to":"1.0.0"}`,
		`{"from":"1.1.0","to":"1.0.0"}`,
		`{"from":"1.0.0","to":"1.1.0","attempts":-1}`,
		`{"from":"1.0.0","to":"1.1.0","attempts":4}`,
		`{"from":"1.0.0","to":"1.1.0","staged_at":-1}`,
	} {
		t.Run(raw, func(t *testing.T) {
			p := testPaths(t)
			if err := os.WriteFile(p.Marker, []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, ok := readMarker(p.Marker); ok {
				t.Fatalf("无效状态被接受：%s", raw)
			}
		})
	}
}

func TestWriteMarkerReplacesCompleteRecord(t *testing.T) {
	p := testPaths(t)
	for attempts := 0; attempts <= maxAttempts; attempts++ {
		m := marker{From: "1.0.0", To: "1.1.0", Attempts: attempts}
		if err := writeMarker(p.Marker, m); err != nil {
			t.Fatal(err)
		}
		if got, ok := readMarker(p.Marker); !ok || got != m {
			t.Fatalf("标记不完整：%+v（ok=%v）", got, ok)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(p.Marker)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("标记权限不正确：%v", info.Mode())
		}
	}
	assertNoMarkerTemps(t, p)
}

func TestWriteMarkerFailurePreservesDestination(t *testing.T) {
	p := testPaths(t)
	// 非空目录稳定模拟最终替换失败，不依赖当前用户是否为 root。
	if err := os.Mkdir(p.Marker, 0o700); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(p.Marker, "keep")
	if err := os.WriteFile(keep, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeMarker(p.Marker, marker{From: "1.0.0", To: "1.1.0"}); err == nil {
		t.Fatal("期望替换失败")
	}
	if got := readAll(t, keep); got != "preserve" {
		t.Fatalf("失败后破坏了已有目标：%q", got)
	}
	assertNoMarkerTemps(t, p)
}

func TestApplyStagedRejectsEmptyJSONMarker(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "previous", 0)
	stage(t, p, "candidate", 0)
	if err := os.WriteFile(p.Marker, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	action, state := applyStaged(p)
	if action != Continue || !state.FailedStage {
		t.Fatalf("空标记不得换装：action=%v state=%+v", action, state)
	}
	if !strings.Contains(readAll(t, p.Current), "previous") {
		t.Fatal("空标记导致当前版本被覆盖")
	}
}

func TestApplyStagedMarkerWriteFailureKeepsCurrent(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "previous", 0)
	// 冒烟子进程在验证标记之后、持久化之前制造目标冲突。
	script := "#!/bin/sh\ncd -- \"$(dirname -- \"$0\")\" || exit 1\nrm kexra.upgrade.json\nmkdir kexra.upgrade.json\n"
	if err := os.WriteFile(p.New, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA256(p.New)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Sum, []byte(sum), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeMarker(p.Marker, marker{From: "1.0.0", To: "1.1.0"}); err != nil {
		t.Fatal(err)
	}
	action, state := applyStaged(p)
	if action != Continue || !state.FailedStage || !strings.Contains(state.Detail, "取消换装") {
		t.Fatalf("标记写入失败仍然换装：action=%v state=%+v", action, state)
	}
	if !strings.Contains(readAll(t, p.Current), "previous") {
		t.Fatal("无法保存标记时修改了当前版本")
	}
	if _, err := os.Stat(p.New); !os.IsNotExist(err) {
		t.Fatalf("未清除失败的暂存件：%v", err)
	}
}

func TestAttemptWriteFailureRollsBackImmediately(t *testing.T) {
	p := testPaths(t)
	if err := os.WriteFile(p.Current, []byte("candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Old, []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(p.Marker, 0o700); err != nil {
		t.Fatal(err)
	}
	action, state := confirmOrRollback(p, marker{From: "1.0.0", To: "1.1.0"})
	if action != Restart || !state.RolledBack || state.Pending {
		t.Fatalf("无法保存尝试次数时没有立即回滚：action=%v state=%+v", action, state)
	}
	if got := readAll(t, p.Current); got != "previous" {
		t.Fatalf("未恢复上一版本：%q", got)
	}
	if got := readAll(t, p.Current+".failed"); got != "candidate" {
		t.Fatalf("未保留失败版本：%q", got)
	}
}

func assertNoMarkerTemps(t *testing.T, p Paths) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(p.Dir, ".kexra-marker-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("遗留标记临时文件：%v（%v）", files, err)
	}
}
