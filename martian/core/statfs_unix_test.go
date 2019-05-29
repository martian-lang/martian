package core

import (
	"os"
	"strings"
	"testing"
)

// Test that GetAvailableSpace and GetMountOptions agree on the filesystem type.
func TestFsType(t *testing.T) {
	p, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	mountType, _, err := GetMountOptions(p)
	if err != nil {
		t.Errorf("Failed to parse mountinfo for %s: %v", p, err)
	}
	if mountType == "" {
		t.Error("parsing mountinfo did not yield a filesystem type")
	}
	_, _, statfsType, err := GetAvailableSpace(p)
	if err != nil {
		t.Errorf("Failed to get type from statfs: %v", err)
	}
	if mountType != statfsType && !strings.HasPrefix(mountType, statfsType) {
		t.Errorf("%q != %q", mountType, statfsType)
	}
}

func TestGetMountOptions(t *testing.T) {
	p, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	_, opts, err := GetMountOptions(p)
	if err != nil {
		t.Errorf("Failed to parse mountinfo for %s: %v", p, err)
	}
	if !strings.ContainsRune(opts, ',') {
		t.Errorf("Unexpected mount options %q", opts)
	}
}
