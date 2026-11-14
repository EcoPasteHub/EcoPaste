//go:build darwin

package app

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nsgo-dev/nsgo/core"
)

const instanceLockName = "instance.lock"

type instanceLock struct {
	file *os.File
}

func (l *instanceLock) Close() error {
	return l.file.Close()
}

// lockInstance takes the per-data-root lock. The second caller learns that
// another process is running instead of opening the database a second time.
func lockInstance(dir string) (*instanceLock, bool, error) {
	path := instancePath(dir)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, false, fmt.Errorf("打开实例锁: %w", err)
	}
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		if err := writeInstancePID(file); err != nil {
			file.Close()
			return nil, false, err
		}
		return &instanceLock{file: file}, false, nil
	}
	file.Close()
	if !errors.Is(err, syscall.EWOULDBLOCK) {
		return nil, false, fmt.Errorf("锁定实例: %w", err)
	}
	return nil, true, nil
}

func writeInstancePID(file *os.File) error {
	pid := strconv.Itoa(os.Getpid())
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("写入实例锁: %w", err)
	}
	if _, err := file.WriteAt([]byte(pid), 0); err != nil {
		return fmt.Errorf("写入实例锁: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("写入实例锁: %w", err)
	}
	return nil
}

// signalInstance asks the running instance to open its preferences. The
// lock file may be observed between locking and writing the PID, so read
// it a few times before giving up.
func signalInstance(dir string) error {
	var pid int
	var err error
	for range 5 {
		pid, err = instancePID(dir)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		return err
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("查找运行实例: %w", err)
	}
	if err := process.Signal(syscall.SIGUSR1); err != nil {
		return fmt.Errorf("通知运行实例: %w", err)
	}
	return nil
}

func instancePID(dir string) (int, error) {
	data, err := os.ReadFile(instancePath(dir))
	if err != nil {
		return 0, fmt.Errorf("读取实例锁: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("实例锁没有有效进程号")
	}
	return pid, nil
}

func instancePath(dir string) string {
	return filepath.Join(dir, instanceLockName)
}

// instanceRequests turns the activation signal into requests the app can
// handle on its main thread.
func instanceRequests(cx *core.Scope) <-chan struct{} {
	requests := make(chan struct{}, 1)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)
	go func() {
		defer signal.Stop(signals)
		for {
			select {
			case <-cx.Context().Done():
				return
			case <-signals:
				select {
				case requests <- struct{}{}:
				default:
				}
			}
		}
	}()
	return requests
}
