//go:build linux || darwin

// Package localstate provides private, bounded, crash-durable local files and
// OS-held advisory locks. Lock files remain on disk; the kernel owns the lease.
package localstate

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

var ErrBusy = errors.New("local state is in use")

func Check(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	return check(info, false)
}

func check(info os.FileInfo, dir bool) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Getuid()) || info.Mode()&os.ModeSymlink != 0 || info.IsDir() != dir || (!dir && !info.Mode().IsRegular()) || info.Mode().Perm()&0077 != 0 {
		return errors.New("unsafe private state ownership or permissions")
	}
	return nil
}

func Directory(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	return check(info, true)
}

func open(path string, flags int) (*os.File, error) {
	fd, err := syscall.Open(path, flags|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err == nil {
		err = check(info, false)
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func Lock(path string, wait bool) (*os.File, error) {
	f, err := open(path, syscall.O_RDWR|syscall.O_CREAT)
	if err != nil {
		return nil, err
	}
	flags := syscall.LOCK_EX
	if !wait {
		flags |= syscall.LOCK_NB
	}
	if err := syscall.Flock(int(f.Fd()), flags); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrBusy
		}
		return nil, err
	}
	return f, nil
}

func Read(path string, max int64) ([]byte, error) {
	f, err := open(path, syscall.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errors.New("local state exceeds limit")
	}
	return data, nil
}

func Write(path string, data []byte) error {
	if info, err := os.Lstat(path); err == nil {
		if err := check(info, false); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".pending-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
