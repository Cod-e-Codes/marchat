package server

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

var (
	allowedWSOrigins     []string
	allowedWSOriginsOnce sync.Once

	trustedProxyNets   []*net.IPNet
	trustedProxyIPs    map[string]struct{}
	trustedProxiesOnce sync.Once
)

func loadAllowedWSOrigins() {
	allowedWSOriginsOnce.Do(func() {
		raw := strings.TrimSpace(os.Getenv("MARCHAT_ALLOWED_ORIGINS"))
		if raw == "" {
			return
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				allowedWSOrigins = append(allowedWSOrigins, part)
			}
		}
	})
}

func loadTrustedProxies() {
	trustedProxiesOnce.Do(func() {
		trustedProxyIPs = make(map[string]struct{})
		raw := strings.TrimSpace(os.Getenv("MARCHAT_TRUSTED_PROXIES"))
		if raw == "" {
			return
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if strings.Contains(part, "/") {
				_, network, err := net.ParseCIDR(part)
				if err != nil {
					log.Printf("Ignoring invalid MARCHAT_TRUSTED_PROXIES CIDR %q: %v", part, err)
					continue
				}
				trustedProxyNets = append(trustedProxyNets, network)
				continue
			}
			ip := net.ParseIP(part)
			if ip == nil {
				log.Printf("Ignoring invalid MARCHAT_TRUSTED_PROXIES entry %q", part)
				continue
			}
			trustedProxyIPs[ip.String()] = struct{}{}
		}
	})
}

func resetRequestContextForTests() {
	allowedWSOriginsOnce = sync.Once{}
	allowedWSOrigins = nil
	trustedProxiesOnce = sync.Once{}
	trustedProxyNets = nil
	trustedProxyIPs = nil
}

func requestPeerIP(r *http.Request) string {
	if r.RemoteAddr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func isTrustedProxyIP(ipStr string) bool {
	loadTrustedProxies()
	if ipStr == "" {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	if _, ok := trustedProxyIPs[ip.String()]; ok {
		return true
	}
	for _, network := range trustedProxyNets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func forwardedClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if comma := strings.Index(xff, ","); comma != -1 {
			return strings.TrimSpace(xff[:comma])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	return ""
}

// getClientIP returns the client IP for logging and connection metadata.
// Forwarded headers are honored only when the immediate remote address is a
// trusted reverse proxy (MARCHAT_TRUSTED_PROXIES).
func getClientIP(r *http.Request) string {
	peer := requestPeerIP(r)
	if peer != "" && isTrustedProxyIP(peer) {
		if client := forwardedClientIP(r); client != "" {
			return client
		}
	}
	if peer != "" {
		return peer
	}
	return "unknown"
}

func hostnameOnly(hostport string) string {
	hostport = strings.TrimSpace(hostport)
	if hostport == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return strings.Trim(strings.TrimSpace(hostport), "[]")
	}
	return strings.Trim(host, "[]")
}

func portOnly(hostport string) string {
	_, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return ""
	}
	return port
}

func isLoopbackHostname(host string) bool {
	switch strings.ToLower(hostnameOnly(host)) {
	case "localhost", "127.0.0.1", "::1", "0:0:0:0:0:0:0:1":
		return true
	default:
		return false
	}
}

func checkWebSocketOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	originURL, err := url.Parse(origin)
	if err != nil || originURL.Scheme == "" || originURL.Host == "" {
		log.Printf("WebSocket origin rejected: invalid origin %q", origin)
		return false
	}

	reqHost := r.Host
	if reqHost == "" {
		reqHost = r.Header.Get("Host")
	}
	originHost := hostnameOnly(originURL.Host)
	reqHostname := hostnameOnly(reqHost)

	if originHost != "" && reqHostname != "" && strings.EqualFold(originHost, reqHostname) {
		return true
	}
	if isLoopbackHostname(originHost) && isLoopbackHostname(reqHostname) {
		return true
	}

	loadAllowedWSOrigins()
	for _, entry := range allowedWSOrigins {
		if webSocketOriginMatchesAllowlist(originURL, entry) {
			return true
		}
	}

	log.Printf("WebSocket origin rejected: %s (host: %s)", origin, reqHost)
	return false
}

func webSocketOriginMatchesAllowlist(originURL *url.URL, entry string) bool {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return false
	}
	if strings.Contains(entry, "://") {
		allowed, err := url.Parse(entry)
		if err != nil || allowed.Host == "" {
			return false
		}
		if !strings.EqualFold(originURL.Scheme, allowed.Scheme) {
			return false
		}
		return strings.EqualFold(hostnameOnly(originURL.Host), hostnameOnly(allowed.Host)) &&
			portOnly(originURL.Host) == portOnly(allowed.Host)
	}
	entryHost := hostnameOnly(entry)
	if entryHost == "" || !strings.EqualFold(hostnameOnly(originURL.Host), entryHost) {
		return false
	}
	entryPort := portOnly(entry)
	if entryPort == "" {
		return true
	}
	return portOnly(originURL.Host) == entryPort
}
