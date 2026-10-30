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

const releasesAPI = "https://api.github.com/repos/Rouarn/wiredeck/releases?per_page=10"

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

// Check 拉取发布列表并与当前版本比较。
//
// 注意：不能使用 /releases/latest —— 它只返回非 prerelease 的版本，
// 而测试版（Alpha）都是 prerelease，会返回 404。
func (s *UpdateService) Check(ctx context.Context) (model.UpdateInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesAPI, nil)
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

	var releases []struct {
		TagName     string `json:"tag_name"`
		HTMLURL     string `json:"html_url"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
		Draft       bool   `json:"draft"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return model.UpdateInfo{}, err
	}

	best := -1
	for i := range releases {
		if releases[i].Draft {
			continue
		}
		if best == -1 || compareVersion(releases[i].TagName, releases[best].TagName) > 0 {
			best = i
		}
	}
	if best == -1 {
		return model.UpdateInfo{}, fmt.Errorf("没有找到可用的发布版本")
	}

	current := strings.TrimPrefix(s.currentVersion, "v")
	latest := strings.TrimPrefix(releases[best].TagName, "v")

	return model.UpdateInfo{
		Current:   current,
		Latest:    latest,
		HasUpdate: compareVersion(latest, current) > 0,
		URL:       releases[best].HTMLURL,
		Notes:     releases[best].Body,
		Published: releases[best].PublishedAt,
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
