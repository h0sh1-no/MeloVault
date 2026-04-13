// Package netease implements the Netease Cloud Music API client.
package netease

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/h0sh1-no/MeloVault/internal/cache"
)

const (
	aesKey = "e82ckenh8dichen8"

	userAgent = "NeteaseMusic/9.1.65.240927161425(9001065);Dalvik/2.1.0 (Linux; U; Android 14; Pixel 7 Build/UQ1A.240205.004)"
	referer   = "https://music.163.com/"

	songURLV1API      = "https://interface3.music.163.com/eapi/song/enhance/player/url/v1"
	songDetailV3API   = "https://interface3.music.163.com/api/v3/song/detail"
	lyricAPI          = "https://interface3.music.163.com/api/song/lyric"
	searchAPI         = "https://music.163.com/api/cloudsearch/pc"
	playlistDetailAPI = "https://music.163.com/api/v6/playlist/detail"
	albumDetailAPI    = "https://music.163.com/api/v1/album/"
)

var (
	defaultRequestCookies = map[string]string{
		"os":       "android",
		"appver":   "9.1.65",
		"osver":    "14",
		"deviceId": "pyncm!",
		"channel":  "google",
	}

	randMu sync.Mutex
	rng    = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// Client wraps all upstream Netease API calls.
type Client struct {
	httpClient *http.Client
	cache      *cache.TTLCache
	realIP     string // optional X-Real-IP to spoof region (e.g. Taiwan/HK)
}

// NewClient creates an optimized reusable client with connection pooling.
// If NETEASE_REAL_IP is set, all requests include X-Real-IP to spoof region.
func NewClient() *Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   6 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   64,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		cache:  cache.New(),
		realIP: os.Getenv("NETEASE_REAL_IP"),
	}
}

// SetRealIP updates the X-Real-IP used for all subsequent requests.
func (c *Client) SetRealIP(ip string) {
	c.realIP = ip
}

// setRealIPHeaders injects X-Real-IP and X-Forwarded-For if configured.
func (c *Client) setRealIPHeaders(req *http.Request) {
	if c.realIP != "" {
		req.Header.Set("X-Real-IP", c.realIP)
		req.Header.Set("X-Forwarded-For", c.realIP)
	}
}

// HTTPClient exposes the underlying shared client (used by file download flow).
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// GetSongURL fetches song playable URL with quality level.
func (c *Client) GetSongURL(ctx context.Context, songID int64, quality string, cookies map[string]string) (map[string]any, error) {
	cacheKey := fmt.Sprintf("song_url:%d:%s", songID, quality)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if m, ok := cached.(map[string]any); ok {
			return cloneMap(m), nil
		}
	}

	headerJSON, err := buildHeaderJSON()
	if err != nil {
		return nil, err
	}

	payload := struct {
		IDs         []int64 `json:"ids"`
		Level       string  `json:"level"`
		EncodeType  string  `json:"encodeType"`
		Header      string  `json:"header"`
		ImmerseType string  `json:"immerseType,omitempty"`
	}{
		IDs:        []int64{songID},
		Level:      quality,
		EncodeType: "flac",
		Header:     headerJSON,
	}
	if quality == "sky" {
		payload.ImmerseType = "c51"
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal song url payload failed: %w", err)
	}

	params, err := encryptParams(songURLV1API, string(payloadJSON))
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("params", params)

	body, err := c.postForm(ctx, songURLV1API, form, cookies)
	if err != nil {
		return nil, err
	}

	result, err := decodeJSONMap(body)
	if err != nil {
		return nil, err
	}
	if readCode(result) != 200 {
		return nil, fmt.Errorf("get song url failed: %s", asString(result["message"]))
	}

	c.cache.Set(cacheKey, result, 20*time.Second)
	return cloneMap(result), nil
}

