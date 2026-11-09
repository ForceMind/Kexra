package guard

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ForceMind/Kexra/db"
	"github.com/ForceMind/Kexra/intercept"
)

// The driver only supplies stored policy rows and injects database failures.
// All loading, caching, matching, and hook decisions run in production code.
type policyDBFixture struct {
	mu           sync.Mutex
	tool         string
	action       string
	pattern      string
	matchType    string
	enabledJSON  string
	missingTools bool
	noRules      bool
	rulesErr     error
	settingsErr  error
}

type policyConnector struct{ fixture *policyDBFixture }
type policyDriver struct{}
type policyConn struct{ fixture *policyDBFixture }
type policyRows struct {
	columns []string
	values  [][]driver.Value
	next    int
}

func (c policyConnector) Connect(context.Context) (driver.Conn, error) {
	return &policyConn{fixture: c.fixture}, nil
}
func (policyConnector) Driver() driver.Driver { return policyDriver{} }
func (policyDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("fixture requires its connector")
}
func (*policyConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("fixture does not prepare statements")
}
func (*policyConn) Close() error { return nil }
func (*policyConn) Begin() (driver.Tx, error) {
	return nil, errors.New("fixture does not start transactions")
}
func (c *policyConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	f := c.fixture
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case strings.Contains(query, "FROM intercept_rules"):
		if f.rulesErr != nil {
			return nil, f.rulesErr
		}
		rows := &policyRows{columns: strings.Split("id,name,enabled,priority,match_target,match_type,pattern,action,message,timeout_enabled,timeout_seconds,timeout_action,created_at,updated_at", ",")}
		if !f.noRules {
			rows.values = [][]driver.Value{{
				int64(1), "fixture policy", true, int64(100),
				"tool_name", f.matchType, f.pattern, f.action, "fixture policy decision",
				false, int64(0), "deny", time.Now(), time.Now(),
			}}
		}
		return rows, nil
	case strings.Contains(query, "FROM settings"):
		if f.settingsErr != nil {
			return nil, f.settingsErr
		}
		rows := &policyRows{columns: []string{"value"}}
		if len(args) > 0 && args[0].Value == "intercept_enabled_tools" && !f.missingTools {
			rows.values = [][]driver.Value{{f.enabledJSON}}
		}
		return rows, nil
	default:
		return nil, errors.New("fixture received unexpected query")
	}
}
func (c *policyConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if strings.Contains(query, "INSERT INTO settings") && len(args) == 2 && args[0].Value == "intercept_enabled_tools" {
		c.fixture.mu.Lock()
		c.fixture.enabledJSON = args[1].Value.(string)
		c.fixture.missingTools = false
		c.fixture.mu.Unlock()
		return driver.RowsAffected(1), nil
	}
	// Audit logging is best effort and does not determine execution permission.
	return nil, errors.New("fixture does not persist audit rows")
}
func (r *policyRows) Columns() []string { return r.columns }
func (*policyRows) Close() error        { return nil }
func (r *policyRows) Next(dst []driver.Value) error {
	if r.next == len(r.values) {
		return io.EOF
	}
	copy(dst, r.values[r.next])
	r.next++
	return nil
}

func newPolicyGuard(t *testing.T, f *policyDBFixture) (*Guard, *intercept.Interceptor) {
	t.Helper()
	sqlDB := sql.OpenDB(policyConnector{fixture: f})
	t.Cleanup(func() { _ = sqlDB.Close() })
	ic := intercept.New(&db.DB{DB: sqlDB})
	return NewWithInterceptor(ic), ic
}

func denyBashPolicy() *policyDBFixture {
	return &policyDBFixture{
		tool: "Bash", action: "deny", pattern: "Bash", matchType: "string",
		enabledJSON: `["Bash"]`,
	}
}

func policyBlocked(g *Guard, tool string) (bool, string) {
	blocked, message, _ := g.Hooks().PreToolUse(context.Background(), tool, []byte(`{"command":"fixture harmless marker"}`))
	return blocked, message
}

func requirePolicyErrorBlock(t *testing.T, g *Guard, tool string) {
	t.Helper()
	blocked, message := policyBlocked(g, tool)
	if !blocked {
		t.Fatal("tool was allowed while intercept policy could not be loaded")
	}
	if !strings.Contains(message, "工具拦截策略不可用") {
		t.Fatalf("policy failure was not reported as a platform block: %q", message)
	}
}

func TestInterceptPolicyRuleReadFailureBlocks(t *testing.T) {
	f := denyBashPolicy()
	g, ic := newPolicyGuard(t, f)
	if blocked, _ := policyBlocked(g, "Bash"); !blocked {
		t.Fatal("control: configured deny did not block")
	}
	f.mu.Lock()
	f.rulesErr = errors.New("rules database unavailable")
	f.mu.Unlock()
	ic.Invalidate()
	requirePolicyErrorBlock(t, g, "Bash")
}

func TestInterceptPolicyToolConfigReadFailureBlocks(t *testing.T) {
	f := denyBashPolicy()
	f.tool, f.pattern, f.enabledJSON = "fixture_write", "fixture_write", `["fixture_write"]`
	g, ic := newPolicyGuard(t, f)
	if blocked, _ := policyBlocked(g, f.tool); !blocked {
		t.Fatal("control: custom tool deny did not block")
	}
	f.mu.Lock()
	f.settingsErr = errors.New("settings database unavailable")
	f.mu.Unlock()
	ic.Invalidate()
	requirePolicyErrorBlock(t, g, f.tool)
}

