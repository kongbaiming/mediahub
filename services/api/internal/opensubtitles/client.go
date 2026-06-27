// Package opensubtitles 通过 OSHash 在 OpenSubtitles 识别影片（IMDb）。
package opensubtitles

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mediahub/api/internal/oshash"
	"github.com/mediahub/api/pkg/logger"
)

const (
	defaultRESTBase   = "https://api.opensubtitles.com/api/v1"
	defaultXMLRPCBase = "https://api.opensubtitles.org/xml-rpc"
	defaultUserAgent  = "MediaHub v0.5"
)

// Config 客户端配置
type Config struct {
	APIKey     string
	Username   string
	Password   string
	UserAgent  string
	RESTBase   string
	XMLRPCBase string
	TimeoutSec int
}

// Client OpenSubtitles 客户端
type Client struct {
	cfg    Config
	http   *http.Client
	token  string
	tokenMu sync.Mutex
}

// Match 影片识别结果
type Match struct {
	IMDBID   string
	Title    string
	Year     int
	Season   int
	Episode  int
	IsTV     bool
	Hash     string
	FileSize int64
}

// New 构造客户端；APIKey 为空时 Enabled() 为 false
func New(cfg Config) *Client {
	if cfg.UserAgent == "" {
		cfg.UserAgent = defaultUserAgent
	}
	if cfg.RESTBase == "" {
		cfg.RESTBase = defaultRESTBase
	}
	if cfg.XMLRPCBase == "" {
		cfg.XMLRPCBase = defaultXMLRPCBase
	}
	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &Client{
		cfg: cfg,
		http: &http.Client{Timeout: timeout},
	}
}

// Enabled 是否已配置（需 API Key）
func (c *Client) Enabled() bool {
	return c != nil && strings.TrimSpace(c.cfg.APIKey) != ""
}

// IdentifyFile 对视频文件计算 OSHash 并查询 OpenSubtitles
func (c *Client) IdentifyFile(ctx context.Context, path string) (*Match, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("OpenSubtitles 未配置")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	fh, err := oshash.ComputeFile(path)
	if err != nil {
		return nil, err
	}
	return c.IdentifyHash(ctx, fh.Hash, fh.FileSize)
}

// IdentifyHash 按 hash + 文件大小识别
func (c *Client) IdentifyHash(ctx context.Context, hash string, fileSize int64) (*Match, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("OpenSubtitles 未配置")
	}
	hash = strings.ToLower(strings.TrimSpace(hash))
	if hash == "" || fileSize <= 0 {
		return nil, fmt.Errorf("无效 hash 或文件大小")
	}

	// 优先 XML-RPC CheckMovieHash2（专用于识片，不依赖字幕是否存在）
	if c.cfg.Username != "" && c.cfg.Password != "" {
		if m, err := c.checkMovieHash2(ctx, hash); err == nil && m != nil {
			m.Hash = hash
			m.FileSize = fileSize
			return m, nil
		} else if err != nil {
			logger.Warn("OpenSubtitles CheckMovieHash2 失败，尝试 REST", "err", err)
		}
	}

	m, err := c.identifyREST(ctx, hash, fileSize)
	if err != nil {
		return nil, err
	}
	if m != nil {
		m.Hash = hash
		m.FileSize = fileSize
	}
	return m, nil
}

func (c *Client) identifyREST(ctx context.Context, hash string, fileSize int64) (*Match, error) {
	u := fmt.Sprintf("%s/subtitles?moviehash=%s&moviebytesize=%d&moviehash_match=only",
		strings.TrimRight(c.cfg.RESTBase, "/"), hash, fileSize)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req, "")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenSubtitles REST %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var parsed restSubtitleResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 {
		return nil, nil
	}
	fd := parsed.Data[0].Attributes.FeatureDetails
	if fd.IMDBID <= 0 {
		return nil, nil
	}
	m := &Match{
		IMDBID:  NormalizeIMDBID(strconv.Itoa(fd.IMDBID)),
		Title:   strings.TrimSpace(fd.Title),
		Year:    fd.Year,
		Season:  fd.SeasonNumber,
		Episode: fd.EpisodeNumber,
		IsTV:    fd.SeasonNumber > 0 || fd.EpisodeNumber > 0 || strings.EqualFold(fd.FeatureType, "episode"),
	}
	if m.Title == "" {
		m.Title = strings.TrimSpace(fd.MovieName)
	}
	return m, nil
}

