// Package ipgeo resolves IP addresses to geographic locations.
package ipgeo

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GeoResult holds the resolved province and city for an IP address.
type GeoResult struct {
	Province string
	City     string
}

// Resolver provides IP-to-geolocation lookups with an in-memory cache.
type Resolver struct {
	client *http.Client
	cache  sync.Map
}

// NewResolver creates a geolocation resolver.
func NewResolver() *Resolver {
	return &Resolver{
		client: &http.Client{Timeout: 3 * time.Second},
	}
}

// Resolve returns province and city for an IP address.
// Uses a free API with in-memory caching to avoid repeated lookups.
func (r *Resolver) Resolve(ip string) GeoResult {
	if ip == "" || isPrivateIP(ip) {
		return GeoResult{Province: "本地网络", City: "本地"}
	}

	if cached, ok := r.cache.Load(ip); ok {
		return cached.(GeoResult)
	}

	result := r.lookupIP(ip)
	r.cache.Store(ip, result)
	return result
}

func (r *Resolver) lookupIP(ip string) GeoResult {
	// Try multiple free IP geolocation APIs
	if result, ok := r.tryIPAPI(ip); ok {
		return result
	}
	return GeoResult{Province: "未知", City: "未知"}
}

type ipAPIResponse struct {
	Status     string `json:"status"`
	Country    string `json:"country"`
	RegionName string `json:"regionName"`
	City       string `json:"city"`
}

func (r *Resolver) tryIPAPI(ip string) (GeoResult, bool) {
	resp, err := r.client.Get(fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN&fields=status,country,regionName,city", ip))
	if err != nil {
		return GeoResult{}, false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return GeoResult{}, false
	}

	var data ipAPIResponse
	if err := json.Unmarshal(body, &data); err != nil || data.Status != "success" {
		return GeoResult{}, false
	}

	province := data.RegionName
	if province == "" {
		province = data.Country
	}
	city := data.City
	if city == "" {
		city = province
	}
	return GeoResult{Province: province, City: city}, true
}

// ExtractIP extracts the real client IP from an HTTP request,
// checking X-Forwarded-For, X-Real-IP, and RemoteAddr in order.
func ExtractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	privateRanges := []struct{ start, end net.IP }{
		{net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")},
		{net.ParseIP("172.16.0.0"), net.ParseIP("172.31.255.255")},
		{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.255.255")},
		{net.ParseIP("127.0.0.0"), net.ParseIP("127.255.255.255")},
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return ip.IsLoopback() || ip.IsLinkLocalUnicast()
	}
	for _, r := range privateRanges {
		if bytesInRange(ip4, r.start.To4(), r.end.To4()) {
			return true
		}
	}
	return false
}

func bytesInRange(ip, start, end net.IP) bool {
	for i := 0; i < len(ip); i++ {
		if ip[i] < start[i] {
			return false
		}
		if ip[i] > end[i] {
			return false
		}
	}
	return true
}
