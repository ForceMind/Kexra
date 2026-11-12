package server

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	jwtKeyFilename = "jwt.key"
	keyChars       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	maxJWTKeySize  = 64 << 10
)

// loadOrCreateJWTKey keeps signing material outside the browsable workspace.
// Directory identities and all subsequent I/O are bound to open directory
// handles. Parent directories and mounts must remain under trusted control;
// os.Root is not a sandbox against a process running with the same privileges.
func loadOrCreateJWTKey(keyDir, dataDir string) ([]byte, error) {
	workspace, err := os.OpenRoot(dataDir)
	if err != nil {
		return nil, fmt.Errorf("检查数据目录失败：%w", err)
	}
	defer workspace.Close()
	workspaceInfo, err := workspace.Stat(".")
	if err != nil {
		return nil, fmt.Errorf("检查数据目录失败：%w", err)
	}
	configured := strings.TrimSpace(os.Getenv("KEXRA_JWT_KEY_DIR"))
	destination := keyDir
	if configured != "" {
		if !filepath.IsAbs(configured) {
			return nil, fmt.Errorf("KEXRA_JWT_KEY_DIR 必须使用绝对路径")
		}
		destination = filepath.Clean(configured)
		if filepath.Dir(destination) == destination {
			return nil, fmt.Errorf("KEXRA_JWT_KEY_DIR 不能指向文件系统根目录")
		}
	}
	root, err := openJWTDirectory(destination, workspaceInfo, configured != "")
	if err != nil {
		return nil, fmt.Errorf("检查 JWT 密钥目录失败：%w", err)
	}
	defer root.Close()
	rootInfo, err := root.Stat(".")
	if err != nil {
		return nil, fmt.Errorf("检查 JWT 密钥目录失败：%w", err)
	}
	if configured != "" {
		// Chmod through the already opened directory; Root.Chmod(path) has a
		// documented symlink race on Unix.
		directory, err := root.Open(".")
		if err != nil {
			return nil, fmt.Errorf("打开 JWT 密钥目录失败：%w", err)
		}
		chmodErr := directory.Chmod(0o700)
		closeErr := directory.Close()
		if chmodErr != nil {
			return nil, fmt.Errorf("设置 JWT 密钥目录权限失败：%w", chmodErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("关闭 JWT 密钥目录失败：%w", closeErr)
		}
	}

	key, _, err := readJWTKey(root, true)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("读取 JWT 密钥失败：%w", err)
	}
	if errors.Is(err, os.ErrNotExist) {
		key = nil
	}
	type legacyKey struct {
		root *os.Root
		info os.FileInfo
		key  []byte
	}
	var legacy []legacyKey
	// Inspect both old locations on every startup. In particular, an earlier
	// copy followed by failed cleanup must never skip cleanup after restart.
	for _, dir := range []string{keyDir, dataDir} {
		oldRoot, err := openJWTDirectory(dir, nil, false)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("打开旧 JWT 密钥目录失败：%w", err)
		}
		defer oldRoot.Close()
		oldDirInfo, err := oldRoot.Stat(".")
		if err != nil {
			return nil, fmt.Errorf("检查旧 JWT 密钥目录失败：%w", err)
		}
		if os.SameFile(rootInfo, oldDirInfo) {
			continue // Never remove the destination through an alias.
		}
		oldKey, info, err := readJWTKey(oldRoot, false)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("读取旧 JWT 密钥失败：%w", err)
		}
		if key == nil {
			key = oldKey
		} else if !bytes.Equal(key, oldKey) {
			return nil, fmt.Errorf("JWT 密钥与旧路径内容冲突，拒绝覆盖或清理")
		}
		legacy = append(legacy, legacyKey{oldRoot, info, oldKey})
	}
	if key == nil {
		key = make([]byte, 32)
		for i := range key {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(keyChars))))
			if err != nil {
				return nil, fmt.Errorf("生成 JWT 密钥失败：%w", err)
			}
			key[i] = keyChars[n.Int64()]
		}
	}
	persisted, err := publishJWTKey(root, key)
	if err != nil {
		return nil, fmt.Errorf("持久化 JWT 密钥失败：%w", err)
	}
	for _, old := range legacy {
		if !bytes.Equal(persisted, old.key) {
			return nil, fmt.Errorf("并发写入的 JWT 密钥与旧路径内容冲突，拒绝清理")
		}
		current, info, err := readJWTKey(old.root, false)
		if errors.Is(err, os.ErrNotExist) {
			continue // Another startup may have completed the same migration.
		}
		if err != nil || !os.SameFile(old.info, info) || !bytes.Equal(current, persisted) {
			return nil, fmt.Errorf("旧 JWT 密钥在迁移期间发生变化，拒绝清理")
		}
		if err := old.root.Remove(jwtKeyFilename); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("JWT 密钥已保存，但无法清理旧路径：%w", err)
		}
	}
	return persisted, nil
}

