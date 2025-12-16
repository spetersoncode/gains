package tool

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ai "github.com/spetersoncode/gains"
)

// FileToolOption configures file tools.
type FileToolOption func(*fileToolConfig)

type fileToolConfig struct {
	basePath          string
	allowedExtensions []string
	maxFileSize       int64
}

// WithBasePath restricts file operations to a specific directory.
// All paths will be resolved relative to this base path.
func WithBasePath(path string) FileToolOption {
	return func(c *fileToolConfig) {
		c.basePath = path
	}
}

// WithAllowedExtensions restricts file operations to specific file extensions.
func WithAllowedExtensions(exts ...string) FileToolOption {
	return func(c *fileToolConfig) {
		c.allowedExtensions = exts
	}
}

// WithMaxFileSize sets the maximum file size for read/write operations.
// Default is 10MB.
func WithMaxFileSize(bytes int64) FileToolOption {
	return func(c *fileToolConfig) {
		c.maxFileSize = bytes
	}
}

func applyFileOpts(opts []FileToolOption) *fileToolConfig {
	cfg := &fileToolConfig{
		maxFileSize: 10 * 1024 * 1024, // 10MB default
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func (c *fileToolConfig) resolvePath(path string) (string, error) {
	// Clean the path
	path = filepath.Clean(path)

	// If base path is set, resolve relative to it
	if c.basePath != "" {
		basePath := filepath.Clean(c.basePath)
		fullPath := filepath.Join(basePath, path)

		// Ensure the resolved path is still within the base path
		rel, err := filepath.Rel(basePath, fullPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("path %q is outside base path %q", path, basePath)
		}
		path = fullPath
	}

	return path, nil
}

func (c *fileToolConfig) checkExtension(path string) error {
	if len(c.allowedExtensions) == 0 {
		return nil
	}

	ext := filepath.Ext(path)
	for _, allowed := range c.allowedExtensions {
		if ext == allowed || ext == "."+allowed {
			return nil
		}
	}

	return fmt.Errorf("extension %q not allowed", ext)
}

// readFileArgs defines arguments for the read file tool.
type readFileArgs struct {
	Path     string `json:"path" desc:"Path to the file to read" required:"true"`
	Encoding string `json:"encoding" desc:"Output encoding" enum:"utf-8,base64"`
}

// NewReadFileTool creates a tool for reading file contents.
func NewReadFileTool(opts ...FileToolOption) (ai.Tool, Handler) {
	cfg := applyFileOpts(opts)

	schema := MustSchemaFor[readFileArgs]()

	t := ai.Tool{
		Name:        "read_file",
		Description: "Read the contents of a file",
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args readFileArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}

		path, err := cfg.resolvePath(args.Path)
		if err != nil {
			return "", err
		}

		if err := cfg.checkExtension(path); err != nil {
			return "", err
		}

		// Check file size
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		if info.Size() > cfg.maxFileSize {
			return "", fmt.Errorf("file size %d exceeds maximum %d", info.Size(), cfg.maxFileSize)
		}

		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()

		// Read with size limit
		content, err := io.ReadAll(io.LimitReader(f, cfg.maxFileSize))
		if err != nil {
			return "", err
		}

		if args.Encoding == "base64" {
			return base64.StdEncoding.EncodeToString(content), nil
		}

		return string(content), nil
	}

	return t, handler
}

// writeFileArgs defines arguments for the write file tool.
type writeFileArgs struct {
	Path    string `json:"path" desc:"Path to the file to write" required:"true"`
	Content string `json:"content" desc:"Content to write to the file" required:"true"`
	Mode    string `json:"mode" desc:"Write mode" enum:"overwrite,append"`
}

// NewWriteFileTool creates a tool for writing file contents.
func NewWriteFileTool(opts ...FileToolOption) (ai.Tool, Handler) {
	cfg := applyFileOpts(opts)

	schema := MustSchemaFor[writeFileArgs]()

	t := ai.Tool{
		Name:        "write_file",
		Description: "Write content to a file",
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args writeFileArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}

		path, err := cfg.resolvePath(args.Path)
		if err != nil {
			return "", err
		}

		if err := cfg.checkExtension(path); err != nil {
			return "", err
		}

		// Check content size
		if int64(len(args.Content)) > cfg.maxFileSize {
			return "", fmt.Errorf("content size %d exceeds maximum %d", len(args.Content), cfg.maxFileSize)
		}

		// Ensure parent directory exists
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}

		var flag int
		if args.Mode == "append" {
			flag = os.O_APPEND | os.O_CREATE | os.O_WRONLY
		} else {
			flag = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
		}

		f, err := os.OpenFile(path, flag, 0644)
		if err != nil {
			return "", err
		}
		defer f.Close()

		n, err := f.WriteString(args.Content)
		if err != nil {
			return "", err
		}

		result := struct {
			Path         string `json:"path"`
			BytesWritten int    `json:"bytes_written"`
			Mode         string `json:"mode"`
		}{
			Path:         path,
			BytesWritten: n,
			Mode:         args.Mode,
		}

		out, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	return t, handler
}

// listDirArgs defines arguments for the list directory tool.
type listDirArgs struct {
	Path      string `json:"path" desc:"Directory path to list" required:"true"`
	Recursive bool   `json:"recursive" desc:"Include subdirectories"`
}

// NewListDirTool creates a tool for listing directory contents.
func NewListDirTool(opts ...FileToolOption) (ai.Tool, Handler) {
	cfg := applyFileOpts(opts)

	schema := MustSchemaFor[listDirArgs]()

	t := ai.Tool{
		Name:        "list_directory",
		Description: "List the contents of a directory",
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args listDirArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}

		path, err := cfg.resolvePath(args.Path)
		if err != nil {
			return "", err
		}

		type entry struct {
			Name  string `json:"name"`
			Path  string `json:"path"`
			IsDir bool   `json:"is_dir"`
			Size  int64  `json:"size,omitempty"`
		}

		var entries []entry

		if args.Recursive {
			err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// Skip the root directory itself
				if p == path {
					return nil
				}

				relPath, _ := filepath.Rel(path, p)
				e := entry{
					Name:  info.Name(),
					Path:  relPath,
					IsDir: info.IsDir(),
				}
				if !info.IsDir() {
					e.Size = info.Size()
				}
				entries = append(entries, e)
				return nil
			})
		} else {
			dirEntries, err := os.ReadDir(path)
			if err != nil {
				return "", err
			}

			for _, de := range dirEntries {
				info, err := de.Info()
				if err != nil {
					continue
				}
				e := entry{
					Name:  de.Name(),
					Path:  de.Name(),
					IsDir: de.IsDir(),
				}
				if !de.IsDir() {
					e.Size = info.Size()
				}
				entries = append(entries, e)
			}
		}

		if err != nil {
			return "", err
		}

		result := struct {
			Path    string  `json:"path"`
			Count   int     `json:"count"`
			Entries []entry `json:"entries"`
		}{
			Path:    path,
			Count:   len(entries),
			Entries: entries,
		}

		out, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	return t, handler
}

// FileTools returns read, write, and list directory tools.
func FileTools(opts ...FileToolOption) []ToolPair {
	readTool, readHandler := NewReadFileTool(opts...)
	writeTool, writeHandler := NewWriteFileTool(opts...)
	listTool, listHandler := NewListDirTool(opts...)

	return []ToolPair{
		{Tool: readTool, Handler: readHandler},
		{Tool: writeTool, Handler: writeHandler},
		{Tool: listTool, Handler: listHandler},
	}
}
