package server

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Workspace file manager — browse / view / edit / download / upload / delete the
// shared work dir (s.m.dir), where all agents write their artifacts. Every path is
// confined through os.Root, including symlinks and concurrent path changes. All routes sit
// behind requireAuth (see Handler()).

const (
	maxWorkspaceRead   = 2 << 20   // 2 MiB: files bigger than this aren't inlined for view/edit (download instead)
	maxWorkspaceUpload = 512 << 20 // 512 MiB per upload request
)

// 用户可见的默认错误使用简体中文；路径、参数、证据和 err.Error() 原文保留。
const (
	errWsIllegalPath      = "路径无效"
	errWsPathNotFound     = "路径不存在"
	errWsNotDir           = "不是目录"
	errWsFileNotFound     = "文件不存在"
	errWsIsDir            = "该路径是目录，无法作为文件读取"
	errWsTargetIsDir      = "目标路径是目录"
	errWsCannotDeleteRoot = "不能删除工作空间根目录"
	errWsUploadDirMissing = "目标目录不存在"
	errWsUploadParse      = "上传处理失败或超过大小限制： " // 뒤에 err.Error() 를 이어 붙인다
	errWsNoUploadFile     = "没有可上传的文件（表单字段 file）"
)

// wsResolve preserves lexical normalization and existing API paths.
// It is not a filesystem security boundary: all I/O must use wsOpenRoot.
func (s *Server) wsResolve(rel string) (string, bool) {
	base := filepath.Clean(s.m.dir)
	rel = strings.TrimPrefix(strings.TrimSpace(rel), "/")
	clean := filepath.Clean("/" + rel) // e.g. "/a/../../etc" → "/etc" (still rooted at "/")
	abs := filepath.Clean(filepath.Join(base, clean))
	if abs != base && !strings.HasPrefix(abs, base+string(os.PathSeparator)) {
		return "", false
	}
	return abs, true
}

// wsRel renders an absolute path back as a workspace-relative path (forward slashes).
func (s *Server) wsRel(abs string) string {
	base := filepath.Clean(s.m.dir)
	rel, err := filepath.Rel(base, abs)
	if err != nil || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}

func (s *Server) wsRootName(abs string) string {
	name := filepath.FromSlash(s.wsRel(abs))
	if name == "" {
		return "."
	}
	return name
}

func (s *Server) wsOpenRoot(w http.ResponseWriter) (*os.Root, bool) {
	root, err := os.OpenRoot(s.m.dir)
	if err != nil {
		writeErr(w, 500, err.Error())
		return nil, false
	}
	return root, true
}

type wsEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Dir   bool   `json:"dir"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"` // unix millis
}

// GET /api/workspace/list?path=<rel>
func (s *Server) wsList(w http.ResponseWriter, r *http.Request) {
	abs, ok := s.wsResolve(r.URL.Query().Get("path"))
	if !ok {
		writeErr(w, 400, errWsIllegalPath)
		return
	}
	root, ok := s.wsOpenRoot(w)
	if !ok {
		return
	}
	defer root.Close()
	fi, err := root.Stat(s.wsRootName(abs))
	if err != nil {
		writeErr(w, 404, errWsPathNotFound)
		return
	}
	if !fi.IsDir() {
		writeErr(w, 400, errWsNotDir)
		return
	}
	directory, err := root.Open(s.wsRootName(abs))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer directory.Close()
	ents, err := directory.ReadDir(-1)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	out := make([]wsEntry, 0, len(ents))
	for _, e := range ents {
		info, err := root.Lstat(filepath.Join(s.wsRootName(abs), e.Name()))
		if err != nil {
			continue
		}
		out = append(out, wsEntry{
			Name:  e.Name(),
			Path:  s.wsRel(filepath.Join(abs, e.Name())),
			Dir:   e.IsDir(),
			Size:  info.Size(),
			MTime: info.ModTime().UnixMilli(),
		})
	}
	// 디렉터리를 먼저 두고, 각각 이름순으로 정렬한다.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	writeJSON(w, 200, map[string]any{"path": s.wsRel(abs), "entries": out})
}

// GET /api/workspace/read?path=<rel> — inline text for view/edit. Binary or oversize
// files return {binary:true}/{too_large:true} with no content (use download instead).
func (s *Server) wsRead(w http.ResponseWriter, r *http.Request) {
	abs, ok := s.wsResolve(r.URL.Query().Get("path"))
	if !ok {
		writeErr(w, 400, errWsIllegalPath)
		return
	}
	root, ok := s.wsOpenRoot(w)
	if !ok {
		return
	}
	defer root.Close()
	file, err := root.Open(s.wsRootName(abs))
	if err != nil {
		writeErr(w, 404, errWsFileNotFound)
		return
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if fi.IsDir() {
		writeErr(w, 400, errWsIsDir)
		return
	}
	if fi.Size() > maxWorkspaceRead {
		writeJSON(w, 200, map[string]any{"path": s.wsRel(abs), "size": fi.Size(), "too_large": true, "binary": true})
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxWorkspaceRead+1))
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if len(data) > maxWorkspaceRead {
		writeJSON(w, 200, map[string]any{"path": s.wsRel(abs), "size": len(data), "too_large": true, "binary": true})
		return
	}
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		writeJSON(w, 200, map[string]any{"path": s.wsRel(abs), "size": fi.Size(), "binary": true})
		return
	}
	writeJSON(w, 200, map[string]any{"path": s.wsRel(abs), "size": fi.Size(), "binary": false, "content": string(data)})
}

