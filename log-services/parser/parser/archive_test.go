package parser

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/voronkov44/microservice-log-parser/log-services/parser/core"
)

func TestReadSourceReadsSupportedSources(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	files := map[string]string{
		"nodes.csv": "node_guid,node_desc,node_type\nnode-1,host-a,1\n",
		"ports.csv": "node_guid,port_guid,port_num,port_state\nnode-1,port-1,1,4\n",
	}

	sourceDir := filepath.Join(dir, "source-dir")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(sourceDir, name), []byte(data), 0o644); err != nil {
			t.Fatalf("write source file: %v", err)
		}
	}

	zipPath := filepath.Join(dir, "source.zip")
	writeZip(t, zipPath, files)

	tarPath := filepath.Join(dir, "source.tar")
	writeTar(t, tarPath, files, false)

	tarGzipPath := filepath.Join(dir, "source.tar.gz")
	writeTar(t, tarGzipPath, files, true)

	gzipPath := filepath.Join(dir, "nodes.csv.gz")
	writeGzip(t, gzipPath, files["nodes.csv"])

	plainPath := filepath.Join(dir, "plain.log")
	if err := os.WriteFile(plainPath, []byte("node_guid=node-plain\n"), 0o644); err != nil {
		t.Fatalf("write plain file: %v", err)
	}

	engine := New(dir, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	tests := []struct {
		name      string
		path      string
		wantFiles int
		wantName  string
	}{
		{name: "dir", path: sourceDir, wantFiles: 2, wantName: "nodes.csv"},
		{name: "zip", path: zipPath, wantFiles: 2, wantName: "nodes.csv"},
		{name: "tar", path: tarPath, wantFiles: 2, wantName: "nodes.csv"},
		{name: "tar gzip", path: tarGzipPath, wantFiles: 2, wantName: "nodes.csv"},
		{name: "gzip", path: gzipPath, wantFiles: 1, wantName: "nodes.csv"},
		{name: "plain", path: plainPath, wantFiles: 1, wantName: "plain.log"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.readSource(ctx, tt.path)
			if err != nil {
				t.Fatalf("readSource() error = %v", err)
			}

			if len(got) != tt.wantFiles {
				t.Fatalf("readSource() returned %d files, want %d", len(got), tt.wantFiles)
			}

			if got[0].name != tt.wantName {
				t.Fatalf("first file name = %q, want %q", got[0].name, tt.wantName)
			}

			if len(got[0].data) == 0 {
				t.Fatalf("first file data is empty")
			}
		})
	}
}

