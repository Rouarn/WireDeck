package platform

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// ErrAlreadyRunning 表示已经有实例持有单实例锁。
var ErrAlreadyRunning = errors.New("WireDeck 已在运行")

// InstanceLock 持有进程级文件锁，进程退出时自动释放。
type InstanceLock struct {
	fl *flock.Flock
}

// LockSingleInstance 在用户配置目录下创建并尝试获取单实例锁。
func LockSingleInstance(appName string) (*InstanceLock, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	fl := flock.New(filepath.Join(dir, "instance.lock"))
	locked, err := fl.TryLock()
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, ErrAlreadyRunning
	}

	return &InstanceLock{fl: fl}, nil
}

// Unlock 释放单实例锁。
func (l *InstanceLock) Unlock() error {
	if l == nil || l.fl == nil {
		return nil
	}
	return l.fl.Unlock()
}
