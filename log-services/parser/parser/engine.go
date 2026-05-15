package parser

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/voronkov44/microservice-log-parser/log-services/parser/core"
)

const (
	maxFileSize   int64 = 64 << 20 // 64 MiB
	maxFiles            = 1000
	maxTotalBytes int64 = 128 << 20 // 128 MiB
)

type Engine struct {
	dataDir string
	log     *slog.Logger
}

type sourceFile struct {
	name string
	data []byte
}

func New(dataDir string, log *slog.Logger) *Engine {
	return &Engine{
		dataDir: dataDir,
		log:     log,
	}
}

func (e *Engine) Parse(ctx context.Context, requestedPath string) (core.ParsedLog, error) {
	path, err := e.resolvePath(requestedPath)
	if err != nil {
		return core.ParsedLog{}, err
	}

	files, err := e.readSource(ctx, path)
	if err != nil {
		return core.ParsedLog{}, err
	}

	parsed, err := parseFiles(ctx, files)
	if err != nil {
		return core.ParsedLog{}, err
	}

	parsed = finalizeParsedLog(parsed)

	if len(parsed.Nodes) == 0 {
		return core.ParsedLog{}, fmt.Errorf("%w: no nodes found", core.ErrParse)
	}

	e.log.Info(
		"log parsed",
		"path", requestedPath,
		"files", len(files),
		"nodes", len(parsed.Nodes),
		"ports", len(parsed.Ports),
		"nodes_info", len(parsed.NodesInfo),
	)

	return parsed, nil
}

func (e *Engine) resolvePath(requestedPath string) (string, error) {
	if strings.TrimSpace(requestedPath) == "" {
		return "", core.ErrBadArguments
	}

	dataAbs, err := filepath.Abs(e.dataDir)
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}

	cleanRequested := filepath.Clean(requestedPath)

	var candidate string
	if filepath.IsAbs(cleanRequested) {
		// Docker case: request comes as /data/file.zip,
		// local data dir may be /project/data.
		if cleanRequested == "/data" || strings.HasPrefix(cleanRequested, "/data/") {
			rel := strings.TrimPrefix(cleanRequested, "/data")
			candidate = filepath.Join(dataAbs, rel)
		} else {
			candidate = cleanRequested
		}
	} else {
		candidate = filepath.Join(dataAbs, cleanRequested)
	}

	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve requested path: %w", err)
	}

	rel, err := filepath.Rel(dataAbs, candidateAbs)
	if err != nil {
		return "", fmt.Errorf("check requested path: %w", err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: path must be inside data dir", core.ErrBadArguments)
	}

	return candidateAbs, nil
}

func parseFiles(ctx context.Context, files []sourceFile) (core.ParsedLog, error) {
	var parsed core.ParsedLog

	for _, file := range files {
		select {
		case <-ctx.Done():
			return core.ParsedLog{}, ctx.Err()
		default:
		}

		if len(trimBytes(file.data)) == 0 {
			continue
		}

		fileParsed, err := parseFile(file)
		if err != nil {
			return core.ParsedLog{}, fmt.Errorf("parse file %s: %w", file.name, err)
		}

		parsed.Nodes = append(parsed.Nodes, fileParsed.Nodes...)
		parsed.Ports = append(parsed.Ports, fileParsed.Ports...)
		parsed.NodesInfo = append(parsed.NodesInfo, fileParsed.NodesInfo...)
	}

	return parsed, nil
}

func parseFile(file sourceFile) (core.ParsedLog, error) {
	name := strings.ToLower(file.name)

	switch {
	case strings.HasSuffix(name, ".db_csv") || strings.Contains(name, "db_csv"):
		return parseDBCSV(file)
	case strings.HasSuffix(name, ".csv"):
		return parseCSV(file)
	default:
		return parseKeyValueSections(file)
	}
}
