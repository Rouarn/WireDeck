package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// WindowConfig 保存窗口位置与大小。
type WindowConfig struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

const (
	defaultWidth  = 1180
	defaultHeight = 760
)

func windowConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "WireDeck", "window.json")
}

// LoadWindowConfig 读取窗口配置，不存在或无效时返回默认值。
func LoadWindowConfig() WindowConfig {
	cfg := WindowConfig{Width: defaultWidth, Height: defaultHeight}

	path := windowConfigPath()
	if path == "" {
		return cfg
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return WindowConfig{Width: defaultWidth, Height: defaultHeight}
	}
	if cfg.Width < 960 {
		cfg.Width = defaultWidth
	}
	if cfg.Height < 640 {
		cfg.Height = defaultHeight
	}
	return cfg
}

// SaveWindowConfig 保存窗口配置。
func SaveWindowConfig(cfg WindowConfig) {
	path := windowConfigPath()
	if path == "" {
		return
	}

	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}
