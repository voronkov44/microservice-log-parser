package parser

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/voronkov44/microservice-log-parser/log-services/parser/core"
)

func (e *Engine) readSource(ctx context.Context, path string) ([]sourceFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("stat source: %w", err)
	}

	if info.IsDir() {
		return readDir(ctx, path)
	}

	name := strings.ToLower(info.Name())

	switch {
	case strings.HasSuffix(name, ".zip"):
		return readZip(ctx, path)
	case strings.HasSuffix(name, ".tar"):
		return readTar(ctx, path, false)
	case strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".tgz"):
		return readTar(ctx, path, true)
	case strings.HasSuffix(name, ".gz"):
		return readGzip(ctx, path)
	default:
		data, err := readSmallFile(path)
		if err != nil {
			return nil, err
		}

		return []sourceFile{{name: info.Name(), data: data}}, nil
	}
}

func readDir(ctx context.Context, root string) ([]sourceFile, error) {
	var files []sourceFile
	var limits archiveLimits

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat directory file %s: %w", path, err)
		}
		if err := limits.addFile(path, info.Size()); err != nil {
			return err
		}

		data, err := readSmallFile(path)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = filepath.Base(path)
		}

		files = append(files, sourceFile{
			name: rel,
			data: data,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	return files, nil
}

func readSmallFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("%w: file %s is too large", core.ErrBadArguments, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return data, nil
}

func readZip(ctx context.Context, path string) ([]sourceFile, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	files := make([]sourceFile, 0, len(reader.File))
	var limits archiveLimits

	for _, file := range reader.File {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if file.FileInfo().IsDir() {
			continue
		}

		if file.UncompressedSize64 > uint64(maxFileSize) {
			return nil, fmt.Errorf("%w: archive file %s is too large", core.ErrBadArguments, file.Name)
		}
		if err := limits.addFile(file.Name, int64(file.UncompressedSize64)); err != nil {
			return nil, err
		}

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip file %s: %w", file.Name, err)
		}

		data, err := io.ReadAll(io.LimitReader(rc, maxFileSize+1))
		closeErr := rc.Close()

		if err != nil {
			return nil, fmt.Errorf("read zip file %s: %w", file.Name, err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close zip file %s: %w", file.Name, closeErr)
		}
		if int64(len(data)) > maxFileSize {
			return nil, fmt.Errorf("%w: archive file %s is too large", core.ErrBadArguments, file.Name)
		}

		files = append(files, sourceFile{
			name: file.Name,
			data: data,
		})
	}

	return files, nil
}

func readTar(ctx context.Context, path string, gzipped bool) ([]sourceFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open tar: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var reader io.Reader = file

	if gzipped {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("open gzip tar: %w", err)
		}
		defer func() {
			_ = gzReader.Close()
		}()

		reader = gzReader
	}

	tarReader := tar.NewReader(reader)
	var files []sourceFile
	var limits archiveLimits

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, fmt.Errorf("read tar header: %w", err)
		}

		if header.FileInfo().IsDir() {
			continue
		}

		if err := limits.addFile(header.Name, header.Size); err != nil {
			return nil, err
		}

		data, err := io.ReadAll(io.LimitReader(tarReader, maxFileSize+1))
		if err != nil {
			return nil, fmt.Errorf("read tar file %s: %w", header.Name, err)
		}
		if int64(len(data)) > maxFileSize {
			return nil, fmt.Errorf("%w: archive file %s is too large", core.ErrBadArguments, header.Name)
		}

		files = append(files, sourceFile{
			name: header.Name,
			data: data,
		})
	}

	return files, nil
}

func readGzip(ctx context.Context, path string) ([]sourceFile, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("open gzip reader: %w", err)
	}
	defer func() {
		_ = gzReader.Close()
	}()

	data, err := io.ReadAll(io.LimitReader(gzReader, maxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}
	if int64(len(data)) > maxFileSize {
		return nil, fmt.Errorf("%w: gzip file is too large", core.ErrBadArguments)
	}

	name := strings.TrimSuffix(filepath.Base(path), ".gz")
	var limits archiveLimits
	if err := limits.addFile(name, int64(len(data))); err != nil {
		return nil, err
	}

	return []sourceFile{{name: name, data: data}}, nil
}

type archiveLimits struct {
	files int
	total int64
}

func (l *archiveLimits) addFile(name string, size int64) error {
	if size < 0 {
		size = 0
	}
	if size > maxFileSize {
		return fmt.Errorf("%w: archive file %s is too large", core.ErrBadArguments, name)
	}
	if l.files+1 > maxFiles {
		return fmt.Errorf("%w: archive contains too many files", core.ErrBadArguments)
	}
	if l.total+size > maxTotalBytes {
		return fmt.Errorf("%w: archive total size is too large", core.ErrBadArguments)
	}

	l.files++
	l.total += size

	return nil
}
