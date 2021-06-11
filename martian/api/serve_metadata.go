// Copyright (c) 2021 10X Genomics, Inc. All rights reserved.

package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/martian-lang/martian/martian/core"
)

var unixEpochTime = time.Unix(0, 0)

// Serve a given metadata file over http.
//
// If the metadata file type is know, the content-type will be set
// appropriately.
//
// If the file supports the Stat method (e.g. os.File or the reader returned
// by zip.File.Open), the content-length and last-modified headers will also
// be set.
//
// If the reader is seekable (e.g. os.File) then http.ServeContent will be used,
// to support range requests.
func ServeMetadataFile(w http.ResponseWriter, req *http.Request,
	name string, data io.Reader) {
	if t := core.MetadataFileName(name).MimeType(); t != "" {
		w.Header().Set("Content-Type", t)
	}
	switch data := data.(type) {
	case interface {
		Stat() (os.FileInfo, error) // TODO(azarchs): Use fs.File, after go 1.15
	}:
		st, err := data.Stat()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else if errors.Is(err, os.ErrPermission) {
				http.Error(w, err.Error(), http.StatusForbidden)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		if seek, ok := data.(io.ReadSeeker); ok {
			http.ServeContent(w, req, name, st.ModTime(), seek)
			return
		} else if t := st.ModTime(); !t.IsZero() && !t.Equal(unixEpochTime) {
			w.Header().Set("Last-Modified", t.UTC().Format(http.TimeFormat))
		}
		w.Header().Set("Content-Length", strconv.FormatInt(st.Size(), 10))
	}
	if _, err := io.Copy(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
