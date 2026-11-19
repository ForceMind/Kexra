package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ForceMind/Kexra/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

type authInitTestDB struct {
	admin   *sql.DB
	quoted  string
	appName string
	newDB   func(int) *db.DB
}

// Each fixture has its own settings table. In particular, these tests never
// delete or replace auth.password_hash in a developer's real application schema.
func newAuthInitTestDB(t *testing.T) *authInitTestDB {
	t.Helper()
	dsn, _, err := db.DSN()
	if err != nil {
		t.Skipf("no database config (%v) — skipping", err)
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	if err := admin.PingContext(t.Context()); err != nil {
		t.Fatalf("configured PostgreSQL is unavailable: %v", err)
	}
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	name := "kexra_authinit_" + hex.EncodeToString(suffix[:])
	quoted := pgx.Identifier{name}.Sanitize()
	if _, err := admin.ExecContext(t.Context(), "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + quoted + " CASCADE"); err != nil {
			t.Errorf("drop isolated auth schema: %v", err)
		}
	})
	fixture := &authInitTestDB{admin: admin, quoted: quoted, appName: name + "_"}
	fixture.newDB = func(index int) *db.DB {
		t.Helper()
		cfg, err := pgx.ParseConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		cfg.RuntimeParams["search_path"] = name
		cfg.RuntimeParams["application_name"] = fmt.Sprintf("%s%d", fixture.appName, index)
		cfg.RuntimeParams["statement_timeout"] = "15000"
		pool := stdlib.OpenDB(*cfg)
		pool.SetMaxOpenConns(1)
		t.Cleanup(func() { pool.Close() })
		return &db.DB{DB: pool}
	}
	setup := fixture.newDB(-1)
	if _, err := setup.Exec(`CREATE TABLE settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func authInitRequest(s *Server, password string) *httptest.ResponseRecorder {
	body, err := json.Marshal(map[string]string{"password": password})
	if err != nil {
		panic(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/init", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.authInit(rec, req)
	return rec
}

// Holding SHARE permits every initial SELECT but blocks the INSERT. Wait until
// every contender is at that database write before releasing the lock: the old
// GetSetting + unconditional SetSetting implementation then deterministically
// produces multiple successful initializations, without scheduler-dependent sleeps.
func TestAuthInitConcurrentAcrossServers(t *testing.T) {
	for _, existingEmpty := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing_empty=%t", existingEmpty), func(t *testing.T) {
			fixture := newAuthInitTestDB(t)
			reader := fixture.newDB(-2)
			if existingEmpty {
				if err := reader.SetSetting(authPassKey, ""); err != nil {
					t.Fatal(err)
				}
			}
			lock, err := fixture.admin.BeginTx(t.Context(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Rollback()
			if _, err := lock.Exec("LOCK TABLE " + fixture.quoted + ".settings IN SHARE MODE"); err != nil {
				t.Fatal(err)
			}
			const contenders = 8
			type outcome struct {
				password string
				rec      *httptest.ResponseRecorder
			}
			results := make(chan outcome, contenders)
			key := []byte(strings.Repeat("test-fixture-", 3))
			for i := 0; i < contenders; i++ {
				s := &Server{m: &Manager{pg: fixture.newDB(i)}, jwtKey: key}
				password := fmt.Sprintf("initial-password-%d", i)
				go func() {
					results <- outcome{password: password, rec: authInitRequest(s, password)}
				}()
			}
			deadline := time.Now().Add(10 * time.Second)
			for {
				var waiting int
				err := fixture.admin.QueryRowContext(t.Context(), `
SELECT count(*) FROM pg_stat_activity
WHERE left(application_name, length($1::text)) = $1 AND wait_event_type = 'Lock'`, fixture.appName).Scan(&waiting)
				if err != nil {
					t.Fatal(err)
				}
				if waiting == contenders {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("only %d/%d initializers reached the locked write", waiting, contenders)
				}
				select {
				case <-time.After(10 * time.Millisecond):
				case <-t.Context().Done():
					t.Fatal(t.Context().Err())
				}
			}
			if err := lock.Commit(); err != nil {
				t.Fatal(err)
			}
			winners, winner := 0, ""
			for i := 0; i < contenders; i++ {
				var result outcome
				select {
				case result = <-results:
				case <-time.After(15 * time.Second):
					t.Fatal("initializer did not complete after releasing the table lock")
				}
				switch result.rec.Code {
				case http.StatusOK:
					winners++
					winner = result.password
					var response struct {
						Token string `json:"token"`
					}
					if err := json.Unmarshal(result.rec.Body.Bytes(), &response); err != nil {
						t.Fatal(err)
					}
					if response.Token == "" || !verifyJWT(response.Token, key) {
						t.Errorf("winner did not receive a valid JWT")
					}
				case http.StatusForbidden:
					if !strings.Contains(result.rec.Body.String(), authErrPasswordAlreadySet) {
						t.Errorf("loser response = %s", result.rec.Body.String())
					}
					if strings.Contains(result.rec.Body.String(), "token") {
						t.Errorf("loser received a token: %s", result.rec.Body.String())
					}
				default:
					t.Errorf("initializer status=%d body=%s", result.rec.Code, result.rec.Body.String())
				}
			}
			if winners != 1 {
				t.Fatalf("initialization winners = %d, want exactly 1", winners)
			}
			hash, ok, err := reader.GetSetting(authPassKey)
			if err != nil || !ok {
				t.Fatalf("read winning password hash: ok=%t err=%v", ok, err)
			}
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(winner)); err != nil {
				t.Fatal("stored password does not belong to the sole successful initializer")
			}
			s := &Server{m: &Manager{pg: reader}, jwtKey: key}
			rec := authInitRequest(s, "later-overwrite")
			if rec.Code != http.StatusForbidden {
				t.Fatalf("reinitialization status=%d body=%s", rec.Code, rec.Body.String())
			}
			if got, _, err := reader.GetSetting(authPassKey); err != nil || got != hash {
				t.Fatal("a later initializer changed the winning password")
			}
		})
	}
}

func TestAuthInitDatabaseFailureDoesNotIssueToken(t *testing.T) {
	// A closed handle must fail without contacting any database or minting a JWT.
	pool, err := sql.Open("pgx", "postgres://unused:unused@127.0.0.1/unused?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	s := &Server{m: &Manager{pg: &db.DB{DB: pool}}, jwtKey: []byte(strings.Repeat("test-fixture-", 3))}
	rec := authInitRequest(s, "new-password")
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "token") ||
		!strings.Contains(rec.Body.String(), "读取失败：") {
		t.Fatalf("database failure status=%d body=%s", rec.Code, rec.Body.String())
	}
}