func TestInterceptPolicyMalformedToolConfigBlocks(t *testing.T) {
	for _, raw := range []string{"invalid-json", "null", "[null]", `["Bash",null]`, "{}", `["Bash",1]`, ""} {
		t.Run(raw, func(t *testing.T) {
			f := denyBashPolicy()
			f.enabledJSON = raw
			g, ic := newPolicyGuard(t, f)
			requirePolicyErrorBlock(t, g, "Bash")
			if _, err := ic.GetEnabledTools(); err == nil {
				t.Fatal("tool config API hid a malformed saved policy")
			}
		})
	}
}

func TestInterceptPolicyInvalidRegexBlocks(t *testing.T) {
	f := denyBashPolicy()
	f.matchType, f.pattern = "regex", "["
	g, _ := newPolicyGuard(t, f)
	requirePolicyErrorBlock(t, g, "Bash")
}

func TestInterceptPolicyFailedLoadDoesNotPublishPartialCache(t *testing.T) {
	f := denyBashPolicy()
	f.settingsErr = errors.New("settings database unavailable")
	g, ic := newPolicyGuard(t, f)
	requirePolicyErrorBlock(t, g, "Bash")
	f.mu.Lock()
	f.settingsErr = nil
	f.action = "allow"
	f.mu.Unlock()
	decision, matched, err := ic.Match("Bash", []byte(`{}`))
	if err != nil || !matched || decision.Action != "allow" {
		t.Fatalf("Match exposed rules from a failed policy load: (%+v, %v, %v)", decision, matched, err)
	}
	if blocked, message := policyBlocked(g, "Bash"); blocked {
		t.Fatalf("recovered policy reused rules from a failed load: %q", message)
	}
}

func TestInterceptPolicyRuleReadRecoversWithoutInvalidation(t *testing.T) {
	f := denyBashPolicy()
	f.rulesErr = errors.New("rules database unavailable")
	g, _ := newPolicyGuard(t, f)
	requirePolicyErrorBlock(t, g, "Bash")
	f.mu.Lock()
	f.rulesErr = nil
	f.action = "allow"
	f.mu.Unlock()
	if blocked, message := policyBlocked(g, "Bash"); blocked {
		t.Fatalf("policy did not retry a failed load after database recovery: %q", message)
	}
}

func TestInterceptPolicyExplicitEmptyToolListStillAllows(t *testing.T) {
	f := denyBashPolicy()
	f.enabledJSON = "[]"
	g, ic := newPolicyGuard(t, f)
	if blocked, message := policyBlocked(g, "Bash"); blocked {
		t.Fatalf("explicitly disabling interception must remain supported: %q", message)
	}
	names, err := ic.GetEnabledTools()
	if err != nil || names == nil || len(names) != 0 {
		t.Fatalf("explicit empty list = %#v, %v; want a valid empty array", names, err)
	}
}

func TestInterceptPolicyUnsetToolConfigStillUsesDefaults(t *testing.T) {
	f := denyBashPolicy()
	f.missingTools = true
	g, _ := newPolicyGuard(t, f)
	if blocked, _ := policyBlocked(g, "Bash"); !blocked {
		t.Fatal("unset tool config did not keep Bash in the default intercept set")
	}
	if blocked, message := policyBlocked(g, "Read"); blocked {
		t.Fatalf("unset config unexpectedly added Read to the intercept set: %q", message)
	}
}

func TestInterceptPolicyNormalNoMatchStillAllows(t *testing.T) {
	f := denyBashPolicy()
	f.pattern = "another_tool"
	g, _ := newPolicyGuard(t, f)
	if blocked, message := policyBlocked(g, "Bash"); blocked {
		t.Fatalf("ordinary no-match behavior changed: %q", message)
	}
}

func TestInterceptPolicyMatchReturnsLoadError(t *testing.T) {
	f := denyBashPolicy()
	wantErr := errors.New("rules database unavailable")
	f.rulesErr = wantErr
	_, ic := newPolicyGuard(t, f)
	decision, matched, err := ic.Match("Bash", []byte(`{}`))
	if !errors.Is(err, wantErr) || matched || decision.Action != "" {
		t.Fatalf("Match = (%+v, %v, %v); want the loading error, not no-match", decision, matched, err)
	}
}

func TestInterceptPolicyNilToolListPersistsAsExplicitEmptyArray(t *testing.T) {
	f := denyBashPolicy()
	g, ic := newPolicyGuard(t, f)
	if err := ic.SetEnabledTools(nil); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	saved := f.enabledJSON
	f.mu.Unlock()
	if saved != "[]" {
		t.Fatalf("nil tool list saved as %q; want []", saved)
	}
	if blocked, message := policyBlocked(g, "Bash"); blocked {
		t.Fatalf("explicitly disabled interception did not persist: %q", message)
	}
}

func TestInterceptPolicyConcurrentFailedReloadBlocks(t *testing.T) {
	f := denyBashPolicy()
	f.rulesErr = errors.New("rules database unavailable")
	g, _ := newPolicyGuard(t, f)
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			for range 8 {
				if blocked, _ := policyBlocked(g, "Bash"); !blocked {
					t.Error("concurrent tool call bypassed unavailable policy")
				}
			}
		})
	}
	workers.Wait()
}
