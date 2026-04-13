// Package cookie manages Netease Cloud Music cookie file reading and writing.
package cookie

import (
	"os"
	"strings"
	"sync"
)

// Manager handles reading and parsing cookie.txt.
type Manager struct {
	path string
	mu   sync.RWMutex
}

// NewManager creates a cookie manager for a cookie file path.
func NewManager(path string) *Manager {
	return &Manager{path: path}
}

// Path returns the cookie file path.
func (m *Manager) Path() string {
	return m.path
}

// ReadRaw reads raw cookie content from file.
func (m *Manager) ReadRaw() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	content, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// ParseFromFile parses cookies from cookie file into key/value map.
func (m *Manager) ParseFromFile() (map[string]string, error) {
	raw, err := m.ReadRaw()
	if err != nil {
		return map[string]string{}, err
	}
	return ParseCookieString(raw), nil
}

// IsValid performs a basic compatibility check for cookie health reporting.
func (m *Manager) IsValid(cookies map[string]string) bool {
	if len(cookies) == 0 {
		return false
	}
	musicU := strings.TrimSpace(cookies["MUSIC_U"])
	if len(musicU) < 10 {
		return false
	}

	important := []string{"MUSIC_A", "__csrf", "NMTID", "WEVNSM", "WNMCID"}
	present := 0
	for _, k := range important {
		if strings.TrimSpace(cookies[k]) != "" {
			present++
		}
	}

	// Keep this check practical: MUSIC_U must exist, plus at least one extra marker.
	return present >= 1
}

// Write persists a cookie map to the cookie file, merging with existing cookies.
func (m *Manager) Write(newCookies map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, _ := m.readRawLocked()
	merged := ParseCookieString(existing)
	for k, v := range newCookies {
		if k != "" && v != "" {
			merged[k] = v
		}
	}

	parts := make([]string, 0, len(merged))
	for k, v := range merged {
		parts = append(parts, k+"="+v)
	}
	content := strings.Join(parts, "; ")
	return os.WriteFile(m.path, []byte(content), 0o644)
}

// readRawLocked reads raw content without acquiring the lock (caller must hold it).
func (m *Manager) readRawLocked() (string, error) {
	content, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// ParseCookieString parses cookie string formats split by ';' or newlines.
func ParseCookieString(cookieString string) map[string]string {
	result := make(map[string]string)
	cookieString = strings.TrimSpace(cookieString)
	if cookieString == "" {
		return result
	}

	var parts []string
	if strings.Contains(cookieString, ";") {
		parts = strings.Split(cookieString, ";")
	} else if strings.Contains(cookieString, "\n") {
		parts = strings.Split(cookieString, "\n")
	} else {
		parts = []string{cookieString}
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || !strings.Contains(part, "=") {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if key != "" && val != "" {
			result[key] = val
		}
	}
	return result
}

