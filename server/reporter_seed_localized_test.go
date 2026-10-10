package server

import (
    "testing"
    "unicode"
)

// Reporter 仅涉及展示名称与描述，不改变 reporter 内部任务逻辑。
func TestReporterSeedLabelsLocalizedToChinese(t *testing.T) {
    for label, value := range map[string]string{
        "reporter_name": reporterAgentName,
        "reporter_description": reporterAgentDescription,
    } {
        var han bool
        for _, r := range value {
            if unicode.Is(unicode.Han, r) { han = true }
            if unicode.Is(unicode.Hangul, r) { t.Errorf("%s: 韩语残留：%q", label, value); break }
        }
        if !han { t.Errorf("%s: 应使用简体中文，实际为 %q", label, value) }
    }
}
