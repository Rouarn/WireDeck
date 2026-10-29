package platform

import (
	"errors"
	"os"
	"path/filepath"
	"time"

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

	// Wails 在构建期会短暂启动应用以生成绑定，可能与已有实例产生竞争。
	// 这里做有限次重试：正常情况下几毫秒即可拿到锁，异常情况下才判定为已运行。
	for attempt := 0; attempt < 3; attempt++ {
		locked, err := fl.TryLock()
		if err != nil {
			return nil, err
		}
		if locked {
			return &InstanceLock{fl: fl}, nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil, ErrAlreadyRunning
}

// Unlock 释放单实例锁。
func (l *InstanceLock) Unlock() error {
	if l == nil || l.fl == nil {
		return nil
	}
	return l.fl.Unlock()
}
