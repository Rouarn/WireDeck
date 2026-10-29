package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"woyuxiaji.cn/wiredeck/internal/model"
)

const historyFileName = "device-history.json"

// historyFilePath 返回设备历史文件路径。
func historyFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "WireDeck", historyFileName)
}

func (s *DeviceService) loadHistory() {
	path := historyFilePath()
	if path == "" {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var entries []model.DeviceHistory
	if err := json.Unmarshal(data, &entries); err != nil {
		return
	}

	s.mu.Lock()
	s.history = entries
	s.mu.Unlock()
}

func (s *DeviceService) saveHistory() {
	path := historyFilePath()
	if path == "" {
		return
	}

	s.mu.RLock()
	data, err := json.MarshalIndent(s.history, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return
	}

	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, data, 0o644)
}

// RecordConnection 记录一次成功的无线连接，供下次启动自动重连。
func (s *DeviceService) RecordConnection(host string, port int, name string) {
	if host == "" || port <= 0 {
		return
	}

	now := time.Now().UnixMilli()

	s.mu.Lock()
	updated := false
	for i := range s.history {
		if s.history[i].Host == host && s.history[i].Port == port {
			s.history[i].Name = name
			s.history[i].LastSeen = now
			updated = true
			break
		}
	}
	if !updated {
		s.history = append(s.history, model.DeviceHistory{
			Host:     host,
			Port:     port,
			Name:     name,
			LastSeen: now,
		})
		// 最多保留 20 条历史。
		if len(s.history) > 20 {
			s.history = s.history[len(s.history)-20:]
		}
	}
	s.mu.Unlock()

	s.saveHistory()
}

// ReconnectKnown 尝试重连所有历史设备，并自动连接 mDNS 发现的无线设备。
func (s *DeviceService) ReconnectKnown(ctx context.Context) {
	// 1. mDNS 发现：已配对设备重新广播时，直接 connect。
	if services, err := s.Discover(ctx); err == nil {
		for _, svc := range services {
			if svc.Kind == "connect" || svc.Kind == "pairing" {
				_, _ = s.server.Run(ctx, 10*time.Second, "connect", svc.Address)
			}
		}
	}

	// 2. 历史地址：端口可能已经变化，连接失败的静默忽略。
	s.mu.RLock()
	entries := append([]model.DeviceHistory(nil), s.history...)
	s.mu.RUnlock()

	for _, entry := range entries {
		if entry.Host == "" || entry.Port <= 0 {
			continue
		}
		addr := fmt.Sprintf("%s:%d", entry.Host, entry.Port)
		_, _ = s.server.Run(ctx, 8*time.Second, "connect", addr)
	}

	s.RefreshAndEmit(ctx)
}

// History 返回当前保存的历史连接（用于设置页展示）。
func (s *DeviceService) History() []model.DeviceHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]model.DeviceHistory(nil), s.history...)
}

// FormatHistoryLabel 生成历史设备的人类可读标签。
func FormatHistoryLabel(entry model.DeviceHistory) string {
	name := strings.TrimSpace(entry.Name)
	if name == "" {
		name = "未知设备"
	}
	return fmt.Sprintf("%s (%s:%d)", name, entry.Host, entry.Port)
}