// GetSongDetail fetches detailed song metadata.
func (c *Client) GetSongDetail(ctx context.Context, songID int64) (map[string]any, error) {
	cacheKey := fmt.Sprintf("song_detail:%d", songID)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if m, ok := cached.(map[string]any); ok {
			return cloneMap(m), nil
		}
	}

	payload := []map[string]any{
		{
			"id": songID,
			"v":  0,
		},
	}
	cJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("c", string(cJSON))

	body, err := c.postForm(ctx, songDetailV3API, form, nil)
	if err != nil {
		return nil, err
	}

	result, err := decodeJSONMap(body)
	if err != nil {
		return nil, err
	}
	if readCode(result) != 200 {
		return nil, fmt.Errorf("get song detail failed: %s", asString(result["message"]))
	}

	c.cache.Set(cacheKey, result, 10*time.Minute)
	return cloneMap(result), nil
}

// GetLyric fetches lyric data by song id.
func (c *Client) GetLyric(ctx context.Context, songID int64, cookies map[string]string) (map[string]any, error) {
	cacheKey := fmt.Sprintf("song_lyric:%d", songID)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if m, ok := cached.(map[string]any); ok {
			return cloneMap(m), nil
		}
	}

	form := url.Values{}
	form.Set("id", strconv.FormatInt(songID, 10))
	form.Set("cp", "false")
	form.Set("tv", "0")
	form.Set("lv", "0")
	form.Set("rv", "0")
	form.Set("kv", "0")
	form.Set("yv", "0")
	form.Set("ytv", "0")
	form.Set("yrv", "0")

	body, err := c.postForm(ctx, lyricAPI, form, cookies)
	if err != nil {
		return nil, err
	}

	result, err := decodeJSONMap(body)
	if err != nil {
		return nil, err
	}
	if readCode(result) != 200 {
		return nil, fmt.Errorf("get lyric failed: %s", asString(result["message"]))
	}

	c.cache.Set(cacheKey, result, 10*time.Minute)
	return cloneMap(result), nil
}

// SearchMusic searches songs and returns normalized list data.
func (c *Client) SearchMusic(ctx context.Context, keywords string, cookies map[string]string, limit int) ([]map[string]any, error) {
	cacheKey := fmt.Sprintf("search:%s:%d", strings.ToLower(strings.TrimSpace(keywords)), limit)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if list, ok := cached.([]map[string]any); ok {
			return cloneMapSlice(list), nil
		}
	}

	form := url.Values{}
	form.Set("s", keywords)
	form.Set("type", "1")
	form.Set("limit", strconv.Itoa(limit))

	body, err := c.postForm(ctx, searchAPI, form, cookies)
	if err != nil {
		return nil, err
	}

	result, err := decodeJSONMap(body)
	if err != nil {
		return nil, err
	}
	if readCode(result) != 200 {
		return nil, fmt.Errorf("search failed: %s", asString(result["message"]))
	}

	resultBlock := asMap(result["result"])
	songsRaw := asSlice(resultBlock["songs"])
	songs := make([]map[string]any, 0, len(songsRaw))
	for _, raw := range songsRaw {
		item := asMap(raw)
		al := asMap(item["al"])
		songs = append(songs, map[string]any{
			"id":      asInt64(item["id"]),
			"name":    asString(item["name"]),
			"artists": strings.Join(extractArtistNames(item["ar"]), "/"),
			"album":   asString(al["name"]),
			"picUrl":  asString(al["picUrl"]),
		})
	}

	c.cache.Set(cacheKey, songs, 30*time.Second)
	return cloneMapSlice(songs), nil
}

