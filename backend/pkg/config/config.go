package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

// ─── Public Config types ──────────────────────────────────────────────────────

type Config struct {
	App          AppConfig
	Server       ServerConfig
	DB           DBConfig
	Redis        RedisConfig
	Auth         AuthConfig
	R2           R2Config
	Queue        QueueConfig
	FFmpeg       FFmpegConfig
	Gemini       GeminiConfig
	TikTok       TikTokConfig
	Facebook     FacebookConfig
	Pexels       PexelsConfig
	Pixabay      PixabayConfig
	YouTube      YouTubeConfig
	Reddit       RedditConfig
	GoogleTrends GoogleTrendsConfig
	EdgeTTS      EdgeTTSConfig
	Schedule     ScheduleConfig
	Channel      ChannelConfig
	Publish      PublishConfig
	Video        VideoConfig
	MediaCollect MediaCollectConfig
}

type AppConfig struct {
	Env         string
	Port        int
	FrontendURL string
}

type ServerConfig struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type DBConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	PingTimeout     time.Duration
}

type RedisConfig struct {
	URL         string
	PingTimeout time.Duration
}

type AuthConfig struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	EncryptionKey   string
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicURL       string
}

type QueueWeights struct {
	Critical int
	Default  int
	Low      int
}

type QueueConfig struct {
	GeneralConcurrency int
	VideoConcurrency   int
	Weights            QueueWeights
}

type FFmpegConfig struct {
	OutputWidth  int
	OutputHeight int
	OutputFPS    int
	OutputCRF    int
	AudioBitrate string
	Preset       string
	TempDir      string
}

type GeminiConfig struct {
	APIKey      string
	Model       string
	APIBase     string
	HTTPTimeout time.Duration
}

type TikTokAPIConfig struct {
	AuthBaseURL      string
	TokenURL         string
	UserInfoURL      string
	PublishInitURL   string
	PublishStatusURL string
	VideoQueryURL    string
	ShopBaseURL      string
}

type TikTokConfig struct {
	ClientKey     string
	ClientSecret  string
	RedirectURL   string
	ShopAPIKey    string
	ShopAPISecret string
	HTTPTimeout   time.Duration
	API           TikTokAPIConfig
}

type FacebookAPIConfig struct {
	Version      string
	AuthBaseURL  string
	TokenURL     string
	GraphBaseURL string
}

type FacebookConfig struct {
	AppID       string
	AppSecret   string
	RedirectURL string
	HTTPTimeout time.Duration
	API         FacebookAPIConfig
}

type PexelsConfig struct {
	APIKey      string
	APIBase     string
	HTTPTimeout time.Duration
}

type PixabayConfig struct {
	APIKey      string
	APIBase     string
	HTTPTimeout time.Duration
}

type YouTubeConfig struct {
	APIKey      string
	APIBase     string
	HTTPTimeout time.Duration
}

type RedditConfig struct {
	APIBase     string
	HTTPTimeout time.Duration
}

type GoogleTrendsConfig struct {
	APIBase     string
	HTTPTimeout time.Duration
}

type EdgeTTSVoices struct {
	EnFemale string
	EnMale   string
	ViMale   string
}

type EdgeTTSConfig struct {
	DefaultVoice string
	Voices       EdgeTTSVoices
}

type ScheduleConfig struct {
	CheckPublish   string
	DiscoverTrends string
	SyncAnalytics  string
	RefreshTokens  string
}

type ChannelConfig struct {
	FacebookTokenExpiry time.Duration
}

type PublishConfig struct {
	MinScheduleBeforeNow time.Duration
}

type VideoConfig struct {
	PresignedURLTTL    time.Duration
	TargetDurationSecs int
	MaxClips           int
}

type MediaCollectConfig struct {
	HTTPTimeout     time.Duration
	AssetsPerSearch int
}

// ─── Raw YAML types ───────────────────────────────────────────────────────────

