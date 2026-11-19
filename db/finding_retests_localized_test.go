package db

import "testing"

func TestFindingRetestReasonsLocalized(t *testing.T) {
	for _, tc := range []struct{ name, got, want string }{
		{"no_conclusion", retestNoConclusionReason, "智能体未保存复测结论，请查看会话后重新发起复测"},
		{"service_restart", retestServiceRestartReason, "服务重启导致复测中断，请重新发起复测"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("reason=%q, want %q", tc.got, tc.want)
			}
		})
	}
}
