//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Zip archive utilities.

package util

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime/trace"
	"strings"
	"time"
)

func readSymlinkInZip(f *zip.File) (string, error) {
	if reader, err := f.Open(); err != nil {
		return "", err
	} else {
		defer reader.Close()
		if symbytes, err := io.ReadAll(reader); err != nil {
			return "", err
		} else {
			return string(symbytes), nil
		}
	}
}

// Find a file in the zip.  Follows symlinks.
func findFileInZip(ctx context.Context, zr *zip.ReadCloser, filePath string) *zip.File {
	for _, f := range zr.File {
		if ctx.Err() != nil {
			return nil
		}
		if f.Mode()&os.ModeSymlink != 0 && strings.HasPrefix(filePath, f.Name+string(os.PathSeparator)) {
			if linkPath, err := readSymlinkInZip(f); err != nil {
				return nil
			} else {
				return findFileInZip(ctx, zr, path.Clean(
					path.Join(path.Dir(f.Name), linkPath,
						strings.TrimPrefix(filePath, f.Name+string(os.PathSeparator)))))
			}
		} else if f.Name == filePath {
			if f.Mode()&os.ModeSymlink != 0 {
				if linkPath, err := readSymlinkInZip(f); err != nil {
					return f
				} else {
					return findFileInZip(ctx, zr, path.Clean(
						path.Join(path.Dir(filePath), linkPath)))
				}
			} else {
				return f
			}
		}
	}
	return nil
}

// Wraps a file within a zip archive, along with the archive itself,
// as an io.ReadCloser
type zipFileReader struct {
	zr   *zip.ReadCloser
	file io.ReadCloser
	info zipFileInfo
}

func (zr *zipFileReader) ModTime() time.Time {
	return zr.info.ModTime()
}

func (zr *zipFileReader) Read(p []byte) (int, error) {
	return zr.file.Read(p)
}

func (zr *zipFileReader) Close() error {
	var err error
	if f := zr.file; f != nil {
		err = f.Close()
	}
	if z := zr.zr; z != nil {
		if ze := z.Close(); ze != nil {
			err = ze
		}
	}
	return err
}

// fs.FileInfo for a zipFileReader.
// Extracts just the bits we need so as to avoid holding pointers to the rest of
// the zip.FileHeader.
type zipFileInfo struct {
	modTime time.Time
	name    string
	size    int64
	mode    fs.FileMode
}

func extractFileInfo(fh *zip.FileHeader, raw bool) zipFileInfo {
	result := zipFileInfo{
		name:    fh.Name,
		mode:    fh.Mode(),
		modTime: fh.Modified,
	}
	if raw && fh.CompressedSize64 > 0 {
		result.size = int64(fh.CompressedSize64)
	} else {
		result.size = int64(fh.UncompressedSize64)
	}
	return result
}

func (fi zipFileInfo) Name() string       { return path.Base(fi.name) }
func (fi zipFileInfo) Size() int64        { return fi.size }
func (fi zipFileInfo) IsDir() bool        { return fi.Mode().IsDir() }
func (fi zipFileInfo) ModTime() time.Time { return fi.modTime }
func (fi zipFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi zipFileInfo) Type() fs.FileMode  { return fi.Mode().Type() }
func (fi zipFileInfo) Sys() any           { return &fi }

func (zr *zipFileReader) Stat() (fs.FileInfo, error) {
	return &zr.info, nil
}

// Opens a file within a zip archive for reading.
func ReadZipFile(zipPath, filePath string) (io.ReadCloser, error) {
	r, _, err := ReadZipFileRaw(context.TODO(), zipPath, filePath, "")
	return r, err
}

func openZipRaw(f *zip.File, accept string) (io.ReadCloser, string, error) {
	if m := acceptedEncoding(accept, f.Method); m != "" {
		r, err := f.OpenRaw()
		if err != nil {
			return nil, "", err
		}
		return io.NopCloser(r), m, err
	}
	r, err := f.Open()
	return r, "", err
}

// zipMethodToEncoding returns the http-header Content-Encoding name
// corresponding to the given zip method, or an empty string.
//
// Go's standard library only knows how to handle deflate, by default,
// but for completeness we're adding a few others here, since this code path
// is intended for use with ZipFile.OpenRaw() where we could in theory
// be sending compressed data to a web client and letting them deal with
// decompression.
func zipMethodToEncoding(method uint16) string {
	// For methods other than deflate, see zip APPNOTE.txt section 4.4.5
	switch method {
	case zip.Deflate:
		return "deflate"
	case 12:
		return "bz2"
	case 93:
		return "zstd"
	case 95:
		return "xz"
	}
	return ""
}

