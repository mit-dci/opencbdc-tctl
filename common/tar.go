package common

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TarExtract extracts a tar(.gz) archive with full folder hierarchy into a directory
// with the name of the archive (minus the extension)
func TarExtract(path string) error {
	return TarExtractFlat(path, false, false)
}

// TarExtractFlat extracts a tar(.gz) archive with optionally (parameter flat)
// flattened directory structure, and optionally (parameter createNoDir)
// extracting alongside the archive in stead of making a folder
func TarExtractFlat(path string, flat bool, createNoDir bool) error {
	var err error

	basePath := path[0 : len(path)-7]

	var uncompressedStream io.ReadCloser

	if strings.HasSuffix(path, ".gz") {
		gzipStream, err := os.Open(path)
		if err != nil {
			return err
		}
		defer gzipStream.Close()

		uncompressedStream, err = gzip.NewReader(gzipStream)
		if err != nil {
			return err
		}
	} else {
		basePath = path[0 : len(path)-4]

		uncompressedStream, err = os.Open(path)
		if err != nil {
			return err
		}
	}
	if createNoDir {
		basePath = filepath.Dir(path)
	}

	if _, err = os.Stat(basePath); os.IsNotExist(err) {
		err = os.MkdirAll(basePath, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	}

	defer uncompressedStream.Close()
	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("ExtractTar: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if !flat {
				err = os.MkdirAll(filepath.Join(basePath, header.Name), 0755)
				if err != nil && !errors.Is(err, os.ErrExist) {
					return err
				}
			}
		case tar.TypeReg:
			targetPath := filepath.Join(basePath, header.Name)
			if flat {
				targetFile := header.Name
				lastSlash := strings.LastIndex(targetFile, "/")
				if lastSlash > -1 {
					targetFile = targetFile[lastSlash+1:]
				}
				targetPath = filepath.Join(basePath, targetFile)
			} else {
				err = os.MkdirAll(filepath.Dir(targetPath), 0755)
				if err != nil && !errors.Is(err, os.ErrExist) {
					return err
				}
			}
			outFile, err := os.OpenFile(
				targetPath,
				os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				os.FileMode(header.Mode),
			)
			if err != nil {
				return fmt.Errorf(
					"ExtractTarGz: Create() failed: %s",
					err.Error(),
				)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf(
					"ExtractTarGz: Copy() failed: %s",
					err.Error(),
				)
			}
			outFile.Close()
		default:
			return fmt.Errorf(
				"ExtractTarGz: uknown type: %x in %s",
				header.Typeflag,
				header.Name)
		}

	}
	return nil
}

// tarAddFile adds a file to a tar archive
func tarAddFile(tw *tar.Writer, path, relativePath string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if stat, err := file.Stat(); err == nil {
		// now lets create the header as needed for this file within the tarball
		header := new(tar.Header)
		header.Name = relativePath
		header.Size = stat.Size()
		header.Mode = int64(stat.Mode())
		header.ModTime = stat.ModTime()
		// write the header to the tarball archive
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// copy the file data to the tarball
		if _, err := io.Copy(tw, file); err != nil {
			return err
		}
	}
	return nil
}

// CreateArchive creates a TAR.GZ archive of sourceFolder and writes it to
// the target path
func CreateArchive(sourceFolder, targetPath string) error {
	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return CreateArchiveToStream(sourceFolder, file)
}

// CreateArchiveToStream creates a TAR.GZ archive of sourceFolder and writes it
// to the target stream
func CreateArchiveToStream(sourceFolder string, target io.Writer) error {
	gw := gzip.NewWriter(target)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	return filepath.Walk(sourceFolder,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				if strings.Contains(path, "/.git/") {
					return nil
				}
				relativePath, err := filepath.Rel(sourceFolder, path)
				if err != nil {
					return err
				}
				return tarAddFile(tw, path, relativePath)
			}
			return nil
		})

}
