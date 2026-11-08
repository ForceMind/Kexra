package agent

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Model credentials, prompts and session headers are scoped to one origin.
// Explicit HTTP/private deployments remain supported; redirecting to a different
// endpoint requires changing the provider's configured base URL.
func checkProviderRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("模型请求重定向次数过多")
	}
	if req == nil || len(via) == 0 || via[0] == nil ||
		!sameProviderOrigin(via[0].URL, req.URL) {
		return errors.New("禁止模型请求重定向到其他来源或不安全地址")
	}
	return nil
}

func sameProviderOrigin(a, b *url.URL) bool {
	return validProviderRedirectURL(a) && validProviderRedirectURL(b) &&
		strings.EqualFold(a.Scheme, b.Scheme) &&
		providerHost(a) == providerHost(b) && providerPort(a) == providerPort(b)
}

func validProviderRedirectURL(u *url.URL) bool {
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

func providerPort(u *url.URL) string {
	if port := u.Port(); port != "" {
		value, _ := strconv.Atoi(port)
		return strconv.Itoa(value)
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	return "80"
}

func providerHost(u *url.URL) string {
	host, zone := u.Hostname(), ""
	if index := strings.LastIndexByte(host, '%'); index >= 0 {
		host, zone = host[:index], host[index:]
	}
	// Unicode case folding can merge different IDNA domains. IPv6 zone names
	// are also exact identifiers, so fold ASCII DNS letters only.
	data := []byte(host)
	for i, value := range data {
		if value >= 'A' && value <= 'Z' {
			data[i] += 'a' - 'A'
		}
	}
	return string(data) + zone
}
