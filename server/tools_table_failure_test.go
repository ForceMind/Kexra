package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	actool "github.com/Autumn-27/norma/tool"
	"github.com/ForceMind/Kexra/agent"
	"github.com/ForceMind/Kexra/db"
)

func TestToolsTableReadFailureDoesNotRestoreTools(t *testing.T) {
	oldResolve, oldAugment, oldBinding := agent.ToolResolve, agent.ToolAugment, agent.FindingTrafficBindingEnabled
	t.Cleanup(func() {
		agent.ToolResolve, agent.ToolAugment, agent.FindingTrafficBindingEnabled = oldResolve, oldAugment, oldBinding
	})
	// sql.Open does not dial; closing it first injects a real database/sql error
	// without connecting to a developer or production database.
	pool, err := sql.Open("pgx", "postgres://unused:unused@127.0.0.1/unused?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	wireTools(&db.DB{DB: pool}, nil)
	fixture := actool.Build(actool.Spec{Name: "fixture_disabled", Run: func(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
		t.Fatal("tool executed after tool-table failure")
		return actool.Result{}, nil
	}})
	base := []actool.CoreTool{fixture}
	selection := agent.ToolResolve(context.Background(), "worker", base)
	if selection == nil || len(selection) != 0 {
		t.Fatal("catalog failure returned default tools")
	}
	if len(base) != 1 || base[0] != fixture {
		t.Fatal("catalog failure mutated the caller's base")
	}

	cleaned := 0
	agent.ToolAugment = func(context.Context, string) ([]actool.CoreTool, agent.DeferredInfo, func()) {
		return []actool.CoreTool{fixture}, agent.DeferredInfo{Deferred: []string{fixture.Name()}, GlobalNames: []string{fixture.Name()}},
			func() { cleaned++ }
	}
	agent.FindingTrafficBindingEnabled = func() bool { return true }
	selection, def, cleanup := agent.AugmentTools(context.Background(), "worker", base)
	cleanup()
	if selection == nil || len(selection) != 0 {
		t.Fatal("assembly resurrected tools after catalog failure")
	}
	if len(def.Deferred) != 0 || len(def.GlobalNames) != 0 {
		t.Fatal("assembly advertised tools after catalog failure")
	}
	if cleaned != 1 {
		t.Fatal("assembly leaked augmented resources")
	}
}