type rawConfig struct {
	App          rawApp          `yaml:"app"`
	Server       rawServer       `yaml:"server"`
	DB           rawDB           `yaml:"db"`
	Redis        rawRedis        `yaml:"redis"`
	Auth         rawAuth         `yaml:"auth"`
	R2           rawR2           `yaml:"r2"`
	Queue        rawQueue        `yaml:"queue"`
	FFmpeg       rawFFmpeg       `yaml:"ffmpeg"`
	Gemini       rawGemini       `yaml:"gemini"`
	TikTok       rawTikTok       `yaml:"tiktok"`
	Facebook     rawFacebook     `yaml:"facebook"`
	Pexels       rawPexels       `yaml:"pexels"`
	Pixabay      rawPixabay      `yaml:"pixabay"`
	YouTube      rawYouTube      `yaml:"youtube"`
	Reddit       rawReddit       `yaml:"reddit"`
	GoogleTrends rawGoogleTrends `yaml:"google_trends"`
	EdgeTTS      rawEdgeTTS      `yaml:"edgetts"`
	Schedule     rawSchedule     `yaml:"schedule"`
	Channel      rawChannel      `yaml:"channel"`
	Publish      rawPublish      `yaml:"publish"`
	Video        rawVideo        `yaml:"video"`
	MediaCollect rawMediaCollect `yaml:"media_collect"`
}

type rawApp struct {
	Env         string `yaml:"env"`
	Port        string `yaml:"port"`
	FrontendURL string `yaml:"frontend_url"`
}

type rawServer struct {
	ReadTimeout     string `yaml:"read_timeout"`
	WriteTimeout    string `yaml:"write_timeout"`
	IdleTimeout     string `yaml:"idle_timeout"`
	ShutdownTimeout string `yaml:"shutdown_timeout"`
}

type rawDB struct {
	URL             string `yaml:"url"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
	PingTimeout     string `yaml:"ping_timeout"`
}

type rawRedis struct {
	URL         string `yaml:"url"`
	PingTimeout string `yaml:"ping_timeout"`
}

type rawAuth struct {
	JWTSecret       string `yaml:"jwt_secret"`
	EncryptionKey   string `yaml:"encryption_key"`
	AccessTokenTTL  string `yaml:"access_token_ttl"`
	RefreshTokenTTL string `yaml:"refresh_token_ttl"`
}

type rawR2 struct {
	AccountID       string `yaml:"account_id"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	BucketName      string `yaml:"bucket_name"`
	PublicURL       string `yaml:"public_url"`
}

type rawQueueWeights struct {
	Critical int `yaml:"critical"`
	Default  int `yaml:"default"`
	Low      int `yaml:"low"`
}

type rawQueue struct {
	GeneralConcurrency int             `yaml:"general_concurrency"`
	VideoConcurrency   int             `yaml:"video_concurrency"`
	Weights            rawQueueWeights `yaml:"weights"`
}

type rawFFmpeg struct {
	OutputWidth  int    `yaml:"output_width"`
	OutputHeight int    `yaml:"output_height"`
	OutputFPS    int    `yaml:"output_fps"`
	OutputCRF    int    `yaml:"output_crf"`
	AudioBitrate string `yaml:"audio_bitrate"`
	Preset       string `yaml:"preset"`
	TempDir      string `yaml:"temp_dir"`
}

type rawGemini struct {
	APIKey      string `yaml:"api_key"`
	Model       string `yaml:"model"`
	APIBase     string `yaml:"api_base"`
	HTTPTimeout string `yaml:"http_timeout"`
}

type rawTikTokAPI struct {
	AuthBaseURL      string `yaml:"auth_base_url"`
	TokenURL         string `yaml:"token_url"`
	UserInfoURL      string `yaml:"user_info_url"`
	PublishInitURL   string `yaml:"publish_init_url"`
	PublishStatusURL string `yaml:"publish_status_url"`
	VideoQueryURL    string `yaml:"video_query_url"`
	ShopBaseURL      string `yaml:"shop_base_url"`
}

type rawTikTok struct {
	ClientKey     string       `yaml:"client_key"`
	ClientSecret  string       `yaml:"client_secret"`
	RedirectURL   string       `yaml:"redirect_url"`
	ShopAPIKey    string       `yaml:"shop_api_key"`
	ShopAPISecret string       `yaml:"shop_api_secret"`
	HTTPTimeout   string       `yaml:"http_timeout"`
	API           rawTikTokAPI `yaml:"api"`
}

