package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	authPassKey = "auth.password_hash"
	jwtTTL      = 7 * 24 * time.Hour
)

// 认证 API 默认返回简体中文错误提示，与 Web 中文登录页面的提示保持一致。
// 文案变更不影响 HTTP 状态码、账号兼容、JWT 和 Cookie。
const (
	authErrUnauthorized         = "请先登录"
	authErrTokenInvalid         = "登录凭证无效或已过期"
	authErrPasswordAlreadySet   = "管理员密码已设置"
	authErrPasswordEmpty        = "请输入密码"
	authErrNewPasswordEmpty     = "请输入新密码"
	authErrPasswordHash         = "密码加密失败"
	authErrReadFailedPrefix     = "读取失败："
	authErrSaveFailedPrefix     = "保存失败："
	authErrTokenGen             = "生成 Token 失败"
	authErrBadRequest           = "请求格式不正确"
	authErrPasswordNotInit      = "尚未设置管理员密码，请先初始化密码"
	authErrCurrentPasswordWrong = "当前密码不正确"
	authErrBadCredential        = "用户名或密码错误"
)

// signJWT issues a 7-day HS256 token for Kexra.
func signJWT(key []byte) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "Kexra",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}).SignedString(key)
}

// verifyJWT returns true when tokenStr is a valid, non-expired HS256 token.
func verifyJWT(tokenStr string, key []byte) bool {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return key, nil
	})
	return err == nil && t.Valid
}

// extractToken reads the JWT from Authorization: Bearer header,
// kexra_token/legacy artex_token cookies, or ?token= (for SSE connections).
func extractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	for _, cookieName := range []string{"kexra_token", "artex_token"} {
		if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
			return c.Value
		}
	}
	return r.URL.Query().Get("token")
}

// requireAuth wraps h with JWT validation.
// /api/auth/* and /api/health are exempt.
func (s *Server) requireAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/auth/") || p == "/api/health" {
			h.ServeHTTP(w, r)
			return
		}
		tok := extractToken(r)
		if tok == "" {
			writeErr(w, 401, authErrUnauthorized)
			return
		}
		if !verifyJWT(tok, s.jwtKey) {
			writeErr(w, 401, authErrTokenInvalid)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// GET /api/auth/status — reports whether the admin password has been initialised.
func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	hash, _, _ := pg.GetSetting(authPassKey)
	writeJSON(w, 200, map[string]any{"initialized": hash != ""})
}

// POST /api/auth/init — sets the password for the first time; rejected if already set.
func (s *Server) authInit(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	existing, _, err := pg.GetSetting(authPassKey)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, authErrReadFailedPrefix+err.Error())
		return
	}
	if existing != "" {
		writeErr(w, 403, authErrPasswordAlreadySet)
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil || req.Password == "" {
		writeErr(w, 400, authErrPasswordEmpty)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, 500, authErrPasswordHash)
		return
	}
	claimed, err := pg.SetSettingIfEmpty(authPassKey, string(hash))
	if err != nil {
		writeErr(w, 500, authErrSaveFailedPrefix+err.Error())
		return
	}
	if !claimed {
		writeErr(w, http.StatusForbidden, authErrPasswordAlreadySet)
		return
	}
	tok, err := signJWT(s.jwtKey)
	if err != nil {
		writeErr(w, 500, authErrTokenGen)
		return
	}
	writeJSON(w, 200, map[string]any{"token": tok})
}

// POST /api/auth/change-password — changes the admin password. Requires a valid
// token (this route is under /api/auth/* which requireAuth exempts, so the token
// is validated here) AND the current password.
func (s *Server) authChangePassword(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	if !verifyJWT(extractToken(r), s.jwtKey) {
		writeErr(w, 401, authErrUnauthorized)
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, authErrBadRequest)
		return
	}
	if req.NewPassword == "" {
		writeErr(w, 400, authErrNewPasswordEmpty)
		return
	}
	hash, ok, _ := pg.GetSetting(authPassKey)
	if !ok || hash == "" {
		writeErr(w, 403, authErrPasswordNotInit)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.OldPassword)); err != nil {
		writeErr(w, 401, authErrCurrentPasswordWrong)
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, 500, authErrPasswordHash)
		return
	}
	if err := pg.SetSetting(authPassKey, string(newHash)); err != nil {
		writeErr(w, 500, authErrSaveFailedPrefix+err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/auth/login — validates username/password and returns a JWT.
func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, authErrBadRequest)
		return
	}
	if req.Username != "Kexra" && req.Username != "ARTEX" {
		writeErr(w, 401, authErrBadCredential)
		return
	}
	hash, ok, _ := pg.GetSetting(authPassKey)
	if !ok || hash == "" {
		writeErr(w, 403, authErrPasswordNotInit)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		writeErr(w, 401, authErrBadCredential)
		return
	}
	tok, err := signJWT(s.jwtKey)
	if err != nil {
		writeErr(w, 500, authErrTokenGen)
		return
	}
	writeJSON(w, 200, map[string]any{"token": tok})
}