// GetPlaylistDetail fetches playlist info and all tracks (in concurrent batches).
func (c *Client) GetPlaylistDetail(ctx context.Context, playlistID int64, cookies map[string]string) (map[string]any, error) {
	cacheKey := fmt.Sprintf("playlist:%d", playlistID)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if m, ok := cached.(map[string]any); ok {
			return cloneMap(m), nil
		}
	}

	form := url.Values{}
	form.Set("id", strconv.FormatInt(playlistID, 10))

	body, err := c.postForm(ctx, playlistDetailAPI, form, cookies)
	if err != nil {
		return nil, err
	}

	result, err := decodeJSONMap(body)
	if err != nil {
		return nil, err
	}
	if readCode(result) != 200 {
		return nil, fmt.Errorf("get playlist detail failed: %s", asString(result["message"]))
	}

	playlist := asMap(result["playlist"])
	info := map[string]any{
		"id":          asInt64(playlist["id"]),
		"name":        asString(playlist["name"]),
		"coverImgUrl": asString(playlist["coverImgUrl"]),
		"creator":     asString(asMap(playlist["creator"])["nickname"]),
		"trackCount":  asInt64(playlist["trackCount"]),
		"playCount":   asInt64(playlist["playCount"]),
		"description": asString(playlist["description"]),
		"tracks":      []map[string]any{},
	}

	trackIDs := make([]int64, 0, 256)
	for _, trackRaw := range asSlice(playlist["trackIds"]) {
		tm := asMap(trackRaw)
		id := asInt64(tm["id"])
		if id > 0 {
			trackIDs = append(trackIDs, id)
		}
	}

	if len(trackIDs) > 0 {
		chunks := chunkIDs(trackIDs, 100)
		type batchResult struct {
			index  int
			tracks []map[string]any
			err    error
		}

		results := make([][]map[string]any, len(chunks))
		batchCh := make(chan batchResult, len(chunks))
		sem := make(chan struct{}, 4)
		var wg sync.WaitGroup

		for i, ids := range chunks {
			wg.Add(1)
			go func(index int, idsChunk []int64) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				tracks, ferr := c.fetchSongBatch(ctx, idsChunk, cookies)
				batchCh <- batchResult{
					index:  index,
					tracks: tracks,
					err:    ferr,
				}
			}(i, ids)
		}

		wg.Wait()
		close(batchCh)

		var firstErr error
		for item := range batchCh {
			if item.err != nil && firstErr == nil {
				firstErr = item.err
			}
			results[item.index] = item.tracks
		}
		if firstErr != nil {
			return nil, firstErr
		}

		merged := make([]map[string]any, 0, len(trackIDs))
		for _, batchTracks := range results {
			merged = append(merged, batchTracks...)
		}
		info["tracks"] = merged
	}

	c.cache.Set(cacheKey, info, 2*time.Minute)
	return cloneMap(info), nil
}

// GetAlbumDetail fetches album metadata and song list.
func (c *Client) GetAlbumDetail(ctx context.Context, albumID int64, cookies map[string]string) (map[string]any, error) {
	cacheKey := fmt.Sprintf("album:%d", albumID)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if m, ok := cached.(map[string]any); ok {
			return cloneMap(m), nil
		}
	}

	fullURL := albumDetailAPI + strconv.FormatInt(albumID, 10)
	body, err := c.get(ctx, fullURL, cookies)
	if err != nil {
		return nil, err
	}

	result, err := decodeJSONMap(body)
	if err != nil {
		return nil, err
	}
	if readCode(result) != 200 {
		return nil, fmt.Errorf("get album detail failed: %s", asString(result["message"]))
	}

	album := asMap(result["album"])
	info := map[string]any{
		"id":          asInt64(album["id"]),
		"name":        asString(album["name"]),
		"coverImgUrl": GetPicURL(asInt64(album["pic"]), 300),
		"artist":      asString(asMap(album["artist"])["name"]),
		"publishTime": asInt64(album["publishTime"]),
		"description": asString(album["description"]),
		"songs":       []map[string]any{},
	}

	songsRaw := asSlice(result["songs"])
	songs := make([]map[string]any, 0, len(songsRaw))
	for _, raw := range songsRaw {
		item := asMap(raw)
		al := asMap(item["al"])
		songs = append(songs, map[string]any{
			"id":      asInt64(item["id"]),
			"name":    asString(item["name"]),
			"artists": strings.Join(extractArtistNames(item["ar"]), "/"),
			"album":   asString(al["name"]),
			"picUrl":  GetPicURL(asInt64(al["pic"]), 300),
		})
	}
	info["songs"] = songs

	c.cache.Set(cacheKey, info, 2*time.Minute)
	return cloneMap(info), nil
}