type rawFacebookAPI struct {
	Version      string `yaml:"version"`
	AuthBaseURL  string `yaml:"auth_base_url"`
	TokenURL     string `yaml:"token_url"`
	GraphBaseURL string `yaml:"graph_base_url"`
}

type rawFacebook struct {
	AppID       string         `yaml:"app_id"`
	AppSecret   string         `yaml:"app_secret"`
	RedirectURL string         `yaml:"redirect_url"`
	HTTPTimeout string         `yaml:"http_timeout"`
	API         rawFacebookAPI `yaml:"api"`
}

type rawPexels struct {
	APIKey      string `yaml:"api_key"`
	APIBase     string `yaml:"api_base"`
	HTTPTimeout string `yaml:"http_timeout"`
}

type rawPixabay struct {
	APIKey      string `yaml:"api_key"`
	APIBase     string `yaml:"api_base"`
	HTTPTimeout string `yaml:"http_timeout"`
}

type rawYouTube struct {
	APIKey      string `yaml:"api_key"`
	APIBase     string `yaml:"api_base"`
	HTTPTimeout string `yaml:"http_timeout"`
}

type rawReddit struct {
	APIBase     string `yaml:"api_base"`
	HTTPTimeout string `yaml:"http_timeout"`
}

type rawGoogleTrends struct {
	APIBase     string `yaml:"api_base"`
	HTTPTimeout string `yaml:"http_timeout"`
}

type rawEdgeTTSVoices struct {
	EnFemale string `yaml:"en_female"`
	EnMale   string `yaml:"en_male"`
	ViMale   string `yaml:"vi_male"`
}

type rawEdgeTTS struct {
	DefaultVoice string           `yaml:"default_voice"`
	Voices       rawEdgeTTSVoices `yaml:"voices"`
}

type rawSchedule struct {
	CheckPublish   string `yaml:"check_publish"`
	DiscoverTrends string `yaml:"discover_trends"`
	SyncAnalytics  string `yaml:"sync_analytics"`
	RefreshTokens  string `yaml:"refresh_tokens"`
}

type rawChannel struct {
	FacebookTokenExpiry string `yaml:"facebook_token_expiry"`
}

type rawPublish struct {
	MinScheduleBeforeNow string `yaml:"min_schedule_before_now"`
}

type rawVideo struct {
	PresignedURLTTL    string `yaml:"presigned_url_ttl"`
	TargetDurationSecs int    `yaml:"target_duration_secs"`
	MaxClips           int    `yaml:"max_clips"`
}

type rawMediaCollect struct {
	HTTPTimeout     string `yaml:"http_timeout"`
	AssetsPerSearch int    `yaml:"assets_per_search"`
}

// ─── Load ─────────────────────────────────────────────────────────────────────

