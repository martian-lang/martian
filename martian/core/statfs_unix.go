// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

package core

//
// File system query utility.
//

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// Converts the fs type magic number to a string.
//
// Source: linux/magic.h plus a few others found around the internet.
func FsTypeString(fsType int64) string {
	switch fsType {
	case unix.ANON_INODE_FS_MAGIC:
		return "anonymous"
	case 0x61756673:
		return "aufs"
	case unix.AUTOFS_SUPER_MAGIC:
		return "auto"
	case unix.AAFS_MAGIC:
		return "aafs"
	case unix.ADFS_SUPER_MAGIC:
		return "adfs"
	case unix.AFFS_SUPER_MAGIC:
		return "affs"
	case unix.AFS_FS_MAGIC, unix.AFS_SUPER_MAGIC:
		return "afs"
	case unix.BALLOON_KVM_MAGIC:
		return "ballon"
	case unix.BDEVFS_MAGIC:
		return "bdev"
	case 0x42465331:
		return "befs"
	case 0x1badface:
		return "bfs"
	case unix.BINFMTFS_MAGIC:
		return "binfmt"
	case unix.BPF_FS_MAGIC:
		return "bpf"
	case unix.BTRFS_SUPER_MAGIC, unix.BTRFS_TEST_MAGIC:
		return "btrfs"
	case 0x00C36400:
		return "ceph"
	case unix.CGROUP_SUPER_MAGIC:
		return "cgroup"
	case unix.CGROUP2_SUPER_MAGIC:
		return "cgroup2"
	case 0xff534d42:
		return "cifs"
	case unix.CODA_SUPER_MAGIC:
		return "coda"
	case 0x012ff7b7:
		return "coh"
	case unix.CRAMFS_MAGIC, 0x453dcd28:
		return "cramfs"
	case unix.DAXFS_MAGIC:
		return "dax"
	case unix.DEBUGFS_MAGIC:
		return "debugfs"
	case 0x1373:
		return "dev"
	case unix.DEVPTS_SUPER_MAGIC:
		return "devpts"
	case unix.ECRYPTFS_SUPER_MAGIC:
		return "ecrypt"
	case unix.EFIVARFS_MAGIC:
		return "efivar"
	case unix.EFS_SUPER_MAGIC:
		return "efs"
	case 0x137d:
		return "ext1"
	case 0xef51:
		return "ext2"
	case unix.EXT2_SUPER_MAGIC:
		// EXT3_SUPER_MAGIC and EXT4_SUPER_MAGIC have the same value
		return "ext"
	case unix.F2FS_SUPER_MAGIC:
		return "f2fs"
	case 0x4006:
		return "fat"
	case 0x19830326:
		return "fhgfs"
	case 0x65735546:
		return "fuse"
	case 0x65735543:
		return "fusectl"
	case unix.FUTEXFS_SUPER_MAGIC:
		return "futex"
	case 0x1161970:
		return "gfs"
	case 0x47504653:
		return "gpfs"
	case 0x4244:
		return "hfs"
	case unix.HOSTFS_SUPER_MAGIC:
		return "hostfs"
	case unix.HPFS_SUPER_MAGIC:
		return "hpfs"
	case unix.HUGETLBFS_MAGIC:
		return "hugetlb"
	case 0x2bad1dea:
		return "inotify"
	case unix.ISOFS_SUPER_MAGIC:
		return "isofs"
	case 0x4004:
		return "isofs_r_win"
	case 0x4000:
		return "isofs_win"
	case 0x07C0:
		return "jffs"
	case unix.JFFS2_SUPER_MAGIC:
		return "jffs2"
	case 0x3153464a:
		return "jfs"
	case 0x0bd00bd0:
		return "lustre"
	case unix.MINIX_SUPER_MAGIC, unix.MINIX_SUPER_MAGIC2:
		return "minix"
	case unix.MINIX2_SUPER_MAGIC, unix.MINIX2_SUPER_MAGIC2:
		return "minix2"
	case unix.MINIX3_SUPER_MAGIC:
		return "minix3"
	case 0x19800202:
		return "mqueue"
	case unix.MSDOS_SUPER_MAGIC:
		return "msdos"
	case unix.MTD_INODE_FS_MAGIC:
		return "mounted inode"
	case unix.NCP_SUPER_MAGIC:
		return "ncp"
	case unix.NFS_SUPER_MAGIC:
		return "nfs"
	case 0x6e667364:
		return "nfsd"
	case unix.NILFS_SUPER_MAGIC:
		return "nilfs"
	case unix.NSFS_MAGIC:
		return "nsfs"
	case 0x5346544e:
		return "ntfs"
	case unix.OCFS2_SUPER_MAGIC:
		return "ocfs2"
	case unix.OPENPROM_SUPER_MAGIC:
		return "openprom"
	case unix.OVERLAYFS_SUPER_MAGIC:
		return "overlayfs"
	case 0xaad7aaea:
		return "panfs"
	case unix.PIPEFS_MAGIC:
		return "pipefs"
	case unix.PROC_SUPER_MAGIC:
		return "proc"
	case 0x7c7c6673:
		return "prl_fs"
	case unix.PSTOREFS_MAGIC:
		return "pstore"
	case unix.QNX4_SUPER_MAGIC:
		return "qnx4"
	case unix.QNX6_SUPER_MAGIC:
		return "qnx6"
	case unix.RAMFS_MAGIC:
		return "ramfs"
	case unix.REISERFS_SUPER_MAGIC:
		return "reiserfs"
	case unix.RDTGROUP_SUPER_MAGIC:
		return "rdtgroup"
	case 0x7275:
		return "romfs"
	case 0x67596969:
		return "rpc_pipefs"
	case unix.SECURITYFS_MAGIC:
		return "securityfs"
	case unix.SELINUX_MAGIC:
		return "selinux"
	case unix.SMACK_MAGIC:
		return "smack"
	case unix.SMB_SUPER_MAGIC:
		return "smb"
	case unix.SOCKFS_MAGIC:
		return "sockfs"
	case unix.SQUASHFS_MAGIC:
		return "squashfs"
	case unix.SYSFS_MAGIC:
		return "sysfs"
	case 0x012ff7b6:
		return "sysv2"
	case 0x012ff7b5:
		return "sysv4"
	case unix.TMPFS_MAGIC:
		return "tmpfs"
	case unix.TRACEFS_MAGIC:
		return "tracefs"
	case unix.UDF_SUPER_MAGIC:
		return "udf"
	case 0x00011954, 0x54190100:
		return "ufs"
	case unix.USBDEVICE_SUPER_MAGIC:
		return "usb"
	case unix.V9FS_MAGIC:
		return "v9fs"
	case 0xbacbacbc:
		return "vmhgfs"
	case 0xa501fcf5:
		return "vxfs"
	case 0x565a4653:
		return "vzfs"
	case unix.XENFS_SUPER_MAGIC:
		return "xenfs"
	case 0x012ff7b4:
		return "xenix"
	case unix.XFS_SUPER_MAGIC:
		return "xfs"
	case 0x012fd16d:
		return "xiafs"
	case 0x2fc12fc1:
		return "zfs"
	case unix.ZSMALLOC_MAGIC:
		return "zsmalloc"
	default:
		return fmt.Sprintf("unknown (%#x)", fsType)
	}
}

