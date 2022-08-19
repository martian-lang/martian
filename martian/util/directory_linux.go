// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Linux-specific directory entry utilities.

package util

import (
	"bytes"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func openAt(dirFd int, target string, flags int, mode uint32) (int, error) {
	for {
		fd, err := unix.Openat(dirFd, target,
			flags|unix.O_CLOEXEC|unix.O_NONBLOCK, mode)
		if err != syscall.EINTR {
			return fd, err
		}
	}
}

func fstat(fd int, stat *unix.Stat_t) error {
	for {
		if err := unix.Fstat(fd, stat); err != syscall.EINTR {
			return err
		}
	}
}

// CountDirNames returns the number of files in the directory opened with the
// given file descriptor.  Unlike the generic implementation, it avoids the
// string allocation overhead of actually getting the names out.
func CountDirNames(fd int) (int, error) {
	count := 0
	var buf [4096]byte
	for {
		read := buf[:]
		end, errno := syscall.Getdents(fd, read)
		if errno != nil || end <= 0 {
			return count, errno
		}
		read = read[:end:end]
		for len(read) > 0 {
			if s := dirEntSize(read); s > 0 {
				const nameOffset = int(unsafe.Offsetof(syscall.Dirent{}.Name))
				// Don't count '.' or '..'
				if nameLen := s - nameOffset; nameLen > 0 && (read[nameOffset] != '.' ||
					(nameLen > 1 && read[nameOffset+1] != '.' && read[nameOffset+1] != 0) ||
					(nameLen > 2 && read[nameOffset+2] != 0)) {
					count++
				}
				read = read[s:]
			} else {
				read = nil
			}
		}
	}
}

const dirEntLenSize = int(unsafe.Sizeof(syscall.Dirent{}.Reclen))

var bigEndian = func() bool {
	var i uint16 = 0x1
	return (*[2]byte)(unsafe.Pointer(&i))[0] == 0
}()

func readInt(buf []byte) int {
	if dirEntLenSize == 1 {
		return int(buf[0])
	} else if bigEndian {
		return readIntBE(buf)
	} else {
		return readIntLE(buf)
	}
}

// big-endian read.
func readIntBE(buf []byte) int {
	// Note the order of array access.  See golang.org/issue/14808
	switch dirEntLenSize {
	case 2:
		return int(buf[1]) | int(buf[0])<<8
	case 4:
		return int(buf[3]) | int(buf[2])<<8 | int(buf[1])<<16 | int(buf[0])<<24
	case 8:
		return int(buf[7]) | int(buf[6])<<8 | int(buf[5])<<16 | int(buf[4])<<24 |
			int(buf[3])<<32 | int(buf[2])<<40 | int(buf[1])<<48 | int(buf[0])<<56
	default:
		return int(buf[0])
	}
}

// little-endian read.
func readIntLE(buf []byte) int {
	// Note the order of array access.  See golang.org/issue/14808
	switch dirEntLenSize {
	case 2:
		return int(buf[1])<<8 | int(buf[0])
	case 4:
		return int(buf[3])<<24 | int(buf[2])<<16 | int(buf[1])<<8 | int(buf[0])
	case 8:
		return int(buf[7])<<56 | int(buf[6])<<48 | int(buf[5])<<40 | int(buf[4])<<32 |
			int(buf[3])<<24 | int(buf[2])<<16 | int(buf[1])<<8 | int(buf[0])
	default:
		return int(buf[0])
	}
}

func dirEntSize(buf []byte) int {
	const offset = int(unsafe.Offsetof(syscall.Dirent{}.Reclen))
	if len(buf) < offset+dirEntLenSize {
		return 0
	}
	return readInt(buf[offset:])
}

// ReadFileAt returns all of the bytes in a file opened relative to the
// directory with the given file descriptor.
func ReadFileAt(dirFd int, name string) (b []byte, err error) {
	if fd, ferr := openAt(dirFd, name,
		unix.O_RDONLY, 0); ferr != nil {
		return nil, ferr
	} else {
		// Do not pass the actual file name, because that causes name to escape
		// to the heap (and it's not useful except for error messages, which
		// should not happen at this point).
		f := os.NewFile(uintptr(fd), "file")
		defer f.Close()

		// capture panic if buffer size is too big.
		defer func() {
			if e := recover(); e == nil {
				return
			} else if perr, ok := e.(error); ok && perr == bytes.ErrTooLarge {
				err = perr
			} else {
				panic(e)
			}
		}()

		var buf bytes.Buffer
		if fi, err := f.Stat(); err == nil {
			if size := fi.Size() + bytes.MinRead; size > bytes.MinRead &&
				int64(int(size)) == size {
				// As initial capacity, use file Size + a little extra to
				// avoid another allocation after Read has filled the
				// buffer. If the size was wrong, we'll either waste some
				// space off the end or reallocate as needed, but in the
				// overwhelmingly common case we'll get it just right.
				buf.Grow(int(size))
			} else {
				buf.Grow(bytes.MinRead)
			}
		} else {
			buf.Grow(bytes.MinRead)
		}
		_, rerr := buf.ReadFrom(f)
		return buf.Bytes(), rerr
	}
}

// ReadFileAt reads all of the bytes in a file opened relative to the
// directory with the given file descriptor into the supplied buffer.
func ReadFileInto(dirFd int, name string, buf *bytes.Buffer) (err error) {
	if fd, ferr := openAt(dirFd, name,
		unix.O_RDONLY, 0); ferr != nil {
		return ferr
	} else {
		// Do not pass the actual file name, because that causes name to escape
		// to the heap (and it's not useful except for error messages, which
		// should not happen at this point).
		f := os.NewFile(uintptr(fd), "file")
		defer f.Close()

		// capture panic if buffer size is too big.
		defer func() {
			if e := recover(); e == nil {
				return
			} else if perr, ok := e.(error); ok && perr == bytes.ErrTooLarge {
				err = perr
			} else {
				panic(e)
			}
		}()

		if fi, err := f.Stat(); err == nil {
			if size := fi.Size() + bytes.MinRead; size > bytes.MinRead &&
				int64(int(size)) == size {
				// As initial capacity, use file Size + a little extra to
				// avoid another allocation after Read has filled the
				// buffer. If the size was wrong, we'll either waste some
				// space off the end or reallocate as needed, but in the
				// overwhelmingly common case we'll get it just right.
				buf.Grow(int(size))
			} else {
				buf.Grow(bytes.MinRead)
			}
		} else {
			buf.Grow(bytes.MinRead)
		}
		_, rerr := buf.ReadFrom(f)
		return rerr
	}
}
