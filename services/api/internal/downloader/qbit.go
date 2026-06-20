// Package downloader 提供下载管理能力
//
// 核心：qBittorrent WebUI API 客户端 + 自建调度层
// 设计：不自研 BT 协议（无底洞），只做调度（命名、入库、监控、RSS）
package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mediahub/api/pkg/logger"
)

// Client qBittorrent WebUI API 客户端
type Client struct {
	baseURL    string
	username   string
	password   string
	http       *http.Client
	mu         sync.Mutex // 保护登录状态
	loggedIn   bool
	loginAt    time.Time
}

// TorrentStatus 任务状态
type TorrentStatus string

const (
	StatusError      TorrentStatus = "error"       // 出错
	StatusMissingFiles TorrentStatus = "missingFiles" // 文件丢失
	StatusUploading   TorrentStatus = "uploading"   // 做种中
	StatusPaused     TorrentStatus = "paused"      // 暂停
	StatusQueued     TorrentStatus = "queued"      // 排队
	StatusStalled    TorrentStatus = "stalled"     // 下载停滞
	StatusChecking   TorrentStatus = "checking"    // 检查中
	StatusForced      TorrentStatus = "forcedUP"    // 强制做种
	StatusAllocating TorrentStatus = "allocating"  // 分配空间
	StatusDownloading TorrentStatus = "downloading" // 下载中
	StatusMetaDL     TorrentStatus = "metaDL"      // 元数据下载
	StatusCompleted  TorrentStatus = "completed"   // 完成
)

// Torrent qBittorrent 任务
type Torrent struct {
	Hash         string  `json:"hash"`
	Name         string  `json:"name"`
	Size         int64   `json:"size"`
	Progress     float64 `json:"progress"`
	DLSpeed      int64   `json:"dlspeed"`
	UPSpeed      int64   `json:"upspeed"`
	Priority     int     `json:"priority"`
	NumSeeds     int     `json:"num_seeds"`
	NumLeechs    int     `json:"num_leechs"`
	Ratio        float64 `json:"ratio"`
	State        string  `json:"state"`
	Category     string  `json:"category"`
	SavePath     string  `json:"save_path"`
	AddedOn      int64   `json:"added_on"`
	CompletionOn int64   `json:"completion_on"`
	ETA          int64   `json:"eta"`
}

// NewClient 构造 qBit 客户端
func NewClient(baseURL, username, password string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}
}

// login 登录获取 SID cookie
func (c *Client) login(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", c.username)
	form.Set("password", c.password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v2/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.baseURL)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("qbit login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return fmt.Errorf("qbit login: IP banned (too many failed attempts)")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qbit login failed: %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("Ok.")) && !bytes.Contains(body, []byte("Ok")) {
		return fmt.Errorf("qbit login: response not Ok: %s", string(body))
	}

	c.loggedIn = true
	c.loginAt = time.Now()
	logger.Info("qBittorrent 登录成功", "host", c.baseURL)
	return nil
}

// ensureLogin 确保已登录（每次请求前调用）
func (c *Client) ensureLogin(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 登录后 30 分钟过期，重新登录
	if c.loggedIn && time.Since(c.loginAt) < 30*time.Minute {
		return nil
	}
	return c.login(ctx)
}

// AddTorrentURL 通过 URL/magnet 添加任务
func (c *Client) AddTorrentURL(ctx context.Context, sourceURL, savePath, category string) (string, error) {
	if err := c.ensureLogin(ctx); err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("urls", sourceURL)
	if savePath != "" {
		form.Set("savepath", savePath)
	}
	if category != "" {
		form.Set("category", category)
	}
	form.Set("paused", "false")
	form.Set("upLimit", "0")
	form.Set("dlLimit", "0")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v2/torrents/add", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.baseURL)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("add torrent failed: %d: %s", resp.StatusCode, string(body))
	}

	// qBit 返回 "Ok." 或 torrent URL
	respStr := strings.TrimSpace(string(body))
	if respStr != "Ok." && !strings.HasPrefix(respStr, "Ok") {
		return "", fmt.Errorf("add torrent rejected: %s", respStr)
	}

	// 获取刚添加任务的 hash（用 ListTorrents 找最新的）
	torrents, err := c.ListTorrents(ctx, "")
	if err != nil {
		return "", err
	}
	// 最新的那个（按 added_on 倒序）
	var latest *Torrent
	for i := range torrents {
		if latest == nil || torrents[i].AddedOn > latest.AddedOn {
			latest = &torrents[i]
		}
	}
	if latest == nil {
		return "", fmt.Errorf("added torrent but not found in list")
	}
	return latest.Hash, nil
}

// AddTorrentFile 通过 .torrent 文件添加
func (c *Client) AddTorrentFile(ctx context.Context, filePath, savePath, category string) (string, error) {
	if err := c.ensureLogin(ctx); err != nil {
		return "", err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fw, err := w.CreateFormFile("torrents", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, file); err != nil {
		return "", err
	}

	_ = w.WriteField("savepath", savePath)
	_ = w.WriteField("category", category)
	_ = w.WriteField("paused", "false")
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v2/torrents/add", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Referer", c.baseURL)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("add torrent file failed: %d: %s", resp.StatusCode, string(body))
	}

	respStr := strings.TrimSpace(string(body))
	if respStr != "Ok." && !strings.HasPrefix(respStr, "Ok") {
		return "", fmt.Errorf("add torrent file rejected: %s", respStr)
	}
	return "", nil
}

// ListTorrents 列出任务
func (c *Client) ListTorrents(ctx context.Context, category string) ([]Torrent, error) {
	if err := c.ensureLogin(ctx); err != nil {
		return nil, err
	}

	q := url.Values{}
	if category != "" {
		q.Set("category", category)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/torrents/info?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", c.baseURL)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list torrents failed: %d: %s", resp.StatusCode, string(body))
	}

	var torrents []Torrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, fmt.Errorf("decode torrents: %w", err)
	}
	return torrents, nil
}

// RemoveTorrent 删除任务
func (c *Client) RemoveTorrent(ctx context.Context, hash string, deleteFiles bool) error {
	if err := c.ensureLogin(ctx); err != nil {
		return err
	}

	q := url.Values{}
	q.Set("hashes", hash)
	q.Set("deleteFiles", boolToStr(deleteFiles))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/torrents/delete?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Referer", c.baseURL)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove torrent failed: %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// PauseTorrent / ResumeTorrent
func (c *Client) PauseTorrent(ctx context.Context, hashes string) error {
	return c.simpleAction(ctx, "/api/v2/torrents/pause", hashes)
}
func (c *Client) ResumeTorrent(ctx context.Context, hashes string) error {
	return c.simpleAction(ctx, "/api/v2/torrents/resume", hashes)
}

func (c *Client) simpleAction(ctx context.Context, path, hashes string) error {
	if err := c.ensureLogin(ctx); err != nil {
		return err
	}
	q := url.Values{}
	q.Set("hashes", hashes)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Referer", c.baseURL)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("action failed: %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// Health 健康检查
func (c *Client) Health(ctx context.Context) error {
	if err := c.ensureLogin(ctx); err != nil {
		return err
	}
	return nil
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
