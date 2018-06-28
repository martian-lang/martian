package api

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/martian-lang/martian/martian/core"
	"github.com/martian-lang/martian/martian/util"
)

// Lists of top-level files and files in "extras"
type FilesListing struct {
	Files  []string `json:"files,omitempty"`
	Extras []string `json:"extras,omitempty"`
}

func fileOnWhitelist(f core.MetadataFileName) bool {
	switch f {
	case "filelist", "sitecheck",
		core.AlarmFile, core.Assert, core.Errors,
		core.LogFile, core.TagsFile:
		return true
	default:
		return false
	}
}

func GetFilesListing(psdir string) (*FilesListing, error) {
	if allfiles, err := util.Readdirnames(psdir); err != nil {
		return nil, err
	} else {
		files := make([]string, 0, len(allfiles))
		for _, f := range allfiles {
			if len(f) > len(core.MetadataFilePrefix) && strings.HasPrefix(f, core.MetadataFilePrefix) {
				if mdf := core.MetadataFileName(strings.TrimPrefix(f, core.MetadataFilePrefix)); fileOnWhitelist(mdf) {
					files = append(files, string(mdf))
				}
			}
		}
		var result FilesListing
		if len(files) > 0 {
			result.Files = files
			sort.Strings(result.Files)
		}
		if files, err := util.Readdirnames(filepath.Join(psdir, "extras")); err == nil && len(files) > 0 {
			sort.Strings(files)
			result.Extras = files
		}
		return &result, nil
	}
}
