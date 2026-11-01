// Created by Yanjunhui
package attachment

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" ||
		(u.Port() != "" && u.Port() != "443") || !allowedHost(u.Hostname()) {
		return fmt.Errorf("attachment URL must use an approved HTTPS host, port 443 and no userinfo or fragment")
	}
	return nil
}

func allowedHost(host string) bool {
	host = strings.ToLower(host)
	for _, suffix := range []string{"oaiusercontent.com", "openai.com", "chatgpt.com",
		"oaistatic.com", "amazonaws.com", "blob.core.windows.net", "googleapis.com"} {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

func redirect(req *http.Request, via []*http.Request) error {
	if len(via) > 5 {
		return fmt.Errorf("attachment exceeds five redirects")
	}
	// A signed URL must not become the next server's Referer.
	// 中文：避免把签名链接作为 Referer 发给跳转目标。
	req.Header.Del("Referer")
	return validateURL(req.URL.String())
}

func publicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, block := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24",
		"192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
		"240.0.0.0/4", "2001:db8::/32", "64:ff9b::/96", "2002::/16"} {
		if netip.MustParsePrefix(block).Contains(ip) {
			return false
		}
	}
	return true
}

func publicDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port != "443" || !allowedHost(host) {
		return nil, fmt.Errorf("attachment destination is not allowed")
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("attachment host lookup failed")
	}
	for _, ip := range ips {
		if !publicIP(ip) {
			return nil, fmt.Errorf("attachment host resolved to a non-public address")
		}
	}
	// Connect to the validated IP, preserving hostname-based TLS verification.
	// 中文：连接已校验的 IP，TLS 仍校验原域名，避免 DNS 二次解析绕过。
	dialer := net.Dialer{Timeout: 15 * time.Second}
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("attachment host connection failed")
}