// acceptedEncoding returns the encoding name corresponding to the zip
// compression method, if it is present in accept.
func acceptedEncoding(accept string, method uint16) string {
	m := zipMethodToEncoding(method)
	// strings.Contains isn't a formally correct way to test whether the
	// encoding is accepted, however there's no case where an encoding
	// we support exists as a substring of a different valid encoding specifier,
	// (ignoring `;q=` modifiers, which we don't care about because we aren't
	// actually giving the user choices here) so this is correct in practice,
	// and more computationally efficient than a test that would be robust
	// against invalid specifiers in an `Accept-Encoding` header.
	if m != "" && !strings.Contains(accept, m) {
		m = ""
	}
	return m
}

// ReadZipRaw opens a file within an archive for reading, without decompressing.
//
// The second return argument specifies the encoding found within the file,
// generally either the empty string (uncompressed) or deflate.
func ReadZipFileRaw(ctx context.Context,
	zipPath, filePath,
	acceptEncoding string) (io.ReadCloser, string, error) {
	defer trace.StartRegion(ctx, "ReadZipFileRaw").End()
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, "", err
	}
	found := false
	defer func() {
		if !found {
			zr.Close()
		}
	}()

	if err := ctx.Err(); err != nil {
		return nil, "", err
	}

	if f := findFileInZip(ctx, zr, filePath); f != nil {
		in, m, err := openZipRaw(f, acceptEncoding)
		if err != nil {
			return nil, "", err
		}
		found = true
		return &zipFileReader{
			zr: zr, file: in,
			info: extractFileInfo(&f.FileHeader, m != ""),
		}, m, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}

	return nil, "", &ZipError{zipPath, filePath}
}

func ReadZip(zipPath string, filePath string) ([]byte, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	if f := findFileInZip(context.TODO(), zr, filePath); f != nil {
		in, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer in.Close()

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, in); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	return nil, &ZipError{zipPath, filePath}
}

func unzipLink(filePath string, f *zip.File) error {
	MkdirAll(path.Dir(filePath))

	in, err := f.Open()
	if err != nil {
		return err
	}
	defer in.Close()

	if symbytes, err := io.ReadAll(in); err == nil {
		return os.Symlink(string(symbytes), filePath)
	} else {
		return err
	}
}

func unzipFile(filePath string, f *zip.File) error {
	MkdirAll(path.Dir(filePath))

	in, err := f.Open()
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(filePath,
		os.O_CREATE|os.O_WRONLY|os.O_EXCL,
		f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func Unzip(zipPath string) error {
	return unzip(zipPath, false)
}

// UnzipIgnoreExisting unzips the given archive, skipping over files which
// already exist.
func UnzipIgnoreExisting(zipPath string) error {
	return unzip(zipPath, true)
}

func unzip(zipPath string, ignoreExisting bool) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	type deferredLink struct {
		file     *zip.File
		filePath string
	}
	links := make([]deferredLink, 0, len(zr.File))
	for _, f := range zr.File {
		filePath := path.Join(path.Dir(zipPath), f.Name)
		if f.Mode()&os.ModeSymlink != 0 {
			links = append(links, deferredLink{filePath: filePath, file: f})
		} else {
			if err := unzipFile(filePath, f); err != nil &&
				(!ignoreExisting || !os.IsExist(err)) {
				return err
			}
		}
	}
	for _, link := range links {
		if err := unzipLink(link.filePath, link.file); err != nil &&
			(!ignoreExisting || !os.IsExist(err)) {
			return err
		}
	}

	return nil
}

func addZipFile(filePath string, out io.Writer) error {
	in, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer in.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// isCompressedExtension returns true if the file path has an extension
// that typically implies compressed data.
//
// Typically CreateZip is used only for metadata files, none of which have these
// extensions, but checking just to be safe.
func isCompressedExtension(p string) bool {
	switch filepath.Ext(p) {
	case ".gz", ".png", ".jpg", "*.jpeg", "*.zip":
		return true
	}
	return false
}

func CreateZip(zipPath string, filePaths []string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, filePath := range filePaths {
		info, err := os.Lstat(filePath)
		if err != nil {
			return err
		}
		if info.IsDir() {
			continue
		}

		relPath, _ := filepath.Rel(path.Dir(zipPath), filePath)
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		// Turn on compression for files > 1kB.
		// For smaller files, the overhead of starting a deflate stream isn't
		// really worth the trouble.
		if info.Size() > 1024 && !isCompressedExtension(relPath) {
			header.Method = zip.Deflate
		}
		out, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			if link, err := os.Readlink(filePath); err != nil {
				return err
			} else if _, err := out.Write([]byte(link)); err != nil {
				return err
			}
		} else {
			if err := addZipFile(filePath, out); err != nil {
				return err
			}
		}
	}
	return zw.Close()
}