// POST /api/workspace/write  {path, content} — create/overwrite a text file.
func (s *Server) wsWrite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	abs, ok := s.wsResolve(req.Path)
	if !ok || abs == filepath.Clean(s.m.dir) {
		writeErr(w, 400, errWsIllegalPath)
		return
	}
	root, ok := s.wsOpenRoot(w)
	if !ok {
		return
	}
	defer root.Close()
	if fi, err := root.Stat(s.wsRootName(abs)); err == nil && fi.IsDir() {
		writeErr(w, 400, errWsTargetIsDir)
		return
	}
	if err := root.MkdirAll(s.wsRootName(filepath.Dir(abs)), 0o755); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if err := root.WriteFile(s.wsRootName(abs), []byte(req.Content), 0o644); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "path": s.wsRel(abs)})
}

// POST /api/workspace/mkdir  {path}
func (s *Server) wsMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	abs, ok := s.wsResolve(req.Path)
	if !ok || abs == filepath.Clean(s.m.dir) {
		writeErr(w, 400, errWsIllegalPath)
		return
	}
	root, ok := s.wsOpenRoot(w)
	if !ok {
		return
	}
	defer root.Close()
	if err := root.MkdirAll(s.wsRootName(abs), 0o755); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "path": s.wsRel(abs)})
}

// DELETE /api/workspace/delete?path=<rel> — removes a file or a directory tree
// (confined to the work dir; the root itself can't be deleted).
func (s *Server) wsDelete(w http.ResponseWriter, r *http.Request) {
	abs, ok := s.wsResolve(r.URL.Query().Get("path"))
	if !ok {
		writeErr(w, 400, errWsIllegalPath)
		return
	}
	if abs == filepath.Clean(s.m.dir) {
		writeErr(w, 400, errWsCannotDeleteRoot)
		return
	}
	root, ok := s.wsOpenRoot(w)
	if !ok {
		return
	}
	defer root.Close()
	if _, err := root.Stat(s.wsRootName(abs)); err != nil {
		writeErr(w, 404, errWsPathNotFound)
		return
	}
	if err := root.RemoveAll(s.wsRootName(abs)); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// GET /api/workspace/download?path=<rel> — stream a file as an attachment.
func (s *Server) wsDownload(w http.ResponseWriter, r *http.Request) {
	abs, ok := s.wsResolve(r.URL.Query().Get("path"))
	if !ok {
		writeErr(w, 400, errWsIllegalPath)
		return
	}
	root, ok := s.wsOpenRoot(w)
	if !ok {
		return
	}
	defer root.Close()
	file, err := root.Open(s.wsRootName(abs))
	if err != nil {
		writeErr(w, 404, errWsFileNotFound)
		return
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		writeErr(w, 404, errWsFileNotFound)
		return
	}
	name := filepath.Base(abs)
	// RFC 5987 filename* keeps non-ASCII names intact; plain filename is the fallback.
	w.Header().Set("Content-Disposition", "attachment; filename=\""+sanitizeFilename(name)+"\"; filename*=UTF-8''"+url.PathEscape(name))
	// Serve the confined descriptor; ServeFile would reopen an unsafe path.
	http.ServeContent(w, r, name, fi.ModTime(), file)
}

// POST /api/workspace/upload?path=<dir> — multipart form field "file" (one or more).
func (s *Server) wsUpload(w http.ResponseWriter, r *http.Request) {
	dirAbs, ok := s.wsResolve(r.URL.Query().Get("path"))
	if !ok {
		writeErr(w, 400, errWsIllegalPath)
		return
	}
	root, ok := s.wsOpenRoot(w)
	if !ok {
		return
	}
	defer root.Close()
	if fi, err := root.Stat(s.wsRootName(dirAbs)); err != nil || !fi.IsDir() {
		writeErr(w, 400, errWsUploadDirMissing)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxWorkspaceUpload)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeErr(w, 400, errWsUploadParse+err.Error())
		return
	}
	defer r.MultipartForm.RemoveAll()
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		writeErr(w, 400, errWsNoUploadFile)
		return
	}
	saved := 0
	for _, hdr := range files {
		name := filepath.Base(hdr.Filename) // strip any path component
		if name == "" || name == "." || name == ".." {
			continue
		}
		destAbs, okd := s.wsResolve(filepath.Join(s.wsRel(dirAbs), name))
		if !okd {
			continue
		}
		if err := saveUpload(root, hdr, s.wsRootName(destAbs)); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		saved++
	}
	writeJSON(w, 200, map[string]any{"uploaded": saved})
}

func saveUpload(root *os.Root, hdr *multipart.FileHeader, dest string) error {
	src, err := hdr.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := root.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

// sanitizeFilename strips characters unsafe for a Content-Disposition filename token.
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "\r", "")
	return name
}