// ── Netease QR code login ─────────────────────────────────────────────────────

const (
	qrUnikeyAPI    = "https://interface3.music.163.com/eapi/login/qrcode/unikey"
	qrCheckAPI     = "https://interface3.music.163.com/eapi/login/qrcode/client/login"
	qrLoginBaseURL = "https://music.163.com/login?codekey="
)

// QRStatus values returned by Netease.
const (
	QRExpired    = 800
	QRWaiting    = 801
	QRScanned    = 802
	QRAuthorized = 803
)

// GetLoginQRKey generates a new Netease QR login key.
// Returns the unikey and the URL to encode as QR content.
func (c *Client) GetLoginQRKey(ctx context.Context) (key, loginURL string, err error) {
	headerJSON, err := buildHeaderJSON()
	if err != nil {
		return "", "", err
	}

	payload := struct {
		Type   int    `json:"type"`
		Header string `json:"header"`
	}{
		Type:   1,
		Header: headerJSON,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("marshal qr key payload failed: %w", err)
	}

	params, err := encryptParams(qrUnikeyAPI, string(payloadJSON))
	if err != nil {
		return "", "", err
	}

	form := url.Values{"params": {params}}
	body, err := c.postForm(ctx, qrUnikeyAPI, form, nil)
	if err != nil {
		return "", "", fmt.Errorf("get qr key: %w", err)
	}
	result, err := decodeJSONMap(body)
	if err != nil {
		return "", "", err
	}
	if readCode(result) != 200 {
		return "", "", fmt.Errorf("get qr key: unexpected code %d", readCode(result))
	}
	key = asString(result["unikey"])
	if key == "" {
		return "", "", fmt.Errorf("empty unikey in response")
	}
	return key, qrLoginBaseURL + key, nil
}

// CheckLoginQRStatus polls the QR scan status.
// Returns the status code (800-803) and, when code==803, a cookie string.
func (c *Client) CheckLoginQRStatus(ctx context.Context, key string) (code int, cookieStr string, err error) {
	headerJSON, err := buildHeaderJSON()
	if err != nil {
		return 0, "", err
	}

	payload := struct {
		Key    string `json:"key"`
		Type   int    `json:"type"`
		Header string `json:"header"`
	}{
		Key:    key,
		Type:   1,
		Header: headerJSON,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, "", fmt.Errorf("marshal qr check payload failed: %w", err)
	}

	params, err := encryptParams(qrCheckAPI, string(payloadJSON))
	if err != nil {
		return 0, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qrCheckAPI, strings.NewReader(url.Values{"params": {params}}.Encode()))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", referer)
	c.setRealIPHeaders(req)
	attachCookies(req, mergeCookies(nil))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("check qr status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}

	result, err := decodeJSONMap(body)
	if err != nil {
		return 0, "", err
	}

	code = int(readCode(result))
	if code == QRAuthorized {
		// Prefer parsed cookies, then complement with raw Set-Cookie headers.
		collected := make(map[string]string, len(resp.Cookies()))
		for _, c := range resp.Cookies() {
			if c.Name == "" || c.Value == "" {
				continue
			}
			collected[c.Name] = c.Value
		}
		mergeCookiePairs(collected, parseRawSetCookieHeaders(resp.Header.Values("Set-Cookie")))

		// Some upstream/proxy implementations return cookie in JSON body.
		if len(collected) == 0 {
			mergeCookiePairs(collected, parseLooseCookieString(asString(result["cookie"])))
		}
		if _, ok := collected["os"]; !ok {
			collected["os"] = "android"
		}
		if _, ok := collected["appver"]; !ok {
			collected["appver"] = "9.1.65"
		}

		cookieStr = flattenCookieMap(collected)
	}
	return code, cookieStr, nil
}

// ─────────────────────────────────────────────────────────────────────────────

func (c *Client) fetchSongBatch(ctx context.Context, ids []int64, cookies map[string]string) ([]map[string]any, error) {
	if len(ids) == 0 {
		return []map[string]any{}, nil
	}

	payload := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		payload = append(payload, map[string]any{
			"id": id,
			"v":  0,
		})
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("c", string(payloadJSON))

	body, err := c.postForm(ctx, songDetailV3API, form, cookies)
	if err != nil {
		return nil, err
	}
	result, err := decodeJSONMap(body)
	if err != nil {
		return nil, err
	}
	if readCode(result) != 200 {
		return nil, fmt.Errorf("batch song detail failed: %s", asString(result["message"]))
	}

	songItems := asSlice(result["songs"])
	tracks := make([]map[string]any, 0, len(songItems))
	for _, raw := range songItems {
		song := asMap(raw)
		al := asMap(song["al"])
		tracks = append(tracks, map[string]any{
			"id":      asInt64(song["id"]),
			"name":    asString(song["name"]),
			"artists": strings.Join(extractArtistNames(song["ar"]), "/"),
			"album":   asString(al["name"]),
			"picUrl":  asString(al["picUrl"]),
		})
	}
	return tracks, nil
}

func (c *Client) postForm(ctx context.Context, endpoint string, form url.Values, cookies map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", referer)
	c.setRealIPHeaders(req)
	attachCookies(req, mergeCookies(cookies))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("post %s failed: status=%d body=%s", endpoint, resp.StatusCode, truncate(string(body), 256))
	}
	return body, nil
}

