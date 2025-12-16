package tool

import (
	"bufio"
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

// splitLines splits content into lines, handling different line endings.
func splitLines(content []byte) []string {
	if len(content) == 0 {
		return []string{}
	}
	text := string(content)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n")
}

// joinLines joins lines back together with newlines.
func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

// readLineRange reads specific lines from a reader.
// startLine and endLine are 1-based and inclusive.
// If startLine is nil, starts from line 1.
// If endLine is nil, reads to the end of file.
func readLineRange(r io.Reader, startLine, endLine *int, maxSize int64) ([]byte, error) {
	start := 1
	if startLine != nil {
		start = *startLine
	}
	if start < 1 {
		return nil, fmt.Errorf("start_line must be >= 1, got %d", start)
	}

	end := -1 // -1 means read to end
	if endLine != nil {
		end = *endLine
		if end < start {
			return nil, fmt.Errorf("end_line (%d) must be >= start_line (%d)", end, start)
		}
	}

	scanner := bufio.NewScanner(r)
	var result strings.Builder
	lineNum := 0
	bytesRead := int64(0)

	for scanner.Scan() {
		lineNum++

		// Skip lines before start
		if lineNum < start {
			continue
		}

		// Stop after end line
		if end > 0 && lineNum > end {
			break
		}

		line := scanner.Text()

		// Check size limit
		lineLen := int64(len(line) + 1) // +1 for newline
		if bytesRead+lineLen > maxSize {
			return nil, fmt.Errorf("line range content exceeds maximum size %d", maxSize)
		}
		bytesRead += lineLen

		if result.Len() > 0 {
			result.WriteByte('\n')
		}
		result.WriteString(line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Check if start line was beyond file length
	if lineNum < start {
		return nil, fmt.Errorf("start_line %d is beyond file length (%d lines)", start, lineNum)
	}

	return []byte(result.String()), nil
}

// readFileArgs defines arguments for the read file tool.
type readFileArgs struct {
	Path      string `json:"path" desc:"Path to the file to read" required:"true"`
	Encoding  string `json:"encoding" desc:"Output encoding" enum:"utf-8,base64"`
	StartLine *int   `json:"start_line" desc:"1-based line number to start reading from"`
	EndLine   *int   `json:"end_line" desc:"1-based line number to stop reading at (inclusive)"`
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

		var content []byte

		// Check if line range is specified
		if args.StartLine != nil || args.EndLine != nil {
			content, err = readLineRange(f, args.StartLine, args.EndLine, cfg.maxFileSize)
			if err != nil {
				return "", err
			}
		} else {
			// Read entire file (current behavior)
			content, err = io.ReadAll(io.LimitReader(f, cfg.maxFileSize))
			if err != nil {
				return "", err
			}
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

// editResultInfo provides details about what changed during an edit operation.
type editResultInfo struct {
	ReplacementsCount int `json:"replacements_count,omitempty"`
	LinesAffected     int `json:"lines_affected,omitempty"`
	LinesInserted     int `json:"lines_inserted,omitempty"`
	LinesDeleted      int `json:"lines_deleted,omitempty"`
}

// editFileArgs defines arguments for the edit file tool.
type editFileArgs struct {
	Path string `json:"path" desc:"Path to the file to edit" required:"true"`
	Mode string `json:"mode" desc:"Edit mode" enum:"replace_string,insert_lines,delete_lines,replace_lines" required:"true"`

	// String replacement mode fields
	Search     string `json:"search" desc:"String to search for (required for replace_string mode)"`
	Replace    string `json:"replace" desc:"String to replace with (for replace_string mode)"`
	ReplaceAll bool   `json:"replace_all" desc:"Replace all occurrences (default: first only)"`

	// Line operation mode fields
	StartLine *int   `json:"start_line" desc:"1-based starting line for line operations"`
	EndLine   *int   `json:"end_line" desc:"1-based ending line for delete_lines/replace_lines (inclusive)"`
	Content   string `json:"content" desc:"Content to insert or replace with (for insert_lines/replace_lines)"`
}

// replaceString performs string replacement in content.
func replaceString(content []byte, args editFileArgs) ([]byte, editResultInfo, error) {
	if args.Search == "" {
		return nil, editResultInfo{}, fmt.Errorf("search is required for replace_string mode")
	}

	text := string(content)
	var count int

	if args.ReplaceAll {
		count = strings.Count(text, args.Search)
		text = strings.ReplaceAll(text, args.Search, args.Replace)
	} else {
		if strings.Contains(text, args.Search) {
			count = 1
			text = strings.Replace(text, args.Search, args.Replace, 1)
		}
	}

	return []byte(text), editResultInfo{ReplacementsCount: count}, nil
}

// insertLines inserts content at a specific line position.
func insertLines(content []byte, args editFileArgs) ([]byte, editResultInfo, error) {
	if args.StartLine == nil {
		return nil, editResultInfo{}, fmt.Errorf("start_line is required for insert_lines mode")
	}

	lineNum := *args.StartLine
	if lineNum < 1 {
		return nil, editResultInfo{}, fmt.Errorf("start_line must be >= 1, got %d", lineNum)
	}

	lines := splitLines(content)

	// Allow inserting at position len(lines)+1 (append)
	if lineNum > len(lines)+1 {
		return nil, editResultInfo{}, fmt.Errorf("start_line %d is beyond file length (%d lines)", lineNum, len(lines))
	}

	insertContent := splitLines([]byte(args.Content))

	// Insert at position (0-indexed is lineNum-1)
	idx := lineNum - 1
	newLines := make([]string, 0, len(lines)+len(insertContent))
	newLines = append(newLines, lines[:idx]...)
	newLines = append(newLines, insertContent...)
	newLines = append(newLines, lines[idx:]...)

	return []byte(joinLines(newLines)), editResultInfo{LinesInserted: len(insertContent)}, nil
}

// deleteLines removes a range of lines from content.
func deleteLines(content []byte, args editFileArgs) ([]byte, editResultInfo, error) {
	if args.StartLine == nil {
		return nil, editResultInfo{}, fmt.Errorf("start_line is required for delete_lines mode")
	}

	startLine := *args.StartLine
	if startLine < 1 {
		return nil, editResultInfo{}, fmt.Errorf("start_line must be >= 1, got %d", startLine)
	}

	endLine := startLine // Default: delete single line
	if args.EndLine != nil {
		endLine = *args.EndLine
		if endLine < startLine {
			return nil, editResultInfo{}, fmt.Errorf("end_line (%d) must be >= start_line (%d)", endLine, startLine)
		}
	}

	lines := splitLines(content)

	if startLine > len(lines) {
		return nil, editResultInfo{}, fmt.Errorf("start_line %d is beyond file length (%d lines)", startLine, len(lines))
	}
	if endLine > len(lines) {
		endLine = len(lines) // Clamp to file end
	}

	// Delete lines (0-indexed)
	startIdx := startLine - 1
	endIdx := endLine // exclusive end for slice

	deletedCount := endIdx - startIdx
	newLines := make([]string, 0, len(lines)-deletedCount)
	newLines = append(newLines, lines[:startIdx]...)
	newLines = append(newLines, lines[endIdx:]...)

	return []byte(joinLines(newLines)), editResultInfo{LinesDeleted: deletedCount}, nil
}

// replaceLines replaces a range of lines with new content.
func replaceLines(content []byte, args editFileArgs) ([]byte, editResultInfo, error) {
	if args.StartLine == nil {
		return nil, editResultInfo{}, fmt.Errorf("start_line is required for replace_lines mode")
	}

	startLine := *args.StartLine
	if startLine < 1 {
		return nil, editResultInfo{}, fmt.Errorf("start_line must be >= 1, got %d", startLine)
	}

	endLine := startLine // Default: replace single line
	if args.EndLine != nil {
		endLine = *args.EndLine
		if endLine < startLine {
			return nil, editResultInfo{}, fmt.Errorf("end_line (%d) must be >= start_line (%d)", endLine, startLine)
		}
	}

	lines := splitLines(content)

	if startLine > len(lines) {
		return nil, editResultInfo{}, fmt.Errorf("start_line %d is beyond file length (%d lines)", startLine, len(lines))
	}
	if endLine > len(lines) {
		endLine = len(lines) // Clamp to file end
	}

	replacementLines := splitLines([]byte(args.Content))

	// Replace lines (0-indexed)
	startIdx := startLine - 1
	endIdx := endLine // exclusive end

	linesAffected := endIdx - startIdx
	newLines := make([]string, 0, len(lines)-linesAffected+len(replacementLines))
	newLines = append(newLines, lines[:startIdx]...)
	newLines = append(newLines, replacementLines...)
	newLines = append(newLines, lines[endIdx:]...)

	return []byte(joinLines(newLines)), editResultInfo{LinesAffected: linesAffected}, nil
}

// NewEditFileTool creates a tool for editing file contents.
// Supports string replacement and line operations (insert, delete, replace).
func NewEditFileTool(opts ...FileToolOption) (ai.Tool, Handler) {
	cfg := applyFileOpts(opts)

	schema := MustSchemaFor[editFileArgs]()

	t := ai.Tool{
		Name:        "edit_file",
		Description: "Edit a file using string replacement or line operations",
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args editFileArgs
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

		// Read current file content
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}

		if int64(len(content)) > cfg.maxFileSize {
			return "", fmt.Errorf("file size %d exceeds maximum %d", len(content), cfg.maxFileSize)
		}

		var newContent []byte
		var editResult editResultInfo

		switch args.Mode {
		case "replace_string":
			newContent, editResult, err = replaceString(content, args)
		case "insert_lines":
			newContent, editResult, err = insertLines(content, args)
		case "delete_lines":
			newContent, editResult, err = deleteLines(content, args)
		case "replace_lines":
			newContent, editResult, err = replaceLines(content, args)
		default:
			return "", fmt.Errorf("unknown edit mode: %s", args.Mode)
		}

		if err != nil {
			return "", err
		}

		// Check new content size
		if int64(len(newContent)) > cfg.maxFileSize {
			return "", fmt.Errorf("resulting file size %d exceeds maximum %d", len(newContent), cfg.maxFileSize)
		}

		// Write back
		if err := os.WriteFile(path, newContent, 0644); err != nil {
			return "", err
		}

		// Return result as JSON
		result := struct {
			Path         string         `json:"path"`
			Mode         string         `json:"mode"`
			BytesWritten int            `json:"bytes_written"`
			Details      editResultInfo `json:"details"`
		}{
			Path:         path,
			Mode:         args.Mode,
			BytesWritten: len(newContent),
			Details:      editResult,
		}

		out, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	return t, handler
}

// FileTools returns read, write, edit, and list directory tools.
func FileTools(opts ...FileToolOption) []ToolPair {
	readTool, readHandler := NewReadFileTool(opts...)
	writeTool, writeHandler := NewWriteFileTool(opts...)
	editTool, editHandler := NewEditFileTool(opts...)
	listTool, listHandler := NewListDirTool(opts...)

	return []ToolPair{
		{Tool: readTool, Handler: readHandler},
		{Tool: writeTool, Handler: writeHandler},
		{Tool: editTool, Handler: editHandler},
		{Tool: listTool, Handler: listHandler},
	}
}
