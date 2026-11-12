package server

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func jwtSecurityWrite(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func jwtSecuritySymlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symbolic links unavailable to this Windows account: %v", err)
		}
		t.Fatal(err)
	}
}

func jwtSecurityUnchanged(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("key fixture changed")
	}
}

func TestPrivateJWTKeyRejectsConfiguredSymlinkIntoWorkspace(t *testing.T) {
	base, workspace, private := jwtDirs(t)
	exposed := filepath.Join(workspace, "keys")
	if err := os.Mkdir(exposed, 0o700); err != nil {
		t.Fatal(err)
	}
	jwtSecuritySymlink(t, exposed, private)
	if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
		t.Fatal("configured directory link must not expose the JWT key in the workspace")
	}
	if _, err := os.Lstat(filepath.Join(exposed, jwtKeyFilename)); !os.IsNotExist(err) {
		t.Fatalf("rejected configuration wrote a workspace key: %v", err)
	}
}

func TestPrivateJWTKeyChecksWorkspacePhysicalDirectory(t *testing.T) {
	base, _, private := jwtDirs(t)
	if err := os.Mkdir(private, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(filepath.Dir(private), "workspace-alias")
	jwtSecuritySymlink(t, private, alias)
	t.Setenv("KEXRA_JWT_KEY_DIR", filepath.Join(private, "keys"))
	if _, err := loadOrCreateJWTKey(base, alias); err == nil {
		t.Fatal("workspace aliases must not bypass the private-directory boundary")
	}
	if _, err := os.Lstat(filepath.Join(private, "keys")); !os.IsNotExist(err) {
		t.Fatalf("rejected physical directory relationship created a key directory: %v", err)
	}
}

func TestPrivateJWTKeyRejectsKeyFileSymlink(t *testing.T) {
	base, workspace, private := jwtDirs(t)
	if err := os.Mkdir(private, 0o700); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(workspace, "exposed-key.txt")
	want := []byte(strings.Repeat("fixture-a", 4))
	jwtSecurityWrite(t, fixture, want)
	path := filepath.Join(private, jwtKeyFilename)
	jwtSecuritySymlink(t, fixture, path)
	if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
		t.Fatal("a key-file symbolic link must be rejected")
	}
	jwtSecurityUnchanged(t, fixture, want)
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("rejected key link must remain intact: %v", err)
	}
}

func TestPrivateJWTKeyConcurrentFirstStartupSharesOneKey(t *testing.T) {
	base, workspace, private := jwtDirs(t)
	const workers = 24
	type result struct {
		key []byte
		err error
	}
	results := make(chan result, workers)
	start := make(chan struct{})
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			key, err := loadOrCreateJWTKey(base, workspace)
			results <- result{key: key, err: err}
		}()
	}
	close(start)
	group.Wait()
	close(results)
	var winner []byte
	for result := range results {
		if result.err != nil {
			t.Errorf("concurrent key initialization failed: %v", result.err)
			continue
		}
		if len(result.key) < 32 {
			t.Error("concurrent startup returned a truncated key")
		}
		if winner == nil {
			winner = result.key
		} else if !bytes.Equal(winner, result.key) {
			t.Error("concurrent startups returned different signing keys")
		}
	}
	if t.Failed() {
		return
	}
	jwtSecurityUnchanged(t, filepath.Join(private, jwtKeyFilename), winner)
	restarted, err := loadOrCreateJWTKey(base, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restarted, winner) {
		t.Fatal("restart changed the concurrently initialized key")
	}
	entries, err := os.ReadDir(private)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != jwtKeyFilename {
		t.Fatal("concurrent key initialization left temporary key files behind")
	}
}

func TestPrivateJWTKeyExistingTargetCleansMatchingLegacyCopies(t *testing.T) {
	base, workspace, private := jwtDirs(t)
	if err := os.Mkdir(private, 0o700); err != nil {
		t.Fatal(err)
	}
	want := []byte(strings.Repeat("fixture-b", 4))
	jwtSecurityWrite(t, filepath.Join(private, jwtKeyFilename), want)
	for _, dir := range []string{base, workspace} {
		jwtSecurityWrite(t, filepath.Join(dir, jwtKeyFilename), want)
	}
	got, err := loadOrCreateJWTKey(base, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("legacy cleanup changed the persistent key")
	}
	for _, dir := range []string{base, workspace} {
		if _, err := os.Lstat(filepath.Join(dir, jwtKeyFilename)); !os.IsNotExist(err) {
			t.Fatalf("matching legacy key was left behind after restart: %v", err)
		}
	}
	jwtSecurityUnchanged(t, filepath.Join(private, jwtKeyFilename), want)
}

func TestPrivateJWTKeyExistingTargetRejectsConflictingLegacyCopy(t *testing.T) {
	for _, location := range []string{"binary", "workspace"} {
		t.Run(location, func(t *testing.T) {
			base, workspace, private := jwtDirs(t)
			if err := os.Mkdir(private, 0o700); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(private, jwtKeyFilename)
			current := []byte(strings.Repeat("fixture-c", 4))
			legacy := []byte(strings.Repeat("fixture-d", 4))
			jwtSecurityWrite(t, target, current)
			dir := base
			if location == "workspace" {
				dir = workspace
			}
			old := filepath.Join(dir, jwtKeyFilename)
			jwtSecurityWrite(t, old, legacy)
			if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
				t.Fatal("conflicting legacy keys must not be ignored on restart")
			}
			jwtSecurityUnchanged(t, target, current)
			jwtSecurityUnchanged(t, old, legacy)
		})
	}
}