func (c *Client) get(ctx context.Context, endpoint string, cookies map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", referer)
	c.setRealIPHeaders(req)
	attachCookies(req, mergeCookies(cookies))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get %s failed: status=%d body=%s", endpoint, resp.StatusCode, truncate(string(body), 256))
	}
	return body, nil
}

func buildHeaderJSON() (string, error) {
	header := struct {
		OS        string `json:"os"`
		AppVer    string `json:"appver"`
		OSVer     string `json:"osver"`
		DeviceID  string `json:"deviceId"`
		RequestID string `json:"requestId"`
	}{
		OS:        "android",
		AppVer:    "9.1.65",
		OSVer:     "14",
		DeviceID:  "pyncm!",
		RequestID: randomRequestID(),
	}

	raw, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func randomRequestID() string {
	randMu.Lock()
	defer randMu.Unlock()
	return strconv.Itoa(rng.Intn(10000000) + 20000000)
}

func encryptParams(apiURL, payloadJSON string) (string, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return "", fmt.Errorf("parse api url failed: %w", err)
	}
	urlPath := strings.ReplaceAll(parsed.Path, "/eapi/", "/api/")
	digest := md5Hex("nobody" + urlPath + "use" + payloadJSON + "md5forencrypt")
	plain := urlPath + "-36cd479b6b5-" + payloadJSON + "-36cd479b6b5-" + digest

	block, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		return "", err
	}

	padded := pkcs7Pad([]byte(plain), block.BlockSize())
	encrypted := make([]byte, len(padded))
	for i := 0; i < len(padded); i += block.BlockSize() {
		block.Encrypt(encrypted[i:i+block.BlockSize()], padded[i:i+block.BlockSize()])
	}
	return hex.EncodeToString(encrypted), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	if padLen == 0 {
		padLen = blockSize
	}
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(data, padding...)
}

