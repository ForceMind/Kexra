package selfupdate

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRollbackRestoreFailureKeepsCurrentExecutable(t *testing.T) {
	p := testPaths(t)
	if err := os.WriteFile(p.Current, []byte("candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Old, []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := rollbackWithRename(p, func(from, to string) error {
		if from == p.Old {
			return errors.New("simulated restore failure")
		}
		return os.Rename(from, to)
	})
	if err == nil {
		t.Fatal("恢复失败未报告")
	}
	if readAll(t, p.Current) != "candidate" || readAll(t, p.Old) != "previous" {
		t.Fatal("回滚失败后当前程序或备份丢失")
	}
}

func TestRollbackReportsBothRestoreFailures(t *testing.T) {
	p := testPaths(t)
	if err := os.WriteFile(p.Current, []byte("candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Old, []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := rollbackWithRename(p, func(from, to string) error {
		if from == p.Current {
			return os.Rename(from, to)
		}
		return errors.New("simulated rename failure")
	})
	if err == nil || !strings.Contains(err.Error(), "恢复当前程序也失败") {
		t.Fatalf("双重失败必须明确报告：%v", err)
	}
	if readAll(t, p.Current+".failed") != "candidate" || readAll(t, p.Old) != "previous" {
		t.Fatal("双重失败后仍应保留可手动恢复的文件")
	}
}

func TestRollbackRejectsDirectoryBackup(t *testing.T) {
	p := testPaths(t)
	if err := os.WriteFile(p.Current, []byte("candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(p.Old, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := rollback(p); err == nil {
		t.Fatal("目录不能作为可执行备份安装")
	}
	if readAll(t, p.Current) != "candidate" {
		t.Fatal("非法备份导致当前程序被移动")
	}
}
