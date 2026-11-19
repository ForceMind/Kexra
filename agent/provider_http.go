package agent

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func providerHTTPClient(proxy string) (*http.Client, error) {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok || base == nil {
		return nil, fmt.Errorf("模型 HTTP 默认传输配置无效")
	}
	transport := base.Clone()
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		transport.Proxy = nil // 留空=直连,不回退 HTTP_PROXY/HTTPS_PROXY 环境变量
	} else {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("模型代理地址无效")
		}
		switch proxyURL.Scheme {
		case "http", "https", "socks5":
		case "":
			return nil, fmt.Errorf("模型代理地址必须使用 http、https 或 socks5 协议")
		default:
			return nil, fmt.Errorf("模型代理地址必须使用 http、https 或 socks5 协议")
		}
		if proxyURL.Hostname() == "" {
			return nil, fmt.Errorf("模型代理地址缺少主机")
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{Transport: transport, CheckRedirect: checkProviderRedirect}, nil
}