// Load reads the config file (CONFIG_FILE env, default ./config.yml), expands
// ${VAR} and ${VAR:-default} placeholders from the environment, then parses
// the result. No os.Getenv calls are needed anywhere else.
func Load() *Config {
	cfgFile := os.Getenv("CONFIG_FILE")
	if cfgFile == "" {
		cfgFile = "config.yml"
	}

	data, err := os.ReadFile(cfgFile)
	if err != nil {
		log.Fatalf("read config file %q: %v", cfgFile, err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal([]byte(expandEnv(string(data))), &raw); err != nil {
		log.Fatalf("parse config file %q: %v", cfgFile, err)
	}

	return &Config{
		App: AppConfig{
			Env:         raw.App.Env,
			Port:        mustInt(raw.App.Port, "app.port"),
			FrontendURL: raw.App.FrontendURL,
		},
		Server: ServerConfig{
			ReadTimeout:     mustDuration(raw.Server.ReadTimeout, "server.read_timeout"),
			WriteTimeout:    mustDuration(raw.Server.WriteTimeout, "server.write_timeout"),
			IdleTimeout:     mustDuration(raw.Server.IdleTimeout, "server.idle_timeout"),
			ShutdownTimeout: mustDuration(raw.Server.ShutdownTimeout, "server.shutdown_timeout"),
		},
		DB: DBConfig{
			URL:             mustField(raw.DB.URL, "db.url"),
			MaxOpenConns:    raw.DB.MaxOpenConns,
			MaxIdleConns:    raw.DB.MaxIdleConns,
			ConnMaxLifetime: mustDuration(raw.DB.ConnMaxLifetime, "db.conn_max_lifetime"),
			PingTimeout:     mustDuration(raw.DB.PingTimeout, "db.ping_timeout"),
		},
		Redis: RedisConfig{
			URL:         mustField(raw.Redis.URL, "redis.url"),
			PingTimeout: mustDuration(raw.Redis.PingTimeout, "redis.ping_timeout"),
		},
		Auth: AuthConfig{
			JWTSecret:       mustField(raw.Auth.JWTSecret, "auth.jwt_secret"),
			AccessTokenTTL:  mustDuration(raw.Auth.AccessTokenTTL, "auth.access_token_ttl"),
			RefreshTokenTTL: mustDuration(raw.Auth.RefreshTokenTTL, "auth.refresh_token_ttl"),
			EncryptionKey:   mustField(raw.Auth.EncryptionKey, "auth.encryption_key"),
		},
		R2: R2Config{
			AccountID:       raw.R2.AccountID,
			AccessKeyID:     raw.R2.AccessKeyID,
			SecretAccessKey: raw.R2.SecretAccessKey,
			BucketName:      raw.R2.BucketName,
			PublicURL:       raw.R2.PublicURL,
		},
		Queue: QueueConfig{
			GeneralConcurrency: raw.Queue.GeneralConcurrency,
			VideoConcurrency:   raw.Queue.VideoConcurrency,
			Weights: QueueWeights{
				Critical: raw.Queue.Weights.Critical,
				Default:  raw.Queue.Weights.Default,
				Low:      raw.Queue.Weights.Low,
			},
		},
		FFmpeg: FFmpegConfig{
			OutputWidth:  raw.FFmpeg.OutputWidth,
			OutputHeight: raw.FFmpeg.OutputHeight,
			OutputFPS:    raw.FFmpeg.OutputFPS,
			OutputCRF:    raw.FFmpeg.OutputCRF,
			AudioBitrate: raw.FFmpeg.AudioBitrate,
			Preset:       raw.FFmpeg.Preset,
			TempDir:      raw.FFmpeg.TempDir,
		},
		Gemini: GeminiConfig{
			APIKey:      raw.Gemini.APIKey,
			Model:       raw.Gemini.Model,
			APIBase:     raw.Gemini.APIBase,
			HTTPTimeout: mustDuration(raw.Gemini.HTTPTimeout, "gemini.http_timeout"),
		},
		TikTok: TikTokConfig{
			ClientKey:     raw.TikTok.ClientKey,
			ClientSecret:  raw.TikTok.ClientSecret,
			RedirectURL:   raw.TikTok.RedirectURL,
			ShopAPIKey:    raw.TikTok.ShopAPIKey,
			ShopAPISecret: raw.TikTok.ShopAPISecret,
			HTTPTimeout:   mustDuration(raw.TikTok.HTTPTimeout, "tiktok.http_timeout"),
			API: TikTokAPIConfig{
				AuthBaseURL:      raw.TikTok.API.AuthBaseURL,
				TokenURL:         raw.TikTok.API.TokenURL,
				UserInfoURL:      raw.TikTok.API.UserInfoURL,
				PublishInitURL:   raw.TikTok.API.PublishInitURL,
				PublishStatusURL: raw.TikTok.API.PublishStatusURL,
				VideoQueryURL:    raw.TikTok.API.VideoQueryURL,
				ShopBaseURL:      raw.TikTok.API.ShopBaseURL,
			},
		},
		Facebook: FacebookConfig{
			AppID:       raw.Facebook.AppID,
			AppSecret:   raw.Facebook.AppSecret,
			RedirectURL: raw.Facebook.RedirectURL,
			HTTPTimeout: mustDuration(raw.Facebook.HTTPTimeout, "facebook.http_timeout"),
			API: FacebookAPIConfig{
				Version:      raw.Facebook.API.Version,
				AuthBaseURL:  raw.Facebook.API.AuthBaseURL,
				TokenURL:     raw.Facebook.API.TokenURL,
				GraphBaseURL: raw.Facebook.API.GraphBaseURL,
			},
		},
		Pexels: PexelsConfig{
			APIKey:      raw.Pexels.APIKey,
			APIBase:     raw.Pexels.APIBase,
			HTTPTimeout: mustDuration(raw.Pexels.HTTPTimeout, "pexels.http_timeout"),
		},
		Pixabay: PixabayConfig{
			APIKey:      raw.Pixabay.APIKey,
			APIBase:     raw.Pixabay.APIBase,
			HTTPTimeout: mustDuration(raw.Pixabay.HTTPTimeout, "pixabay.http_timeout"),
		},
		YouTube: YouTubeConfig{
			APIKey:      raw.YouTube.APIKey,
			APIBase:     raw.YouTube.APIBase,
			HTTPTimeout: mustDuration(raw.YouTube.HTTPTimeout, "youtube.http_timeout"),
		},
		Reddit: RedditConfig{
			APIBase:     raw.Reddit.APIBase,
			HTTPTimeout: mustDuration(raw.Reddit.HTTPTimeout, "reddit.http_timeout"),
		},
		GoogleTrends: GoogleTrendsConfig{
			APIBase:     raw.GoogleTrends.APIBase,
			HTTPTimeout: mustDuration(raw.GoogleTrends.HTTPTimeout, "google_trends.http_timeout"),
		},
		EdgeTTS: EdgeTTSConfig{
			DefaultVoice: raw.EdgeTTS.DefaultVoice,
			Voices: EdgeTTSVoices{
				EnFemale: raw.EdgeTTS.Voices.EnFemale,
				EnMale:   raw.EdgeTTS.Voices.EnMale,
				ViMale:   raw.EdgeTTS.Voices.ViMale,
			},
		},
		Schedule: ScheduleConfig{
			CheckPublish:   raw.Schedule.CheckPublish,
			DiscoverTrends: raw.Schedule.DiscoverTrends,
			SyncAnalytics:  raw.Schedule.SyncAnalytics,
			RefreshTokens:  raw.Schedule.RefreshTokens,
		},
		Channel: ChannelConfig{
			FacebookTokenExpiry: mustDuration(raw.Channel.FacebookTokenExpiry, "channel.facebook_token_expiry"),
		},
		Publish: PublishConfig{
			MinScheduleBeforeNow: mustDuration(raw.Publish.MinScheduleBeforeNow, "publish.min_schedule_before_now"),
		},
		Video: VideoConfig{
			PresignedURLTTL:    mustDuration(raw.Video.PresignedURLTTL, "video.presigned_url_ttl"),
			TargetDurationSecs: raw.Video.TargetDurationSecs,
			MaxClips:           raw.Video.MaxClips,
		},
		MediaCollect: MediaCollectConfig{
			HTTPTimeout:     mustDuration(raw.MediaCollect.HTTPTimeout, "media_collect.http_timeout"),
			AssetsPerSearch: raw.MediaCollect.AssetsPerSearch,
		},
	}
}

// expandEnv replaces ${VAR} and ${VAR:-default} in s with environment values.
func expandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		if name, def, ok := strings.Cut(key, ":-"); ok {
			if v := os.Getenv(name); v != "" {
				return v
			}
			return def
		}
		return os.Getenv(key)
	})
}

func mustField(val, field string) string {
	if val == "" {
		log.Fatalf("required config field %q is empty (set in config.yml or via environment variable)", field)
	}
	return val
}

func mustInt(s, field string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("invalid integer for %q: %q", field, s)
	}
	return n
}

func mustDuration(s, field string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("invalid duration for %q: %q", field, s)
	}
	return d
}