// canonicalJWTPath resolves the existing ancestor before appending any missing
// suffix. This retains normal OS aliases such as macOS /var -> /private/var.
// The result is only a hint: openJWTDirectory validates and pins every directory.
func canonicalJWTPath(name string) (string, error) {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	current := filepath.Clean(absolute)
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) || filepath.Dir(current) == current {
			return "", err
		}
		missing = append(missing, filepath.Base(current))
		current = filepath.Dir(current)
	}
}

func openJWTDirectory(name string, workspaceInfo os.FileInfo, create bool) (*os.Root, error) {
	absolute, err := canonicalJWTPath(name)
	if err != nil {
		return nil, err
	}
	if workspaceInfo != nil && filepath.Dir(absolute) == absolute {
		return nil, fmt.Errorf("JWT 密钥目录不能指向文件系统根目录")
	}
	volumeRoot := filepath.VolumeName(absolute) + string(filepath.Separator)
	current, err := os.OpenRoot(volumeRoot)
	if err != nil {
		return nil, err
	}
	failed := true
	defer func() {
		if failed {
			current.Close()
		}
	}()
	relative, err := filepath.Rel(volumeRoot, absolute)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(relative, string(filepath.Separator))
	for i := 0; ; i++ {
		info, err := current.Stat(".")
		if err != nil {
			return nil, err
		}
		if workspaceInfo != nil && os.SameFile(workspaceInfo, info) {
			return nil, fmt.Errorf("JWT 密钥目录不能位于可浏览的数据目录内")
		}
		if i == len(parts) || relative == "." {
			failed = false
			return current, nil
		}
		component := parts[i]
		info, err = current.Lstat(component)
		if errors.Is(err, os.ErrNotExist) && create {
			if err := current.Mkdir(component, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
				return nil, err
			}
			info, err = current.Lstat(component)
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("JWT 路径组件不是可信目录")
		}
		next, err := current.OpenRoot(component)
		if err != nil {
			return nil, err
		}
		opened, err := next.Stat(".")
		if err != nil || !os.SameFile(info, opened) {
			next.Close()
			return nil, fmt.Errorf("JWT 路径组件在打开期间发生变化")
		}
		current.Close()
		current = next
	}
}

func readJWTKey(root *os.Root, securePermissions bool) ([]byte, os.FileInfo, error) {
	info, err := root.Lstat(jwtKeyFilename)
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("JWT 密钥必须是普通文件，不能使用符号链接")
	}
	if info.Size() > maxJWTKeySize {
		return nil, nil, fmt.Errorf("JWT 密钥文件超过大小限制")
	}
	file, err := root.Open(jwtKeyFilename)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, nil, fmt.Errorf("JWT 密钥在打开期间发生变化")
	}
	if opened.Size() > maxJWTKeySize {
		return nil, nil, fmt.Errorf("JWT 密钥文件超过大小限制")
	}
	if securePermissions {
		if err := file.Chmod(0o600); err != nil {
			return nil, nil, err
		}
	}
	content, err := io.ReadAll(io.LimitReader(file, maxJWTKeySize+1))
	if err != nil {
		return nil, nil, err
	}
	if len(content) > maxJWTKeySize {
		return nil, nil, fmt.Errorf("JWT 密钥文件超过大小限制")
	}
	key := []byte(strings.TrimSpace(string(content)))
	if len(key) < 32 {
		return nil, nil, fmt.Errorf("JWT 密钥内容无效，拒绝静默轮换")
	}
	return key, opened, nil
}

func publishJWTKey(root *os.Root, key []byte) ([]byte, error) {
	if current, _, err := readJWTKey(root, true); err == nil {
		return current, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	temporary := fmt.Sprintf(".jwt-key-%x.tmp", nonce)
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	defer root.Remove(temporary)
	n, writeErr := file.Write(key)
	if writeErr == nil && n != len(key) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return nil, writeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	// Link publishes complete bytes without replacing another startup's key.
	// Unsupported filesystems fail closed rather than falling back to overwrite.
	if err := root.Link(temporary, jwtKeyFilename); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	if runtime.GOOS != "windows" {
		// Windows does not expose portable directory fsync through os.File.
		directory, err := root.Open(".")
		if err != nil {
			return nil, err
		}
		syncErr := directory.Sync()
		closeErr := directory.Close()
		if syncErr != nil {
			return nil, syncErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	persisted, _, err := readJWTKey(root, true)
	return persisted, err
}
