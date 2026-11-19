package mcphttp

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const maxSSEEndpointFrame = 64 << 10

// MCP credentials and session data are scoped to the configured origin. This
// policy preserves explicitly configured local/private services; it is not an
// address allowlist or an operating-system network sandbox.
func parseMCPURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || !validMCPURL(u) {
		return nil, errors.New("MCP 地址必须为绝对 HTTP(S) 地址，不能包含用户信息或片段")
	}
	return u, nil
}

func validMCPURL(u *url.URL) bool {
	if u == nil || !u.IsAbs() || u.Opaque != "" || u.Hostname() == "" ||
		u.User != nil || u.Fragment != "" {
		return false
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return false
	}
	if port := u.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return false
		}
	}
	return true
}

func mcpPort(u *url.URL) string {
	if port := u.Port(); port != "" {
		value, _ := strconv.Atoi(port)
		return strconv.Itoa(value)
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	return "80"
}

func sameMCPOrigin(a, b *url.URL) bool {
	return validMCPURL(a) && validMCPURL(b) &&
		strings.EqualFold(a.Scheme, b.Scheme) &&
		mcpHost(a) == mcpHost(b) && mcpPort(a) == mcpPort(b)
}

// Fold ASCII DNS letters only. Unicode case folding can equate different IDNA
// domains (for example Greek sigma variants). Keep non-ASCII bytes and IPv6
// zone identifiers exact rather than widening the configured credential scope.
func mcpHost(u *url.URL) string {
	host := u.Hostname()
	zone := ""
	if i := strings.LastIndexByte(host, '%'); i >= 0 {
		host, zone = host[:i], host[i:]
	}
	bytes := []byte(host)
	for i, value := range bytes {
		if value >= 'A' && value <= 'Z' {
			bytes[i] += 'a' - 'A'
		}
	}
	return string(bytes) + zone
}

func checkMCPRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("MCP 重定向次数过多")
	}
	if req == nil || len(via) == 0 || via[0] == nil ||
		!sameMCPOrigin(via[0].URL, req.URL) {
		return errors.New("禁止 MCP 重定向到其他来源或不安全地址")
	}
	return nil
}

// readMCPEndpoint accepts only a complete, named SSE endpoint event. Ordinary
// message/log data cannot redirect a credential-bearing POST. ReadSlice keeps
// later SSE messages in the shared reader instead of losing scanner lookahead.
func readMCPEndpoint(r *bufio.Reader, base string) (string, error) {
	baseURL, err := parseMCPURL(base)
	if err != nil {
		return "", err
	}
	var event string
	var data strings.Builder
	hasData := false
	frameBytes := 0
	for {
		line, err := readMCPEndpointLine(r)
		if err != nil {
			return "", err
		}
		frameBytes += len(line) + 1
		if frameBytes > maxSSEEndpointFrame {
			return "", errors.New("MCP SSE endpoint 事件超过大小限制")
		}
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			if event == "endpoint" && hasData {
				candidate := strings.TrimSpace(data.String())
				if candidate == "" {
					return "", errors.New("MCP SSE endpoint 地址为空")
				}
				ref, err := url.Parse(candidate)
				if err != nil {
					return "", errors.New("MCP SSE endpoint 地址无效")
				}
				endpoint := baseURL.ResolveReference(ref)
				if !sameMCPOrigin(baseURL, endpoint) {
					return "", errors.New("MCP SSE endpoint 必须与配置地址同源")
				}
				return endpoint.String(), nil
			}
			event = ""
			data.Reset()
			hasData = false
			frameBytes = 0
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			event = value
		case "data":
			hasData = true
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(value)
		}
	}
}

func readMCPEndpointLine(r *bufio.Reader) (string, error) {
	var line []byte
	for {
		fragment, err := r.ReadSlice('\n')
		if len(line)+len(fragment) > maxSSEEndpointFrame {
			return "", errors.New("MCP SSE endpoint 行超过大小限制")
		}
		line = append(line, fragment...)
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if err != nil {
			if errors.Is(err, io.EOF) && len(line) > 0 {
				return "", io.ErrUnexpectedEOF
			}
			return "", err
		}
		return strings.TrimSuffix(string(line), "\n"), nil
	}
}