func GetAvailableSpace(path string) (bytes, inodes uint64, fstype string, err error) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, 0, "", err
	}
	return fs.Bavail * uint64(fs.Bsize), fs.Ffree, FsTypeString(fs.Type), nil
}

// The minimum number of inodes available in the pipestance directory
// below which the pipestance will not run.
const PIPESTANCE_MIN_INODES uint64 = 500

// The minimum amount of available disk space for a pipestance directory.
// If the available space falls below this at any time during the run, the
// the pipestance is killed.
const PIPESTANCE_MIN_DISK uint64 = 50 * 1024 * 1024

type DiskSpaceError struct {
	Bytes   uint64
	Inodes  uint64
	Message string
}

func (self *DiskSpaceError) Error() string {
	return self.Message
}

var disableDiskSpaceCheck = (os.Getenv("MRO_DISK_SPACE_CHECK") == "disable")

// Returns an error if the current available space on the disk drive is
// very low.
func CheckMinimalSpace(path string) error {
	if disableDiskSpaceCheck {
		return nil
	}
	bytes, inodes, _, err := GetAvailableSpace(path)
	if err != nil {
		return err
	}
	// Allow zero, as if we haven't already failed to write a file it's
	// likely that the filesystem is just lying to us.
	if bytes < PIPESTANCE_MIN_DISK && bytes != 0 {
		return &DiskSpaceError{bytes, inodes, fmt.Sprintf(
			"%s has only %dkB remaining space available.\n"+
				"To ignore this error, set MRO_DISK_SPACE_CHECK=disable in your environment.",
			path, bytes/1024)}
	}
	if inodes < PIPESTANCE_MIN_INODES && inodes != 0 {
		return &DiskSpaceError{bytes, inodes, fmt.Sprintf(
			"%s has only %d free inodes remaining.\n"+
				"To ignore this error, set MRO_DISK_SPACE_CHECK=disable in your environment.",
			path, inodes)}
	}
	return nil
}

