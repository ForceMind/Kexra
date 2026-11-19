package db

import "testing"

func TestManualTaskScopeSummaryLocalized(t *testing.T) {
	const want = "用户在测试资产页面手动添加"
	if manualTaskScopeSummary != want {
		t.Fatalf("manual scope summary=%q, want %q", manualTaskScopeSummary, want)
	}
}