func TestReadSourceErrors(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	engine := New(dir, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	t.Run("missing file", func(t *testing.T) {
		_, err := engine.readSource(ctx, filepath.Join(dir, "missing.zip"))
		if !errors.Is(err, core.ErrNotFound) {
			t.Fatalf("readSource() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("corrupt zip", func(t *testing.T) {
		path := filepath.Join(dir, "bad.zip")
		if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
			t.Fatalf("write bad zip: %v", err)
		}

		_, err := engine.readSource(ctx, path)
		if err == nil {
			t.Fatalf("readSource() error is nil, want error")
		}
	})

	t.Run("too large plain file", func(t *testing.T) {
		path := filepath.Join(dir, "huge.log")
		file, err := os.Create(path)
		if err != nil {
			t.Fatalf("create huge file: %v", err)
		}
		if err := file.Truncate(maxFileSize + 1); err != nil {
			_ = file.Close()
			t.Fatalf("truncate huge file: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close huge file: %v", err)
		}

		_, err = engine.readSource(ctx, path)
		if !errors.Is(err, core.ErrBadArguments) {
			t.Fatalf("readSource() error = %v, want ErrBadArguments", err)
		}
	})

	t.Run("too many archive files", func(t *testing.T) {
		path := filepath.Join(dir, "too-many.zip")
		files := make(map[string]string, maxFiles+1)
		for i := 0; i <= maxFiles; i++ {
			files[filepath.Join("logs", strconv.Itoa(i)+".log")] = "node_guid=node\n"
		}
		writeZip(t, path, files)

		_, err := engine.readSource(ctx, path)
		if !errors.Is(err, core.ErrBadArguments) {
			t.Fatalf("readSource() error = %v, want ErrBadArguments", err)
		}
	})
}

func TestArchiveLimits(t *testing.T) {
	t.Run("too large single file", func(t *testing.T) {
		var limits archiveLimits
		if err := limits.addFile("huge.log", maxFileSize+1); !errors.Is(err, core.ErrBadArguments) {
			t.Fatalf("addFile() error = %v, want ErrBadArguments", err)
		}
	})

	t.Run("too large total size", func(t *testing.T) {
		var limits archiveLimits
		fileSize := maxTotalBytes / 2

		if err := limits.addFile("one.log", fileSize); err != nil {
			t.Fatalf("addFile(one) error = %v", err)
		}
		if err := limits.addFile("two.log", fileSize); err != nil {
			t.Fatalf("addFile(two) error = %v", err)
		}
		if err := limits.addFile("three.log", 1); !errors.Is(err, core.ErrBadArguments) {
			t.Fatalf("addFile(three) error = %v, want ErrBadArguments", err)
		}
	})
}

func TestEngineParseArchive(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "log.zip")
	writeZip(t, zipPath, map[string]string{
		"nodes.csv":      "node_guid,node_desc,node_type,node_kind,num_ports,class_version,base_version,system_image_guid,port_guid\nhost-1,host alpha,1,host,2,1,2,sys-host,hca-port\nswitch-1,switch beta,2,switch,36,3,4,sys-switch,sw-port\n",
		"ports.csv":      "node_guid,port_guid,port_num,lid,local_port_num,port_state,port_phy_state,link_width_active,link_speed_active\nhost-1,hp-1,1,101,1,4,5,4,100\nswitch-1,sp-1,1,201,1,4,5,4,100\n",
		"nodes_info.csv": "node_guid,serial_number,part_number,revision,product_name\nhost-1,SN-host,PN-host,A1,Host Product\n",
	})

	engine := New(dir, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	parsed, err := engine.Parse(context.Background(), "log.zip")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(parsed.Nodes) != 2 {
		t.Fatalf("nodes count = %d, want 2", len(parsed.Nodes))
	}
	if len(parsed.Ports) != 2 {
		t.Fatalf("ports count = %d, want 2", len(parsed.Ports))
	}
	if len(parsed.NodesInfo) != 1 {
		t.Fatalf("nodes info count = %d, want 1", len(parsed.NodesInfo))
	}

	host := parsed.Nodes[0]
	if host.NodeGUID != "host-1" || host.NodeKind != "host" || host.NumPorts != 2 {
		t.Fatalf("host parsed incorrectly: %+v", host)
	}
	if host.ClassVersion != 1 || host.BaseVersion != 2 || host.SystemImageGUID != "sys-host" {
		t.Fatalf("host numeric/system fields parsed incorrectly: %+v", host)
	}

	port := parsed.Ports[0]
	if port.NodeGUID != "host-1" || port.PortGUID != "hp-1" || port.PortNum != 1 {
		t.Fatalf("port parsed incorrectly: %+v", port)
	}
	if port.LID != 101 || port.PortState != 4 || port.LinkWidthActive != 4 || port.LinkSpeedActive != 100 {
		t.Fatalf("port numeric fields parsed incorrectly: %+v", port)
	}

	info := parsed.NodesInfo[0]
	if info.NodeGUID != "host-1" || info.SerialNumber != "SN-host" || info.ProductName != "Host Product" {
		t.Fatalf("node info parsed incorrectly: %+v", info)
	}
}

func TestEngineParseIncompleteObjectsCreatesUnknownNode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "ports.csv"),
		[]byte("node_guid,port_guid,port_num,port_state\nport-only-node,p1,1,4\n"),
		0o644,
	); err != nil {
		t.Fatalf("write ports file: %v", err)
	}

	engine := New(dir, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	parsed, err := engine.Parse(context.Background(), "ports.csv")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(parsed.Nodes) != 1 {
		t.Fatalf("nodes count = %d, want 1", len(parsed.Nodes))
	}
	if parsed.Nodes[0].NodeGUID != "port-only-node" || parsed.Nodes[0].NodeKind != "unknown" {
		t.Fatalf("created node = %+v, want unknown node for port-only-node", parsed.Nodes[0])
	}
}

