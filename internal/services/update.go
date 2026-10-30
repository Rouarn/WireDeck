package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"woyuxiaji.cn/wiredeck/internal/model"
)

const releaseAPI = "https://api.github.com/repos/Rouarn/wiredeck/releases/latest"

// UpdateService 负责检查 GitHub Releases 上的新版本。
type UpdateService struct {
	currentVersion string
	client         *http.Client
}

// NewUpdateService 创建更新检查服务。
func NewUpdateService(currentVersion string) *UpdateService {
	return &UpdateService{
		currentVersion: currentVersion,
		client:         &http.Client{Timeout: 15 * time.Second},
	}
}

// Check 拉取最新 Release 并与当前版本比较。
func (s *UpdateService) Check(ctx context.Context) (model.UpdateInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPI, nil)
	if err != nil {
		return model.UpdateInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "WireDeck/"+s.currentVersion)

	resp, err := s.client.Do(req)
	if err != nil {
		return model.UpdateInfo{}, fmt.Errorf("检查更新失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return model.UpdateInfo{}, fmt.Errorf("检查更新失败: HTTP %d", resp.StatusCode)
	}

	var release struct {
		TagName     string `json:"tag_name"`
		HTMLURL     string `json:"html_url"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return model.UpdateInfo{}, err
	}

	current := strings.TrimPrefix(s.currentVersion, "v")
	latest := strings.TrimPrefix(release.TagName, "v")

	return model.UpdateInfo{
		Current:   current,
		Latest:    latest,
		HasUpdate: compareVersion(latest, current) > 0,
		URL:       release.HTMLURL,
		Notes:     release.Body,
		Published: release.PublishedAt,
	}, nil
}

// compareVersion 比较点分版本号，返回 1/0/-1。
func compareVersion(a, b string) int {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")

	for i := 0; i < len(pa) || i < len(pb); i++ {
		var na, nb int
		if i < len(pa) {
			na, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			nb, _ = strconv.Atoi(pb[i])
		}
		if na > nb {
			return 1
		}
		if na < nb {
			return -1
		}
	}
	return 0
}
