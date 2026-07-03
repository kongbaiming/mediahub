// Package config 负责从环境变量加载所有配置
//
// 优先级：环境变量 > .env 文件 > 默认值
package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config 是全局配置的根结构
type Config struct {
	Mode      string // debug | release
	Port      string
	LogLevel  string
	JWTSecret string

	Database DatabaseConfig
	Redis    RedisConfig
	TMDB     TMDBConfig
	OpenSubtitles OpenSubtitlesConfig
	Media    MediaConfig
	Scraper  ScraperConfig
	Transcode TranscodeConfig
	Downloader DownloaderConfig
	Indexer    IndexerConfig
	Live       LiveConfig
	AI       AIConfig
}

type DownloaderConfig struct {
	Enabled     bool
	QBittorrent QBitConfig
	WatchInterval int // 分钟（自动入库监听间隔）
}

type QBitConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	UseTLS   bool
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type TMDBConfig struct {
	APIKey    string
	Language  string
	BaseURL   string
	ImageBase string
	Timeout   int // 秒
}

type OpenSubtitlesConfig struct {
	APIKey     string
	Username   string
	Password   string
	UserAgent  string
	RESTBase   string
	XMLRPCBase string
	Timeout    int // 秒
}

type MediaConfig struct {
	Root         string
	DownloadRoot string
}

type ScraperConfig struct {
	WorkerCount int
	RetryTimes  int
}

type TranscodeConfig struct {
	Enabled      bool
	HWAccel      string
	MaxBitrate   string
	MaxHeight    int
	Preset       string
	SegmentTime  int
	PreferCopy   bool // 内网优先 HLS 流复制（4K 不重编码）
}

type IndexerConfig struct {
	URL    string
	APIKey string
}

type LiveConfig struct {
	Enabled        bool
	RTMPHost       string
	MediaMTXURL    string
	MediaMTXAPIURL string
}

type AIConfig struct {
	Enabled  bool
	Provider string // openai | claude | ollama
	APIKey   string
	Model    string
	BaseURL  string // 自定义 endpoint（Ollama 用 http://localhost:11434）
}

// Load 加载配置（先尝试读 .env，再读环境变量）
func Load() (*Config, error) {
	// 尝试加载项目根的 .env（不强制要求存在）
	_ = godotenv.Load()

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	cfg := &Config{
		Mode:      getEnv("API_MODE", "debug"),
		Port:      getEnv("API_PORT", "3000"),
		LogLevel:  getEnv("API_LOG_LEVEL", "info"),
		JWTSecret: getEnv("API_JWT_SECRET", ""),
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", ""),
		},
		TMDB: TMDBConfig{
			APIKey:    getEnv("TMDB_API_KEY", ""),
			Language:  getEnv("TMDB_LANGUAGE", "zh-CN"),
			BaseURL:   getEnv("TMDB_BASE_URL", "https://api.themoviedb.org/3"),
			ImageBase: getEnv("TMDB_IMAGE_BASE_URL", "https://image.tmdb.org/t/p"),
			Timeout:   getEnvInt("TMDB_TIMEOUT", 45),
		},
		OpenSubtitles: OpenSubtitlesConfig{
			APIKey:     getEnv("OPENSUBTITLES_API_KEY", ""),
			Username:   getEnv("OPENSUBTITLES_USERNAME", ""),
			Password:   getEnv("OPENSUBTITLES_PASSWORD", ""),
			UserAgent:  getEnv("OPENSUBTITLES_USER_AGENT", "MediaHub v0.5"),
			RESTBase:   getEnv("OPENSUBTITLES_REST_BASE", "https://api.opensubtitles.com/api/v1"),
			XMLRPCBase: getEnv("OPENSUBTITLES_XMLRPC_BASE", "https://api.opensubtitles.org/xml-rpc"),
			Timeout:    getEnvInt("OPENSUBTITLES_TIMEOUT", 20),
		},
		Media: MediaConfig{
			Root:         getEnv("MEDIA_ROOT", "/media"),
			DownloadRoot: getEnv("DOWNLOAD_ROOT", "/downloads"),
		},
		Scraper: ScraperConfig{
			WorkerCount: getEnvInt("SCRAPER_WORKER_COUNT", 3),
			RetryTimes:  getEnvInt("SCRAPER_RETRY_TIMES", 3),
		},
		Transcode: TranscodeConfig{
			Enabled:     getEnv("TRANSCODE_ENABLED", "true") == "true",
			HWAccel:     getEnv("TRANSCODE_HW_ACCEL", "qsv"),
			MaxBitrate:  getEnv("TRANSCODE_MAX_BITRATE", "2500k"),
			MaxHeight:   getEnvInt("TRANSCODE_MAX_HEIGHT", 480),
			Preset:      getEnv("TRANSCODE_PRESET", "ultrafast"),
			SegmentTime: getEnvInt("TRANSCODE_SEGMENT_TIME", 4),
			PreferCopy:  getEnv("TRANSCODE_PREFER_COPY", "true") == "true",
		},
		Downloader: DownloaderConfig{
			Enabled:      getEnv("DOWNLOADER_ENABLED", "true") == "true",
			WatchInterval: getEnvInt("DOWNLOADER_WATCH_INTERVAL", 1),
			QBittorrent: QBitConfig{
				Host:     getEnv("QBIT_HOST", "qbittorrent"),
				Port:     atoiEnv(getEnv("QBIT_PORT", "8080"), 8080),
				Username: getEnv("QBIT_USER", "admin"),
				Password: getEnv("QBIT_PASSWORD", "adminadmin"),
				UseTLS:   getEnv("QBIT_TLS", "false") == "true",
			},
		},
		Indexer: IndexerConfig{
			URL:    getEnv("INDEXER_URL", ""),
			APIKey: getEnv("INDEXER_API_KEY", ""),
		},
		Live: LiveConfig{
			Enabled:        getEnv("LIVE_ENABLED", "true") == "true",
			RTMPHost:       getEnv("LIVE_RTMP_HOST", "localhost:1935"),
			MediaMTXURL:    getEnv("LIVE_MEDIAMTX_URL", "http://mediamtx:8888"),
			MediaMTXAPIURL: getEnv("LIVE_MEDIAMTX_API_URL", "http://mediamtx:9997"),
		},
		AI: AIConfig{
			Enabled:  getEnv("AI_ENABLED", "false") == "true",
			Provider: getEnv("AI_PROVIDER", "ollama"),
			APIKey:   getEnv("AI_API_KEY", ""),
			Model:    getEnv("AI_MODEL", "gpt-4o"),
			BaseURL:  getEnv("AI_BASE_URL", ""),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 校验关键配置
func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("API_JWT_SECRET 不能为空")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("API_JWT_SECRET 至少 32 字符")
	}
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL 不能为空")
	}
	if c.Redis.URL == "" {
		return fmt.Errorf("REDIS_URL 不能为空")
	}
	if c.AI.Enabled {
		switch c.AI.Provider {
		case "openai", "ollama":
			// ok
		default:
			return fmt.Errorf("不支持的 AI_PROVIDER: %s (支持 openai, ollama)", c.AI.Provider)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := viper.GetString(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := viper.GetInt(key); v != 0 {
		return v
	}
	return fallback
}

func atoiEnv(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