type restSubtitleResponse struct {
	Data []struct {
		Attributes struct {
			FeatureDetails struct {
				IMDBID        int    `json:"imdb_id"`
				Title         string `json:"title"`
				MovieName     string `json:"movie_name"`
				Year          int    `json:"year"`
				SeasonNumber  int    `json:"season_number"`
				EpisodeNumber int    `json:"episode_number"`
				FeatureType   string `json:"feature_type"`
			} `json:"feature_details"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) checkMovieHash2(ctx context.Context, hash string) (*Match, error) {
	token, err := c.login(ctx)
	if err != nil {
		return nil, err
	}
	payload := fmt.Sprintf(`<?xml version="1.0"?>
<methodCall>
  <methodName>CheckMovieHash2</methodName>
  <params>
    <param><value><string>%s</string></value></param>
    <param><value><array><data><value><string>%s</string></value></data></array></value></param>
  </params>
</methodCall>`, xmlEscape(token), xmlEscape(hash))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.XMLRPCBase, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("XML-RPC HTTP %d", resp.StatusCode)
	}
	return parseCheckMovieHash2(body)
}

var (
	reIMDB   = regexp.MustCompile(`(?i)<name>MovieImdbID</name>\s*<value><(?:string|int)>([^<]+)</`)
	reName   = regexp.MustCompile(`(?i)<name>MovieName</name>\s*<value><string>([^<]*)</`)
	reYear   = regexp.MustCompile(`(?i)<name>MovieYear</name>\s*<value><(?:string|int)>([^<]+)</`)
	reSeason = regexp.MustCompile(`(?i)<name>SeriesSeason</name>\s*<value><(?:string|int)>([^<]+)</`)
	reEpisode = regexp.MustCompile(`(?i)<name>SeriesEpisode</name>\s*<value><(?:string|int)>([^<]+)</`)
	reKind   = regexp.MustCompile(`(?i)<name>MovieKind</name>\s*<value><string>([^<]*)</`)
)

func parseCheckMovieHash2(body []byte) (*Match, error) {
	s := string(body)
	if strings.Contains(s, "<fault>") {
		return nil, fmt.Errorf("XML-RPC fault")
	}
	imdbM := reIMDB.FindStringSubmatch(s)
	if len(imdbM) < 2 || strings.TrimSpace(imdbM[1]) == "" || imdbM[1] == "0" {
		return nil, nil
	}
	m := &Match{IMDBID: NormalizeIMDBID(imdbM[1])}
	if nameM := reName.FindStringSubmatch(s); len(nameM) > 1 {
		m.Title = strings.TrimSpace(nameM[1])
	}
	if yearM := reYear.FindStringSubmatch(s); len(yearM) > 1 {
		m.Year, _ = strconv.Atoi(strings.TrimSpace(yearM[1]))
	}
	if seasonM := reSeason.FindStringSubmatch(s); len(seasonM) > 1 {
		m.Season, _ = strconv.Atoi(strings.TrimSpace(seasonM[1]))
	}
	if epM := reEpisode.FindStringSubmatch(s); len(epM) > 1 {
		m.Episode, _ = strconv.Atoi(strings.TrimSpace(epM[1]))
	}
	if kindM := reKind.FindStringSubmatch(s); len(kindM) > 1 {
		kind := strings.ToLower(strings.TrimSpace(kindM[1]))
		m.IsTV = kind == "episode" || kind == "tvseries" || m.Season > 0 || m.Episode > 0
	}
	return m, nil
}

func (c *Client) login(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.token != "" {
		return c.token, nil
	}

	payload := fmt.Sprintf(`<?xml version="1.0"?>
<methodCall>
  <methodName>LogIn</methodName>
  <params>
    <param><value><string>%s</string></value></param>
    <param><value><string>%s</string></value></param>
    <param><value><string>eng</string></value></param>
    <param><value><string>%s</string></value></param>
  </params>
</methodCall>`,
		xmlEscape(c.cfg.Username),
		xmlEscape(c.cfg.Password),
		xmlEscape(c.cfg.UserAgent),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.XMLRPCBase, strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type loginResp struct {
		Params struct {
			Param struct {
				Value struct {
					Struct struct {
						Member []struct {
							Name  string `xml:"name"`
							Value struct {
								String string `xml:"string"`
							} `xml:"value"`
						} `xml:"member"`
					} `xml:"struct"`
				} `xml:"value"`
			} `xml:"param"`
		} `xml:"params"`
	}
	var lr loginResp
	if err := xml.NewDecoder(bytes.NewReader(body)).Decode(&lr); err != nil {
		return "", fmt.Errorf("解析 LogIn 响应: %w", err)
	}
	for _, m := range lr.Params.Param.Value.Struct.Member {
		if strings.EqualFold(m.Name, "token") && m.Value.String != "" {
			c.token = m.Value.String
			return c.token, nil
		}
	}
	return "", fmt.Errorf("OpenSubtitles 登录失败")
}

func (c *Client) setHeaders(req *http.Request, bearer string) {
	req.Header.Set("Api-Key", c.cfg.APIKey)
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
}

// NormalizeIMDBID 规范为 tt1234567
func NormalizeIMDBID(id string) string {
	id = strings.TrimSpace(strings.ToLower(id))
	if id == "" {
		return ""
	}
	if strings.HasPrefix(id, "tt") {
		return id
	}
	if n, err := strconv.Atoi(id); err == nil && n > 0 {
		return fmt.Sprintf("tt%07d", n)
	}
	return id
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
