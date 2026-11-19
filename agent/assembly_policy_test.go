package agent

import (
	"context"
	"encoding/json"
	"testing"

	actool "github.com/Autumn-27/norma/tool"
)

// Exercise the real assembly pipeline, including findingWorkflowTools' nil-copy
// branch, rather than only checking the resolver's immediate return value.
func TestResolvedEmptyToolsNeverRestoreSDKDefaults(t *testing.T) {
	oldAugment, oldResolve, oldBinding := ToolAugment, ToolResolve, FindingTrafficBindingEnabled
	t.Cleanup(func() { ToolAugment, ToolResolve, FindingTrafficBindingEnabled = oldAugment, oldResolve, oldBinding })
	fixture := actool.Build(actool.Spec{Name: "fixture_not_permitted", Run: func(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
		t.Fatal("disallowed fixture tool executed")
		return actool.Result{}, nil
	}})
	for _, binding := range []bool{false, true} {
		for _, nilSelection := range []bool{false, true} {
			t.Run(fmtAssemblyCase(binding, nilSelection), func(t *testing.T) {
				cleaned := 0
				FindingTrafficBindingEnabled = func() bool { return binding }
				ToolAugment = func(context.Context, string) ([]actool.CoreTool, DeferredInfo, func()) {
					return []actool.CoreTool{fixture}, DeferredInfo{
						Deferred: []string{fixture.Name()}, GlobalNames: []string{fixture.Name()},
						Unlock: actool.NewUnlockSet(fixture.Name()), UnlockSkill: func(string) {},
					}, func() { cleaned++ }
				}
				ToolResolve = func(context.Context, string, []actool.CoreTool) []actool.CoreTool {
					if nilSelection {
						return nil
					}
					return []actool.CoreTool{}
				}
				tools, def, cleanup := AugmentTools(context.Background(), "worker", []actool.CoreTool{fixture})
				cleanup()
				if tools == nil || len(tools) != 0 {
					t.Fatal("resolved empty selection can restore default tools")
				}
				if len(def.Deferred) != 0 || len(def.GlobalNames) != 0 || def.Unlock != nil || def.UnlockSkill != nil || def.FindingGuidance != "" {
					t.Fatal("denied tool discovery metadata survived resolution")
				}
				if cleaned != 1 {
					t.Fatal("augmented resources were not cleaned")
				}
			})
		}
	}
}

func fmtAssemblyCase(binding, nilSelection bool) string {
	if binding {
		if nilSelection {
			return "binding_on_nil"
		}
		return "binding_on_empty"
	}
	if nilSelection {
		return "binding_off_nil"
	}
	return "binding_off_empty"
}

func TestResolvedPermittedToolStillAssembles(t *testing.T) {
	oldAugment, oldResolve, oldBinding := ToolAugment, ToolResolve, FindingTrafficBindingEnabled
	t.Cleanup(func() { ToolAugment, ToolResolve, FindingTrafficBindingEnabled = oldAugment, oldResolve, oldBinding })
	ToolAugment = nil
	FindingTrafficBindingEnabled = func() bool { return true }
	ToolResolve = func(_ context.Context, _ string, tools []actool.CoreTool) []actool.CoreTool { return tools }
	fixture := actool.Build(actool.Spec{Name: "fixture_permitted", Run: func(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
		return actool.Text("ok"), nil
	}})
	tools, _, cleanup := AugmentTools(context.Background(), "worker", []actool.CoreTool{fixture})
	defer cleanup()
	if len(tools) != 1 || tools[0].Name() != fixture.Name() {
		t.Fatal("permitted tool was dropped")
	}
	result, err := tools[0].Call(context.Background(), nil, nil)
	if err != nil || result.Flatten() != "ok" {
		t.Fatal("permitted tool no longer executes")
	}
}