func md5Hex(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// GetPicURL builds Netease encrypted picture URL.
func GetPicURL(picID int64, size int) string {
	if picID <= 0 {
		return ""
	}
	encID := neteaseEncryptID(strconv.FormatInt(picID, 10))
	return fmt.Sprintf("https://p3.music.126.net/%s/%d.jpg?param=%dy%d", encID, picID, size, size)
}

func neteaseEncryptID(id string) string {
	magic := []byte("3go8&$8*3*3h0k(2)2")
	plain := []byte(id)
	for i := range plain {
		plain[i] = plain[i] ^ magic[i%len(magic)]
	}
	sum := md5.Sum(plain)
	enc := base64.StdEncoding.EncodeToString(sum[:])
	enc = strings.ReplaceAll(enc, "/", "_")
	enc = strings.ReplaceAll(enc, "+", "-")
	return enc
}

func decodeJSONMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty response body")
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode json failed: %w", err)
	}
	return out, nil
}

func readCode(m map[string]any) int64 {
	return asInt64(m["code"])
}

func mergeCookies(user map[string]string) map[string]string {
	merged := make(map[string]string, len(defaultRequestCookies)+len(user))
	for k, v := range defaultRequestCookies {
		merged[k] = v
	}
	for k, v := range user {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		merged[k] = v
	}
	// Always enforce Android identity to unlock higher audio quality tiers.
	merged["os"] = defaultRequestCookies["os"]
	merged["appver"] = defaultRequestCookies["appver"]
	return merged
}

func attachCookies(req *http.Request, cookies map[string]string) {
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  k,
			Value: v,
			Path:  "/",
		})
	}
}

func mergeCookiePairs(dst map[string]string, src map[string]string) {
	for k, v := range src {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		dst[k] = v
	}
}

func parseRawSetCookieHeaders(headers []string) map[string]string {
	result := make(map[string]string, len(headers))
	for _, raw := range headers {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		firstPair := raw
		if i := strings.Index(firstPair, ";"); i >= 0 {
			firstPair = firstPair[:i]
		}
		kv := strings.SplitN(firstPair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if key == "" || val == "" {
			continue
		}
		result[key] = val
	}
	return result
}

func parseLooseCookieString(cookieString string) map[string]string {
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
		if key == "" || val == "" {
			continue
		}
		result[key] = val
	}
	return result
}

func flattenCookieMap(cookies map[string]string) string {
	if len(cookies) == 0 {
		return ""
	}
	parts := make([]string, 0, len(cookies))
	for k, v := range cookies {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}

func chunkIDs(ids []int64, size int) [][]int64 {
	if size <= 0 || len(ids) == 0 {
		return nil
	}
	out := make([][]int64, 0, (len(ids)+size-1)/size)
	for i := 0; i < len(ids); i += size {
		end := i + size
		if end > len(ids) {
			end = len(ids)
		}
		chunk := make([]int64, end-i)
		copy(chunk, ids[i:end])
		out = append(out, chunk)
	}
	return out
}

func extractArtistNames(raw any) []string {
	artistsRaw := asSlice(raw)
	artists := make([]string, 0, len(artistsRaw))
	for _, item := range artistsRaw {
		name := asString(asMap(item)["name"])
		if name != "" {
			artists = append(artists, name)
		}
	}
	return artists
}

func asMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return m
}

func asSlice(v any) []any {
	if v == nil {
		return []any{}
	}
	s, ok := v.([]any)
	if !ok {
		return []any{}
	}
	return s
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case float32:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case uint32:
		return strconv.FormatUint(uint64(t), 10)
	default:
		return ""
	}
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case int32:
		return int64(t)
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	case json.Number:
		n, err := t.Int64()
		if err == nil {
			return n
		}
		f, ferr := t.Float64()
		if ferr == nil {
			return int64(f)
		}
		return 0
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err == nil {
			return n
		}
		return 0
	default:
		return 0
	}
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func cloneMap(in map[string]any) map[string]any {
	raw, err := json.Marshal(in)
	if err != nil {
		return in
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return in
	}
	return out
}

func cloneMapSlice(in []map[string]any) []map[string]any {
	raw, err := json.Marshal(in)
	if err != nil {
		return in
	}
	out := []map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return in
	}
	return out
}
