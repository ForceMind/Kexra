package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// settingsTestDB uses an isolated schema instead of changing the application's
// real auth.password_hash or other settings in a developer database.
func settingsTestDB(t *testing.T) (*DB, func() *DB) {
	t.Helper()
	dsn := testDSN(t)
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
	schema := "kexra_settings_" + hex.EncodeToString(suffix[:])
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.ExecContext(t.Context(), "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + quoted + " CASCADE"); err != nil {
			t.Errorf("drop isolated settings schema: %v", err)
		}
	})
	newHandle := func() *DB {
		t.Helper()
		cfg, err := pgx.ParseConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		cfg.RuntimeParams["search_path"] = schema
		pool := stdlib.OpenDB(*cfg)
		pool.SetMaxOpenConns(1)
		t.Cleanup(func() { pool.Close() })
		return &DB{DB: pool}
	}
	d := newHandle()
	// This is the settings table contract from schema.sql; no seeds, application
	// tables, user passwords or running services are changed by these tests.
	if _, err := d.Exec(`CREATE TABLE settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		t.Fatal(err)
	}
	return d, newHandle
}

func TestSetSettingIfEmptyConcurrent(t *testing.T) {
	for _, existingEmpty := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing_empty=%t", existingEmpty), func(t *testing.T) {
			d, newHandle := settingsTestDB(t)
			const key = "auth.password_hash"
			if existingEmpty {
				if err := d.SetSetting(key, ""); err != nil {
					t.Fatal(err)
				}
			}
			const contenders = 16
			type outcome struct {
				value string
				won   bool
				err   error
			}
			results := make(chan outcome, contenders)
			start := make(chan struct{})
			var ready sync.WaitGroup
			ready.Add(contenders)
			for i := 0; i < contenders; i++ {
				handle := newHandle()
				value := fmt.Sprintf("contender-%d", i)
				go func() {
					ready.Done()
					<-start
					won, err := handle.SetSettingIfEmpty(key, value)
					results <- outcome{value: value, won: won, err: err}
				}()
			}
			ready.Wait()
			close(start)
			winners, winner := 0, ""
			for i := 0; i < contenders; i++ {
				result := <-results
				if result.err != nil {
					t.Errorf("claim %s: %v", result.value, result.err)
				}
				if result.won {
					winners++
					winner = result.value
				}
			}
			if winners != 1 {
				t.Fatalf("initialization winners = %d, want exactly 1", winners)
			}
			got, ok, err := d.GetSetting(key)
			if err != nil || !ok || got != winner {
				t.Fatalf("stored value = %q, ok=%t err=%v; winner=%q", got, ok, err, winner)
			}
			if changed, err := d.SetSettingIfEmpty(key, "later-overwrite"); err != nil || changed {
				t.Fatalf("existing setting overwritten: changed=%t err=%v", changed, err)
			}
			if got, _, err := d.GetSetting(key); err != nil || got != winner {
				t.Fatalf("winner changed after rejected claim: %q, %v", got, err)
			}
		})
	}
}

func TestSetSettingIfEmptyPreservesExistingValue(t *testing.T) {
	d, _ := settingsTestDB(t)
	if err := d.SetSetting("auth.password_hash", "already-initialized"); err != nil {
		t.Fatal(err)
	}
	if changed, err := d.SetSettingIfEmpty("auth.password_hash", "replacement"); err != nil || changed {
		t.Fatalf("existing value must be retained: changed=%t err=%v", changed, err)
	}
	if got, _, err := d.GetSetting("auth.password_hash"); err != nil || got != "already-initialized" {
		t.Fatalf("existing value changed: %q, %v", got, err)
	}
}

func TestSetSettingIfEmptyPropagatesDatabaseError(t *testing.T) {
	// Closing the handle before Exec verifies error propagation without contacting
	// a database or requiring a configured DSN.
	pool, err := sql.Open("pgx", "postgres://unused:unused@127.0.0.1/unused?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	d := &DB{DB: pool}
	if changed, err := d.SetSettingIfEmpty("auth.password_hash", "replacement"); err == nil || changed {
		t.Fatalf("closed DB claimed success: changed=%t err=%v", changed, err)
	}
}