// GetMountOptions returns the mount type and options for the mount on
// which the given path exists.
func GetMountOptions(path string) (fstype, opts string, err error) {
	mountId := make([]byte, 0, 21)
	if info, err := os.Stat(path); err != nil || info == nil {
		return "", "", err
	} else if sysInfo, ok := info.Sys().(*syscall.Stat_t); !ok {
		return "", "", fmt.Errorf("Incorrect stat type %T", info.Sys())
	} else {
		itoa := func(i uint32) {
			if i == 0 {
				mountId = append(mountId, '0')
			} else {
				mountId = append(mountId, strconv.Itoa(int(i))...)
			}
		}
		itoa(unix.Major(sysInfo.Dev))
		mountId = append(mountId, ':')
		itoa(unix.Minor(sysInfo.Dev))
	}
	m, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", "", err
	}
	defer m.Close()
	// Abbreviated from the `proc` man page:
	//
	// The file contains lines of the form:
	//
	//  36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
	//  (1)(2)(3)   (4)   (5)      (6)      (7)   (8) (9)   (10)         (11)
	//
	//  (3)  major:minor: the value of st_dev for files on this filesystem (see stat(2)).
	//
	//  (6)  mount options: per-mount options.
	//
	//  (7)  optional fields: zero or more fields of the form "tag[:value]"; see below.
	//
	//  (8)  separator: the end of the optional fields is marked by a single hyphen.
	//
	//  (9)  filesystem type: the filesystem type in the form "type[.subtype]".
	//
	//  (10) mount source: filesystem-specific information or "none".
	//
	//  (11) super options: per-superblock options.
	scanner := bufio.NewScanner(m)
	for scanner.Scan() {
		fields := bytes.Fields(scanner.Bytes())
		if len(fields) >= 8 && bytes.Equal(mountId, fields[2]) {
			var fsType string
			opts := fields[5][0:len(fields[5]):len(fields[5])]
			for i, f := range fields[6:] {
				if len(f) == 1 && f[0] == '-' {
					if len(fields) >= i+7 {
						fsType = string(fields[i+7])
						if len(fields) >= i+9 && len(fields[i+9]) > 0 {
							opts = append(opts, ',')
							opts = append(opts, fields[i+9]...)
						}
					}
					return fsType, string(opts), nil
				}
			}
			return fsType, string(opts), nil
		}
	}
	return "", "", fmt.Errorf(
		"failed to find mount ID %s", string(mountId))
}
