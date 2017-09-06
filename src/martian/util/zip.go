//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// Zip archive utilities.

package util

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

// Find a file in the zip.  Follows symlinks.
func findFileInZip(zr *zip.ReadCloser, filePath string) *zip.File {
	for _, f := range zr.File {
		if f.Name == filePath {
			if f.Mode()&os.ModeSymlink != 0 {
				if reader, err := f.Open(); err != nil {
					if symbytes, err := ioutil.ReadAll(reader); err != nil {
						reader.Close()
						return findFileInZip(zr, path.Clean(
							path.Join(path.Dir(filePath), string(symbytes))))
					} else {
						reader.Close()
						return f
					}
				} else {
					return f
				}
			} else {
				return f
			}
		}
	}
	return nil
}

func ReadZip(zipPath string, filePath string) (string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	if f := findFileInZip(zr, filePath); f != nil {
		in, err := f.Open()
		if err != nil {
			return "", err
		}
		defer in.Close()

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, in); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	return "", &ZipError{zipPath, filePath}
}

func unzipLink(filePath string, f *zip.File) error {
	MkdirAll(path.Dir(filePath))

	in, err := f.Open()
	if err != nil {
		return err
	}
	defer in.Close()

	if symbytes, err := ioutil.ReadAll(in); err == nil {
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
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	type deferredLink struct {
		filePath string
		file     *zip.File
	}
	links := make([]deferredLink, 0, len(zr.File))
	for _, f := range zr.File {
		filePath := path.Join(path.Dir(zipPath), f.Name)
		if f.Mode()&os.ModeSymlink != 0 {
			links = append(links, deferredLink{filePath: filePath, file: f})
		} else {
			if err := unzipFile(filePath, f); err != nil {
				return err
			}
		}
	}
	for _, link := range links {
		if err := unzipLink(link.filePath, link.file); err != nil && !os.IsNotExist(err) {
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
		out, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			if link, err := os.Readlink(filePath); err != nil {
				return err
			} else {
				out.Write([]byte(link))
			}
		} else {
			if err := addZipFile(filePath, out); err != nil {
				return err
			}
		}
	}
	return zw.Close()
}