func TestPrivateJWTKeyConflictingLegacyCopiesDoNotCreateTarget(t *testing.T) {
	base, workspace, private := jwtDirs(t)
	first := []byte(strings.Repeat("fixture-g", 4))
	second := []byte(strings.Repeat("fixture-h", 4))
	oldBinary := filepath.Join(base, jwtKeyFilename)
	oldWorkspace := filepath.Join(workspace, jwtKeyFilename)
	jwtSecurityWrite(t, oldBinary, first)
	jwtSecurityWrite(t, oldWorkspace, second)
	if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
		t.Fatal("conflicting legacy keys must be resolved before publishing a target")
	}
	jwtSecurityUnchanged(t, oldBinary, first)
	jwtSecurityUnchanged(t, oldWorkspace, second)
	if _, err := os.Lstat(filepath.Join(private, jwtKeyFilename)); !os.IsNotExist(err) {
		t.Fatalf("conflicting legacy copies created a destination key: %v", err)
	}
}

func TestPrivateJWTKeyRetriesCleanupAfterMigrationFailure(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires POSIX directory permissions enforced for an unprivileged user")
	}
	base, workspace, private := jwtDirs(t)
	want := []byte(strings.Repeat("fixture-i", 4))
	old := filepath.Join(base, jwtKeyFilename)
	jwtSecurityWrite(t, old, want)
	if err := os.Chmod(base, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(base, 0o700) })
	if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
		t.Fatal("failed removal of a legacy key must fail startup")
	}
	jwtSecurityUnchanged(t, filepath.Join(private, jwtKeyFilename), want)
	jwtSecurityUnchanged(t, old, want)
	if err := os.Chmod(base, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := loadOrCreateJWTKey(base, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("cleanup retry changed the signing key")
	}
	if _, err := os.Lstat(old); !os.IsNotExist(err) {
		t.Fatalf("startup with an existing target skipped failed migration cleanup: %v", err)
	}
}

func TestPrivateJWTKeyRejectsLegacyFileSymlink(t *testing.T) {
	base, workspace, private := jwtDirs(t)
	fixture := filepath.Join(workspace, "source-key.txt")
	want := []byte(strings.Repeat("fixture-e", 4))
	jwtSecurityWrite(t, fixture, want)
	legacy := filepath.Join(base, jwtKeyFilename)
	jwtSecuritySymlink(t, fixture, legacy)
	if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
		t.Fatal("legacy key-file symbolic links must not be migrated")
	}
	jwtSecurityUnchanged(t, fixture, want)
	if _, err := os.Lstat(filepath.Join(private, jwtKeyFilename)); !os.IsNotExist(err) {
		t.Fatalf("rejected legacy link created a private key: %v", err)
	}
}

func TestPrivateJWTKeyRejectsFilesystemRoot(t *testing.T) {
	base, workspace, _ := jwtDirs(t)
	root := filepath.VolumeName(base) + string(filepath.Separator)
	t.Setenv("KEXRA_JWT_KEY_DIR", root)
	if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
		t.Fatal("filesystem or volume roots must not be used as the key directory")
	}
}

func TestPrivateJWTKeyRejectsFilesystemRootAlias(t *testing.T) {
	base, workspace, private := jwtDirs(t)
	root := filepath.VolumeName(base) + string(filepath.Separator)
	jwtSecuritySymlink(t, root, private)
	if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
		t.Fatal("a filesystem-root alias must not be opened as the private key directory")
	}
}

func TestPrivateJWTKeySafeDirectoryAliasStillPersists(t *testing.T) {
	base, workspace, private := jwtDirs(t)
	physical := filepath.Join(filepath.Dir(private), "physical-private")
	if err := os.Mkdir(physical, 0o700); err != nil {
		t.Fatal(err)
	}
	jwtSecuritySymlink(t, physical, private)
	first, err := loadOrCreateJWTKey(base, workspace)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadOrCreateJWTKey(base, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("safe private-directory alias changed the signing key")
	}
	jwtSecurityUnchanged(t, filepath.Join(physical, jwtKeyFilename), first)
}

func TestPrivateJWTKeyRejectsOversizedFilesWithoutChangingThem(t *testing.T) {
	for _, location := range []string{"private", "binary", "workspace"} {
		t.Run(location, func(t *testing.T) {
			base, workspace, private := jwtDirs(t)
			if err := os.Mkdir(private, 0o700); err != nil {
				t.Fatal(err)
			}
			valid := []byte(strings.Repeat("fixture-f", 4))
			target := filepath.Join(private, jwtKeyFilename)
			jwtSecurityWrite(t, target, valid)
			dir := private
			if location == "binary" {
				dir = base
			} else if location == "workspace" {
				dir = workspace
			}
			oversized := []byte(strings.Repeat("x", maxJWTKeySize+1))
			path := filepath.Join(dir, jwtKeyFilename)
			jwtSecurityWrite(t, path, oversized)
			if _, err := loadOrCreateJWTKey(base, workspace); err == nil {
				t.Fatal("oversized signing-key files must be rejected")
			}
			jwtSecurityUnchanged(t, path, oversized)
			if location != "private" {
				jwtSecurityUnchanged(t, target, valid)
			}
		})
	}
}

func TestPrivateJWTKeyDefaultBinaryDirectoryStillPersists(t *testing.T) {
	base, workspace, _ := jwtDirs(t)
	t.Setenv("KEXRA_JWT_KEY_DIR", "")
	first, err := loadOrCreateJWTKey(base, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) < 32 {
		t.Fatal("default signing key is truncated")
	}
	second, err := loadOrCreateJWTKey(base, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("default binary-directory key changed across startup")
	}
	jwtSecurityUnchanged(t, filepath.Join(base, jwtKeyFilename), first)
	if _, err := os.Lstat(filepath.Join(workspace, jwtKeyFilename)); !os.IsNotExist(err) {
		t.Fatalf("default startup exposed a workspace key: %v", err)
	}
}
