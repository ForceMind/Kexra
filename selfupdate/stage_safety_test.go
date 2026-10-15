package selfupdate

import (
	"os"
	"strings"
	"testing"
)

// 模拟可执行文件和升级数据，只操作测试临时目录，不触碰真实安装文件。
func TestSaveRollbackMarkerAllowsVerifiedUpgrade(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "previous", 0)
	stage(t, p, "candidate", 0)
	if err := saveRollbackMarkerOrDiscard(p, "1.0.0", "v1.1.0"); err != nil {
		t.Fatalf("有效升级标记应写入成功：%v", err)
	}
	m, ok := readMarker(p.Marker)
	if !ok || m.From != "1.0.0" || m.To != "1.1.0" || m.StagedAt <= 0 {
		t.Fatalf("升级标记内容不正确：%+v（ok=%v）", m, ok)
	}
	action, state := applyStaged(p)
	if action != Restart || !state.Pending {
		t.Fatalf("合法暂存文件应换装后要求重启：action=%v state=%+v", action, state)
	}
	if !strings.Contains(readAll(t, p.Current), "candidate") {
		t.Fatal("正确版本没有换装")
	}
	if !strings.Contains(readAll(t, p.Old), "previous") {
		t.Fatal("旧版本备份丢失")
	}
}

func TestFailedRollbackMarkerWriteDiscardsStagedVersion(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "previous", 0)
	stage(t, p, "candidate", 0)
	// 使用同名目录稳定模拟 WriteFile 失败（不依赖 root 权限）。
	if err := os.Mkdir(p.Marker, 0o700); err != nil {
		t.Fatal(err)
	}
	err := saveRollbackMarkerOrDiscard(p, "1.0.0", "v1.1.0")
	if err == nil || !strings.Contains(err.Error(), "取消更新") {
		t.Fatalf("写入标记失败必须取消更新并告知用户：%v", err)
	}
	for _, path := range []string{p.New, p.Sum} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("失败后不得遗留可安装暂存件 %s：%v", path, err)
		}
	}
	if !strings.Contains(readAll(t, p.Current), "previous") {
		t.Fatal("升级标记失败时不得修改当前版本")
	}
}

func TestApplyStagedRejectsMissingRollbackMarker(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "previous", 0)
	stage(t, p, "candidate", 0)
	action, state := applyStaged(p)
	if action != Continue || !state.FailedStage {
		t.Fatalf("标记不存在时必须拒绝安装：action=%v state=%+v", action, state)
	}
	if !strings.Contains(state.Detail, "升级标记") {
		t.Fatalf("拒绝安装应包含用户可理解的理由：%q", state.Detail)
	}
	if !strings.Contains(readAll(t, p.Current), "previous") {
		t.Fatal("缺失升级标记时修改了当前版本")
	}
	if _, err := os.Stat(p.New); !os.IsNotExist(err) {
		t.Fatal("拒绝安装后必须清除暂存文件")
	}
}

func TestApplyStagedRejectsCorruptedRollbackMarker(t *testing.T) {
	requireUnix(t)
	p := testPaths(t)
	fakeBin(t, p.Current, "previous", 0)
	stage(t, p, "candidate", 0)
	if err := os.WriteFile(p.Marker, []byte("{corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	action, state := applyStaged(p)
	if action != Continue || !state.FailedStage {
		t.Fatalf("无效标记必须拒绝安装：action=%v state=%+v", action, state)
	}
	if _, err := os.Stat(p.Marker); !os.IsNotExist(err) {
		t.Errorf("已损坏的升级标记应清理：%v", err)
	}
	if !strings.Contains(readAll(t, p.Current), "previous") {
		t.Fatal("无效标记时当前程序遭到修改")
	}
}
