// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

// Linux-specific implementation of filepath.Walk.  Faster, especially on NFS,
// because it uses Openat and Fstatat to avoid forcing extra dirent syncs.
// Also safer in the face of move ops.

package util

import (
	"os"
	"path"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// Implements the os.FileInfo interface to wrap unix.Stat_t
type unixFileInfo struct {
	name string
	sys  unix.Stat_t
}

// base name of the file
func (info *unixFileInfo) Name() string {
	if info == nil {
		return ""
	}
	return info.name
}

// Length in bytes for regular files; system-dependent for others.
func (info *unixFileInfo) Size() int64 {
	if info == nil {
		return 0
	}
	return info.sys.Size
}

// File mode bits
func (info *unixFileInfo) Mode() os.FileMode {
	if info == nil {
		return 0
	}
	return os.FileMode(info.sys.Mode & (0777 | unix.S_IFMT))
}

// Modification time
func (info *unixFileInfo) ModTime() time.Time {
	if info == nil {
		return time.Time{}
	}
	return time.Unix(info.sys.Mtim.Sec, info.sys.Mtim.Nsec)
}

// Abbreviation for Mode().IsDir()
func (info *unixFileInfo) IsDir() bool {
	if info == nil {
		return false
	}
	return info.sys.Mode&unix.S_IFDIR == unix.S_IFDIR
}

// Underlying data source - nil or *unix.Stat_t
func (info *unixFileInfo) Sys() interface{} {
	if info == nil {
		return nil
	}
	return &info.sys
}

// Faster, Unix-specific implementation of filepath.Walk, which avoids the
// directory sort and uses openat and fstatat to avoid forcing extra dirent
// syncs.  This can save a lot of time on NFS servers, and also provides more
// consistent behavior in the face of directory renames or changes to the
// current process working directory.
//
// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. Walk does not follow symbolic links.
//
// Unlike filepath.Walk, Walk makes ABSOLUTELY NO GUARANTEES about the order
// in which the files are traversed.
//
// Unlike filepath.Walk, the Sys() method of the os.FileInfo object passed to
// walkFn will be of type golang.org/x/sys/unix.Stat_t.
func Walk(root string, walkFn filepath.WalkFunc) error {
	if start, err := os.Open(root); err != nil {
		return walkFn(root, nil, err)
	} else {
		info := unixFileInfo{name: path.Base(root)}
		if err := unix.Fstat(int(start.Fd()), &info.sys); err != nil {
			start.Close()
			return walkFn(root, &info, err)
		}
		if err := walkFn(root, &info, nil); err == filepath.SkipDir {
			start.Close()
			return nil
		} else if err != nil {
			start.Close()
			return err
		} else if info.IsDir() {
			return walkInternal(root, start, walkFn)
		} else {
			start.Close()
			return nil
		}
	}
}

func walkInternal(root string, start *os.File, walkFn filepath.WalkFunc) error {

	defer func() {
		if start != nil {
			start.Close()
		}
	}()

	if list, err := start.Readdirnames(-1); err != nil {
		if !os.IsNotExist(err) {
			if err := walkFn(path.Join(root, start.Name()), nil, err); err != filepath.SkipDir {
				return err
			} else {
				return nil
			}
		} else {
			return nil
		}
	} else {
		dirs := make([]string, 0, len(list))
		startFd := int(start.Fd())
		for _, name := range list {
			info := unixFileInfo{name: name}
			if err := unix.Fstatat(startFd, name,
				&info.sys,
				unix.AT_SYMLINK_NOFOLLOW|unix.AT_NO_AUTOMOUNT); err != nil {
				if walkFn(path.Join(root, name), &info, err); err != nil && err != filepath.SkipDir {
					return err
				} else if err == filepath.SkipDir {
					return nil
				}
			} else if werr := walkFn(path.Join(root, name), &info, err); werr != nil {
				if werr != filepath.SkipDir {
					return werr
				} else if !info.IsDir() {
					return nil
				}
			} else if info.IsDir() {
				dirs = append(dirs, name)
			}
		}
		for i, dirName := range dirs {
			if fd, err := unix.Openat(
				startFd,
				dirName,
				os.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC,
				0); err != nil {
				if ferr := walkFn(path.Join(root, dirName), nil, err); ferr != nil && ferr != filepath.SkipDir {
					return ferr
				}
			} else {
				// Close this FD before recursing to limit the number
				// of FDs we have open.  It's unavoidable to have a bunch
				// for trees that are deep and wide, but this cuts the FD
				// count in the common case of directories with a single
				// child.
				if i == len(dirs)-1 {
					start.Close()
					start = nil
				}
				if err := walkInternal(path.Join(root, dirName),
					os.NewFile(uintptr(fd),
						path.Join(dirName)), walkFn); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
