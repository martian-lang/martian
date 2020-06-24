package core

import (
	"path"
	"testing"
	"testing/quick"
)

func TestMetadataFilePath(t *testing.T) {
	m := NewMetadata("blah", "/dev/null")
	if err := quick.CheckEqual(func(s string) string {
		return path.Join("/dev/null", MetadataFilePrefix+path.Base(s))
	}, func(s string) string {
		s = path.Join("/dev/null", MetadataFilePrefix+path.Base(s))
		return m.MetadataFilePath(metadataFileNameFromPath(s))
	}, nil); err != nil {
		t.Error(err)
	}
}
