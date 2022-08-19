// Copyright (c) 2022 10X Genomics, Inc. All rights reserved.

// Functions for efficiently and reliably writing files atomically.
//
// A naiive implementation (as is used in the generic version) will
// write to a temporary file and then rename it over the target file.
//
// The problem with that approach is that there can be a race whereby
// the directory is renamed (and possibly replaced) between opening
// the temporary file for writing and doing the rename.
//
// There is also a small efficiency penalty involved in repeatedly traversing
// the path to the target directory.

package core

import (
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

func writeAtomic(target string, data []byte) error {
	dir, target := filepath.Split(target)
	dirFd, err := openAt(unix.AT_FDCWD, dir,
		unix.O_DIRECTORY|unix.O_PATH|os.O_RDONLY, 0644)
	if err != nil {
		return wrapAtomicError("could not open target directory", err)
	}
	defer closeHandle(dirFd)
	return writeAtomicAt(dirFd, target, data)
}

func openAt(dirFd int, target string, flags int, mode uint32) (int, error) {
	for {
		fd, err := unix.Openat(dirFd, target,
			flags|unix.O_CLOEXEC|unix.O_NONBLOCK, mode)
		if err != syscall.EINTR {
			return fd, err
		}
	}
}

func closeHandle(fd int) {
	for {
		err := unix.Close(fd)
		if err != syscall.EINTR {
			return
		}
	}
}

func fstat(fd int, stat *unix.Stat_t) error {
	for {
		err := unix.Fstat(fd, stat)
		if err != syscall.EINTR {
			return err
		}
	}
}

func renameat(dirFd int, src, dest string) error {
	for {
		err := unix.Renameat(dirFd, src, dirFd, dest)
		if err != syscall.EINTR {
			return err
		}
	}
}

func unlinkat(dirFd int, target string) {
	for {
		err := unix.Unlinkat(dirFd, target, 0)
		if err != syscall.EINTR {
			return
		}
	}
}

func openFileAt(dirFd int, target string, flags int, mode uint32) (*os.File, error) {
	fd, err := openAt(dirFd, target, flags, mode)
	if err != nil {
		return nil, wrapAtomicError("could not open file:", err)
	}
	return os.NewFile(uintptr(fd), target), nil
}

func writeFileAt(dirFd int, target string, data []byte) error {
	file, err := openFileAt(dirFd, target,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC|unix.O_NOCTTY|unix.O_NOFOLLOW, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return wrapAtomicError("could not write file:", err)
	}
	return wrapAtomicError("could not close file:", file.Close())
}

// Write a file in a directory by writing it to a temporary file and then
// renaming that temporary file to the target name.
//
// This way, the target name never contains an incompletely-written file name.
//
// This is using `Openat` and `Renameat`, rather than the usual `os.Open()`, to
// make it resilient against the directory being renamed while the operation is
// in progress.
func writeAtomicAt(dirFd int, target string, data []byte) error {
	tmp := target + ".tmp"
	err := writeFileAt(dirFd, tmp, data)
	if err != nil {
		return err
	}
	// On heavily loaded NFS servers, if a request is taking a long time,
	// the client will re-send the request.  The server is supposed to note
	// that the request is a duplicate and de-duplicate it, but if it's
	// heavily loaded, its duplicate request cache might have already been
	// flushed, in which case the second request will see ENOENT and fail.
	// So, we ignore `IsNotExist` errors.
	if err := renameat(dirFd, tmp, target); err != nil && !os.IsNotExist(err) {
		unlinkat(dirFd, tmp) // attempt cleanup
		return wrapAtomicError("could not rename temp file", err)
	}
	return nil
}
