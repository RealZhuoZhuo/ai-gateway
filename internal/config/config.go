package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Addr string `mapstructure:"addr"`

	GatewayAPIKeys      []string        `mapstructure:"gateway_api_keys"`
	ImageModelProviders []ModelProvider `mapstructure:"image_model_providers"`
	VideoModelProviders []ModelProvider `mapstructure:"video_model_providers"`

	ArkImageEndpoint string `mapstructure:"ark_image_endpoint"`
	ArkImageAPIKey   string `mapstructure:"ark_image_api_key"`
	DashScopeBaseURL string `mapstructure:"dashscope_base_url"`
	DashScopeAPIKey  string `mapstructure:"dashscope_api_key"`
	YunwuBaseURL     string `mapstructure:"yunwu_base_url"`
	YunwuAPIKey      string `mapstructure:"yunwu_api_key"`

	ArkVideoEndpoint string `mapstructure:"ark_video_endpoint"`
	ArkVideoAPIKey   string `mapstructure:"ark_video_api_key"`

	LogLevel           string        `mapstructure:"log_level"`
	HTTPTimeoutSeconds int           `mapstructure:"http_timeout_seconds"`
	HTTPTimeout        time.Duration `mapstructure:"-"`
}

type ModelProvider struct {
	Model    string `mapstructure:"model"`
	Provider string `mapstructure:"provider"`
}

func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	addConfigPaths(v)
	v.AutomaticEnv()

	if configFile := strings.TrimSpace(v.GetString("config_file")); configFile != "" {
		v.SetConfigFile(configFile)
	}
	v.SetDefault("yunwu_base_url", "https://yunwu.ai")

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, err
		}
	}

	cfg := Config{
		Addr:               v.GetString("addr"),
		GatewayAPIKeys:     compactStrings(v.GetStringSlice("gateway_api_keys")),
		ArkImageEndpoint:   v.GetString("ark_image_endpoint"),
		ArkImageAPIKey:     v.GetString("ark_image_api_key"),
		DashScopeBaseURL:   strings.TrimRight(v.GetString("dashscope_base_url"), "/"),
		DashScopeAPIKey:    v.GetString("dashscope_api_key"),
		YunwuBaseURL:       strings.TrimRight(v.GetString("yunwu_base_url"), "/"),
		YunwuAPIKey:        v.GetString("yunwu_api_key"),
		ArkVideoEndpoint:   v.GetString("ark_video_endpoint"),
		ArkVideoAPIKey:     v.GetString("ark_video_api_key"),
		LogLevel:           v.GetString("log_level"),
		HTTPTimeoutSeconds: v.GetInt("http_timeout_seconds"),
	}
	imageModelProviders, err := loadModelProviders(v, "image_model_providers")
	if err != nil {
		return Config{}, err
	}
	videoModelProviders, err := loadModelProviders(v, "video_model_providers")
	if err != nil {
		return Config{}, err
	}
	cfg.ImageModelProviders = imageModelProviders
	cfg.VideoModelProviders = videoModelProviders
	if single := strings.TrimSpace(v.GetString("gateway_api_key")); single != "" {
		cfg.GatewayAPIKeys = append(cfg.GatewayAPIKeys, single)
	}
	if cfg.HTTPTimeoutSeconds <= 0 {
		cfg.HTTPTimeoutSeconds = 120
	}
	cfg.HTTPTimeout = time.Duration(cfg.HTTPTimeoutSeconds) * time.Second
	return cfg, nil
}

func addConfigPaths(v *viper.Viper) {
	seen := map[string]struct{}{}
	add := func(path string) {
		if path = strings.TrimSpace(path); path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err == nil {
			path = abs
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		v.AddConfigPath(path)
	}

	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; ; dir = filepath.Dir(dir) {
			add(dir)
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}

	if executable, err := os.Executable(); err == nil {
		add(filepath.Dir(executable))
	}
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func loadModelProviders(v *viper.Viper, key string) ([]ModelProvider, error) {
	var providers []ModelProvider
	_ = v.UnmarshalKey(key, &providers)
	return normalizeModelProviders(key, providers)
}

func normalizeModelProviders(key string, providers []ModelProvider) ([]ModelProvider, error) {
	out := make([]ModelProvider, 0, len(providers))
	seen := map[string]struct{}{}
	for index, item := range providers {
		item.Model = strings.TrimSpace(item.Model)
		item.Provider = normalizeProvider(item.Provider)
		if item.Model == "" && item.Provider == "" {
			continue
		}
		if item.Model == "" {
			return nil, fmt.Errorf("%s[%d].model is required", key, index)
		}
		if item.Provider == "" {
			return nil, fmt.Errorf("%s[%d].provider is required", key, index)
		}
		if !validProvider(key, item.Provider) {
			return nil, fmt.Errorf("%s[%d].provider %q is invalid", key, index, item.Provider)
		}
		if _, ok := seen[item.Model]; ok {
			return nil, fmt.Errorf("%s[%d].model %q is duplicated", key, index, item.Model)
		}
		seen[item.Model] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func validProvider(key, provider string) bool {
	switch key {
	case "image_model_providers":
		switch provider {
		case "ark", "dashscope", "yunwu":
			return true
		default:
			return false
		}
	case "video_model_providers":
		switch provider {
		case "ark", "dashscope":
			return true
		default:
			return false
		}
	default:
		switch provider {
		case "ark", "dashscope":
			return true
		default:
			return false
		}
	}
}
