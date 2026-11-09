package server

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// saveUniqueRootUpload reserves the final filename before writing. O_EXCL treats
// existing files, directories and dangling symlinks as collisions without opening
// or truncating their targets. The caller supplies a validated base filename.
func saveUniqueRootUpload(root *os.Root, header *multipart.FileHeader, dir, name string) (string, error) {
	source, err := header.Open()
	if err != nil {
		return "", err
	}
	defer source.Close()

	extension := filepath.Ext(name)
	stem := strings.TrimSuffix(name, extension)
	for suffix := 0; ; suffix++ {
		candidate := filepath.Join(dir, name)
		if suffix > 0 {
			candidate = filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, suffix, extension))
		}
		file, err := root.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		written, copyErr := io.Copy(file, source)
		if copyErr == nil && written != header.Size {
			copyErr = io.ErrUnexpectedEOF
		}
		closeErr := file.Close()
		if copyErr == nil {
			copyErr = closeErr
		}
		if copyErr != nil {
			if removeErr := root.Remove(candidate); removeErr != nil {
				return "", fmt.Errorf("%w (remove incomplete upload: %v)", copyErr, removeErr)
			}
			return "", copyErr
		}
		return candidate, nil
	}
}
