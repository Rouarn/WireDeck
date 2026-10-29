package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"woyuxiaji.cn/wiredeck/internal/adb"
	"woyuxiaji.cn/wiredeck/internal/model"
)

// FileService 负责截图与文件互传。
type FileService struct {
	server *adb.Server
	emit   func(event string, data any)
}

// NewFileService 创建文件服务。
func NewFileService(server *adb.Server, emit func(string, any)) *FileService {
	return &FileService{server: server, emit: emit}
}

// Screenshot 截取设备屏幕，保存到图片目录并返回 dataURL 供预览。
func (s *FileService) Screenshot(ctx context.Context, serial, saveDir string) (*model.ScreenshotResult, error) {
	if serial == "" {
		return nil, fmt.Errorf("未选择设备")
	}

	if saveDir == "" {
		home, _ := os.UserHomeDir()
		saveDir = filepath.Join(home, "Pictures", "WireDeck")
	}
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return nil, fmt.Errorf("无法创建截图目录: %w", err)
	}

	data, err := s.server.RunRaw(ctx, 30*time.Second, "-s", serial, "exec-out", "screencap", "-p")
	if err != nil {
		return nil, err
	}
	if len(data) < 8 || data[0] != 0x89 || string(data[1:4]) != "PNG" {
		return nil, fmt.Errorf("截图数据无效（设备可能拒绝了 screencap）")
	}

	name := fmt.Sprintf("screenshot-%s.png", time.Now().Format("20060102-150405"))
	path := filepath.Join(saveDir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, fmt.Errorf("保存截图失败: %w", err)
	}

	return &model.ScreenshotResult{
		Path:    path,
		DataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data),
	}, nil
}

// Push 将本地文件推送到设备。
func (s *FileService) Push(ctx context.Context, serial, local, remote string) error {
	if serial == "" || local == "" || remote == "" {
		return fmt.Errorf("设备、本地文件与远程路径不能为空")
	}
	if info, err := os.Stat(local); err != nil || info.IsDir() {
		return fmt.Errorf("本地文件不存在: %s", local)
	}

	return s.server.PushWithProgress(ctx, serial, local, remote, func(pct int) {
		s.emitProgress("push", pct)
	})
}

// Pull 从设备拉取文件到本地。
func (s *FileService) Pull(ctx context.Context, serial, remote, local string) error {
	if serial == "" || remote == "" || local == "" {
		return fmt.Errorf("设备、远程路径与本地路径不能为空")
	}

	return s.server.PullWithProgress(ctx, serial, remote, local, func(pct int) {
		s.emitProgress("pull", pct)
	})
}

func (s *FileService) emitProgress(stage string, pct int) {
	if s.emit == nil {
		return
	}

	label := "拉取"
	if stage == "push" {
		label = "推送"
	}

	s.emit("file:progress", model.FileProgress{
		Stage:   stage,
		Percent: pct,
		Message: fmt.Sprintf("%s %d%%", label, pct),
	})
}
