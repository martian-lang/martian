// Copyright (c) 2019 10X Genomics, Inc. All rights reserved.

// Code for querying cgroup information.

package util

import (
	"os"
	"path"
	"testing"
)

func Test_getMemoryCgroupPath(t *testing.T) {
	dir := t.TempDir()
	mountInfoPath = path.Join(dir, "mountinfo")
	cgroupProcPath = path.Join(dir, "cgroup")
	defer func() {
		mountInfoPath = "/proc/self/mountinfo"
		cgroupProcPath = "/proc/self/cgroup"
	}()
	t.Run("v2", func(t *testing.T) {
		if err := os.WriteFile(mountInfoPath, []byte(`
22 27 0:20 / /sys rw,nosuid,nodev,noexec,relatime shared:7 - sysfs sysfs rw
23 27 0:21 / /proc rw,nosuid,nodev,noexec,relatime shared:13 - proc proc rw
24 27 0:5 / /dev rw,nosuid,relatime shared:2 - devtmpfs udev rw,size=528k,nr_inodes=13206,mode=755,inode64
25 24 0:22 / /dev/pts rw,nosuid,noexec,relatime shared:3 - devpts devpts rw,gid=5,mode=620,ptmxmode=000
26 27 0:23 / /run rw,nosuid,nodev,noexec,relatime shared:5 - tmpfs tmpfs rw,size=105653748k,mode=755,inode64
27 1 254:0 / / rw,relatime shared:1 - ext4 /dev/mapper/thing--vg-root rw,errors=remount-ro
28 22 0:6 / /sys/kernel/security rw,nosuid,nodev,noexec,relatime shared:8 - securityfs securityfs rw
29 24 0:24 / /dev/shm rw,nosuid,nodev shared:4 - tmpfs tmpfs rw,inode64
30 26 0:25 / /run/lock rw,nosuid,nodev,noexec,relatime shared:6 - tmpfs tmpfs rw,size=5120k,inode64
31 22 0:26 / /sys/fs/cgroup rw,noexec,relatime shared:9 - cgroup2 cgroup2 rw,nsdelegate,memory_recursiveprot
32 22 0:27 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime shared:10 - pstore pstore rw
33 22 0:28 / /sys/firmware/efi/efivars rw,nosuid,nodev,noexec,relatime shared:11 - efivarfs efivarfs rw
34 22 0:29 / /sys/fs/bpf rw,nosuid,nodev,noexec,relatime shared:12 - bpf bpf rw,mode=700
35 23 0:30 / /proc/sys/fs/binfmt_misc rw,relatime shared:14 - autofs systemd-1 rw,fd=30,pgrp=1
36 24 0:19 / /dev/mqueue rw,nosuid,nodev,noexec,relatime shared:15 - mqueue mqueue rw
37 22 0:12 / /sys/kernel/tracing rw,nosuid,nodev,noexec,relatime shared:16 - tracefs tracefs rw
38 24 0:31 / /dev/hugepages rw,relatime shared:17 - hugetlbfs hugetlbfs rw,pagesize=2M
39 22 0:7 / /sys/kernel/debug rw,nosuid,nodev,noexec,relatime shared:18 - debugfs debugfs rw
40 22 0:32 / /sys/kernel/config rw,nosuid,nodev,noexec,relatime shared:19 - configfs configfs rw
62 26 0:33 / /run/credentials/systemd-sysctl.service ronoexec,relatime shared:20 - ramfs ramfs rw,mode=700
41 22 0:34 / /sys/fs/fuse/connections rw,nosuid,nodev,noexec,relatime shared:21 - fusectl fusectl rw
66 26 0:35 / /run/credentials/systemd-sysusers.service ro,relatime shared:22 - ramfs ramfs rw,mode=700
94 27 259:2 / /boot rw,relatime shared:33 - ext2 /dev/nvme0n1p2 rw,stripe=4
100 35 0:39 / /proc/sys/fs/binfmt_misc rw,nosuid,nodev,noexec,relatime shared:53 - binfmt_misc binfmt_misc rw
104 26 0:40 / /run/rpc_pipefs rw,relatime shared:55 - rpc_pipefs sunrpc rw
`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cgroupProcPath,
			[]byte("0::/system.slice/martian.service\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		p, v2 := getMemoryCgroupPath()
		if !v2 {
			t.Error("Expected v2")
		}
		if p != "/sys/fs/cgroup/system.slice/martian.service" {
			t.Errorf(
				"Expected /sys/fs/cgroup/system.slice/martian.service, got %q",
				p)
		}
	})

	t.Run("v1", func(t *testing.T) {
		if err := os.WriteFile(mountInfoPath, []byte(`
20 41 0:19 / /sys rw,nosuid,nodev,noexec,relatime shared:6 - sysfs sysfs rw,seclabel
21 41 0:20 / /proc rw,nosuid,nodev,noexec,relatime shared:5 - proc proc rw
22 41 0:5 / /dev rw,nosuid shared:2 - devtmpfs devtmpfs rw,size=392011804k,nr_inodes=98002951,mode=755
23 20 0:6 / /sys/kernel/security rw,nosuid,noexec,relatime shared:7 - securityfs securityfs rw
24 22 0:21 / /dev/shm rw,nosuid,nodev shared:3 - tmpfs tmpfs rw,seclabel
25 22 0:22 / /dev/pts rw,nosuid,noexec,relatime shared:4 - devpts devpts rw,seclabel,gid=5,mode=620
26 41 0:23 / /run rw,nosuid,nodev shared:22 - tmpfs tmpfs rw,seclabel,mode=755
27 20 0:24 / /sys/fs/cgroup ro,nosuid,nodev,noexec shared:8 - tmpfs tmpfs ro,seclabel,mode=755
28 27 0:25 / /sys/fs/cgroup/systemd rw,nosuid,nodev shared:9 - cgroup cgroup rw,seclabel,xattr,name=systemd
29 20 0:26 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime shared:20 - pstore pstore rw,seclabel
31 27 0:28 / /sys/fs/cgroup/cpuset rw,nosuid,nodev,relatime shared:11 - cgroup cgroup rw,seclabel,cpuset
32 27 0:29 / /sys/fs/cgroup/blkio rw,nosuid,nodev,noexec,relatime shared:12 - cgroup cgroup rw,seclabel,blkio
34 27 0:31 / /sys/fs/cgroup/hugetlb rw,nosuid,nodev,noexec shared:14 - cgroup cgroup rw,seclabel,hugetlb
35 27 0:32 / /sys/fs/cgroup/cpu,cpuacct rw,nodev,noexec shared:15 - cgroup cgroup rw,seclabel,cpu,cpuacct
36 27 0:33 / /sys/fs/cgroup/freezer rw,nodev,noexec,relatime shared:16 - cgroup cgroup rw,seclabel,freezer
37 27 0:34 / /sys/fs/cgroup/memory rw,nodev,noexec,relatime shared:17 - cgroup cgroup rw,seclabel,memory
38 27 0:35 / /sys/fs/cgroup/pids rw,nosuid,nodev,noexec,relatime shared:18 - cgroup cgroup rw,seclabel,pids
39 27 0:36 / /sys/fs/cgroup/devices rw,nosuid,nodev,relatime shared:19 - cgroup cgroup rw,seclabel,devices
41 1 259:3 / / rw,noatime shared:1 - xfs /dev/nvme0n1p1 rw,seclabel,attr2,inode64,logbufs=8,noquota
43 20 0:18 / /sys/fs/selinux rw,relatime shared:21 - selinuxfs selinuxfs rw
42 21 0:38 / /proc/sys/fs/binfmt_misc rw,relatime shared:23 - autofs systemd-1 rw,direct,pipe_ino=14755
`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cgroupProcPath, []byte(`11:devices:/
10:pids:/
9:memory:/
8:freezer:/
7:cpu,cpuacct:/
6:hugetlb:/
5:net_cls,net_prio:/
4:blkio:/
3:cpuset:/
2:perf_event:/
1:name=systemd:/system.slice/martian.service`), 0o644); err != nil {
			t.Fatal(err)
		}
		p, v2 := getMemoryCgroupPath()
		if v2 {
			t.Error("Expected v1")
		}
		if p != "/sys/fs/cgroup/memory/" {
			t.Errorf(
				"Expected /sys/fs/cgroup/memory/, got %q",
				p)
		}
	})
	t.Run("v1_subpath", func(t *testing.T) {
		if err := os.WriteFile(mountInfoPath, []byte(`
20 41 0:19 / /sys rw,nosuid,nodev,noexec,relatime shared:6 - sysfs sysfs rw,seclabel
21 41 0:20 / /proc rw,nosuid,nodev,noexec,relatime shared:5 - proc proc rw
22 41 0:5 / /dev rw,nosuid shared:2 - devtmpfs devtmpfs rw,size=392011804k,nr_inodes=98002951,mode=755
23 20 0:6 / /sys/kernel/security rw,nosuid,noexec,relatime shared:7 - securityfs securityfs rw
24 22 0:21 / /dev/shm rw,nosuid,nodev shared:3 - tmpfs tmpfs rw,seclabel
25 22 0:22 / /dev/pts rw,nosuid,noexec,relatime shared:4 - devpts devpts rw,seclabel,gid=5,mode=620
26 41 0:23 / /run rw,nosuid,nodev shared:22 - tmpfs tmpfs rw,seclabel,mode=755
27 20 0:24 / /sys/fs/cgroup ro,nosuid,nodev,noexec shared:8 - tmpfs tmpfs ro,seclabel,mode=755
28 27 0:25 / /sys/fs/cgroup/systemd rw,nosuid,nodev shared:9 - cgroup cgroup rw,seclabel,xattr,name=systemd
29 20 0:26 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime shared:20 - pstore pstore rw,seclabel
31 27 0:28 / /sys/fs/cgroup/cpuset rw,nosuid,nodev,relatime shared:11 - cgroup cgroup rw,seclabel,cpuset
32 27 0:29 / /sys/fs/cgroup/blkio rw,nosuid,nodev,noexec,relatime shared:12 - cgroup cgroup rw,seclabel,blkio
34 27 0:31 / /sys/fs/cgroup/hugetlb rw,nosuid,nodev,noexec shared:14 - cgroup cgroup rw,seclabel,hugetlb
35 27 0:32 / /sys/fs/cgroup/cpu,cpuacct rw,nodev,noexec shared:15 - cgroup cgroup rw,seclabel,cpu,cpuacct
36 27 0:33 / /sys/fs/cgroup/freezer rw,nodev,noexec,relatime shared:16 - cgroup cgroup rw,seclabel,freezer
37 27 0:34 / /sys/fs/cgroup/memory rw,nodev,noexec,relatime shared:17 - cgroup cgroup rw,seclabel,memory
38 27 0:35 / /sys/fs/cgroup/pids rw,nosuid,nodev,noexec,relatime shared:18 - cgroup cgroup rw,seclabel,pids
39 27 0:36 / /sys/fs/cgroup/devices rw,nosuid,nodev,relatime shared:19 - cgroup cgroup rw,seclabel,devices
41 1 259:3 / / rw,noatime shared:1 - xfs /dev/nvme0n1p1 rw,seclabel,attr2,inode64,logbufs=8,noquota
43 20 0:18 / /sys/fs/selinux rw,relatime shared:21 - selinuxfs selinuxfs rw
42 21 0:38 / /proc/sys/fs/binfmt_misc rw,relatime shared:23 - autofs systemd-1 rw,direct,pipe_ino=14755
`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cgroupProcPath, []byte(`11:devices:/
10:pids:/
9:memory:/mrp
8:freezer:/
7:cpu,cpuacct:/
6:hugetlb:/
5:net_cls,net_prio:/
4:blkio:/
3:cpuset:/
2:perf_event:/
1:name=systemd:/system.slice/martian.service`), 0o644); err != nil {
			t.Fatal(err)
		}
		p, v2 := getMemoryCgroupPath()
		if v2 {
			t.Error("Expected v1")
		}
		if p != "/sys/fs/cgroup/memory/mrp" {
			t.Errorf(
				"Expected /sys/fs/cgroup/memory/mrp, got %q",
				p)
		}
	})
}