func TestEngineParseStrictValidation(t *testing.T) {
	dir := t.TempDir()
	engine := New(dir, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	t.Run("invalid numeric value", func(t *testing.T) {
		if err := os.WriteFile(
			filepath.Join(dir, "bad-numeric.csv"),
			[]byte("node_guid,node_desc,node_type\nnode-1,host,not-a-number\n"),
			0o644,
		); err != nil {
			t.Fatalf("write bad numeric file: %v", err)
		}

		_, err := engine.Parse(context.Background(), "bad-numeric.csv")
		if !errors.Is(err, core.ErrParse) {
			t.Fatalf("Parse() error = %v, want ErrParse", err)
		}
	})

	t.Run("malformed key-value section", func(t *testing.T) {
		if err := os.WriteFile(
			filepath.Join(dir, "bad-section.log"),
			[]byte("node_guid=node-1\nthis line is malformed\n"),
			0o644,
		); err != nil {
			t.Fatalf("write bad section file: %v", err)
		}

		_, err := engine.Parse(context.Background(), "bad-section.log")
		if !errors.Is(err, core.ErrParse) {
			t.Fatalf("Parse() error = %v, want ErrParse", err)
		}
	})

	t.Run("missing optional numeric fields", func(t *testing.T) {
		if err := os.WriteFile(
			filepath.Join(dir, "missing-optional.csv"),
			[]byte("node_guid,node_desc\nnode-1,host\n"),
			0o644,
		); err != nil {
			t.Fatalf("write missing optional file: %v", err)
		}

		parsed, err := engine.Parse(context.Background(), "missing-optional.csv")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if len(parsed.Nodes) != 1 || parsed.Nodes[0].NodeGUID != "node-1" {
			t.Fatalf("parsed = %+v", parsed)
		}
	})
}

func TestEngineParseReturnsExpectedErrors(t *testing.T) {
	dir := t.TempDir()
	engine := New(dir, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	t.Run("empty path", func(t *testing.T) {
		_, err := engine.Parse(context.Background(), " ")
		if !errors.Is(err, core.ErrBadArguments) {
			t.Fatalf("Parse() error = %v, want ErrBadArguments", err)
		}
	})

	t.Run("path outside data dir", func(t *testing.T) {
		_, err := engine.Parse(context.Background(), "../outside.log")
		if !errors.Is(err, core.ErrBadArguments) {
			t.Fatalf("Parse() error = %v, want ErrBadArguments", err)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := engine.Parse(context.Background(), "missing.zip")
		if !errors.Is(err, core.ErrNotFound) {
			t.Fatalf("Parse() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("empty archive", func(t *testing.T) {
		path := filepath.Join(dir, "empty.zip")
		writeZip(t, path, nil)

		_, err := engine.Parse(context.Background(), "empty.zip")
		if !errors.Is(err, core.ErrParse) {
			t.Fatalf("Parse() error = %v, want ErrParse", err)
		}
	})
}

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()

	out, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			t.Fatalf("close zip file: %v", err)
		}
	}()

	zw := zip.NewWriter(out)
	defer func() {
		if err := zw.Close(); err != nil {
			t.Fatalf("close zip writer: %v", err)
		}
	}()

	for _, name := range sortedFileNames(files) {
		data := files[name]
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := writer.Write([]byte(data)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
}

func writeTar(t *testing.T, path string, files map[string]string, gzipped bool) {
	t.Helper()

	out, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tar: %v", err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			t.Fatalf("close tar file: %v", err)
		}
	}()

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, name := range sortedFileNames(files) {
		data := files[name]
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(data)),
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(data)); err != nil {
			t.Fatalf("write tar entry: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	if gzipped {
		gw := gzip.NewWriter(out)
		if _, err := gw.Write(buf.Bytes()); err != nil {
			_ = gw.Close()
			t.Fatalf("write gzip tar: %v", err)
		}
		if err := gw.Close(); err != nil {
			t.Fatalf("close gzip tar: %v", err)
		}
		return
	}

	if _, err := out.Write(buf.Bytes()); err != nil {
		t.Fatalf("write tar: %v", err)
	}
}

func writeGzip(t *testing.T, path string, data string) {
	t.Helper()

	out, err := os.Create(path)
	if err != nil {
		t.Fatalf("create gzip: %v", err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			t.Fatalf("close gzip file: %v", err)
		}
	}()

	gw := gzip.NewWriter(out)
	if _, err := gw.Write([]byte(data)); err != nil {
		_ = gw.Close()
		t.Fatalf("write gzip: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
}

func sortedFileNames(files map[string]string) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}
