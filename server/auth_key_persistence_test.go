package server

import (
    "bytes"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func jwtDirs(t *testing.T) (base, workspace, private string) {
    t.Helper()
    root := t.TempDir()
    base = filepath.Join(root, "app")
    workspace = filepath.Join(root, "workspace")
    private = filepath.Join(root, "private")
    for _, dir := range []string{base, workspace} {
        if err := os.MkdirAll(dir, 0o700); err != nil { t.Fatal(err) }
    }
    t.Setenv("KEXRA_JWT_KEY_DIR", private)
    return
}

func TestPrivateJWTKeyPersistsAcrossStartup(t *testing.T) {
    base, workspace, private := jwtDirs(t)
    a, err := loadOrCreateJWTKey(base, workspace)
    if err != nil { t.Fatal(err) }
    if len(a) < 32 { t.Fatalf("short key: %d", len(a)) }
    path := filepath.Join(private, jwtKeyFilename)
    info, err := os.Stat(path)
    if err != nil { t.Fatal(err) }
    if info.Mode().Perm() != 0o600 {
        t.Fatalf("JWT key permissions = %04o, want 0600", info.Mode().Perm())
    }
    b, err := loadOrCreateJWTKey(base, workspace)
    if err != nil { t.Fatal(err) }
    if !bytes.Equal(a, b) { t.Fatal("JWT key changed across startup") }
    if _, err := os.Stat(filepath.Join(workspace, jwtKeyFilename)); !os.IsNotExist(err) {
        t.Fatalf("JWT key must not be exposed in workspace: %v", err)
    }
}

func TestPrivateJWTKeyMigratesFromBinaryDirectory(t *testing.T) {
    base, workspace, private := jwtDirs(t)
    old := filepath.Join(base, jwtKeyFilename)
    want := []byte(strings.Repeat("b", 32))
    if err := os.WriteFile(old, want, 0o600); err != nil { t.Fatal(err) }
    got, err := loadOrCreateJWTKey(base, workspace)
    if err != nil { t.Fatal(err) }
    if !bytes.Equal(got, want) { t.Fatal("JWT key not preserved") }
    if _, err := os.Stat(old); !os.IsNotExist(err) { t.Fatalf("legacy key left behind: %v", err) }
    if _, err := os.Stat(filepath.Join(private, jwtKeyFilename)); err != nil { t.Fatal(err) }
}

func TestPrivateJWTKeyMigratesOutOfWorkspace(t *testing.T) {
    base, workspace, _ := jwtDirs(t)
    old := filepath.Join(workspace, jwtKeyFilename)
    want := []byte(strings.Repeat("c", 32))
    if err := os.WriteFile(old, want, 0o600); err != nil { t.Fatal(err) }
    got, err := loadOrCreateJWTKey(base, workspace)
    if err != nil { t.Fatal(err) }
    if !bytes.Equal(got, want) { t.Fatal("JWT key changed during migration") }
    if _, err := os.Stat(old); !os.IsNotExist(err) { t.Fatalf("workspace key still exposed: %v", err) }
}

func TestPrivateJWTKeyRejectsBrowsableDirectory(t *testing.T) {
    base, workspace, _ := jwtDirs(t)
    t.Setenv("KEXRA_JWT_KEY_DIR", filepath.Join(workspace, "keys"))
    if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
        t.Fatal("JWT key must never be stored inside browsable workspace")
    }
    t.Setenv("KEXRA_JWT_KEY_DIR", "relative/keys")
    if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
        t.Fatal("JWT key directory must be absolute")
    }
}

func TestPrivateJWTKeyRefusesSilentRotation(t *testing.T) {
    base, workspace, private := jwtDirs(t)
    if err := os.MkdirAll(private, 0o700); err != nil { t.Fatal(err) }
    if err := os.WriteFile(filepath.Join(private, jwtKeyFilename), []byte("damaged"), 0o600); err != nil { t.Fatal(err) }
    if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
        t.Fatal("damaged persistent JWT key must not be silently replaced")
    }
}
