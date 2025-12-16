package tool

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	ai "github.com/spetersoncode/gains"
)

// SearchToolOption configures the search tool.
type SearchToolOption func(*searchToolConfig)

type searchToolConfig struct {
	basePath        string
	maxResults      int
	includePatterns []string
	excludePatterns []string
}

// WithSearchPath sets the base path for searches.
func WithSearchPath(path string) SearchToolOption {
	return func(c *searchToolConfig) {
		c.basePath = path
	}
}

// WithMaxResults limits the number of search results.
// Default is 100.
func WithMaxResults(n int) SearchToolOption {
	return func(c *searchToolConfig) {
		c.maxResults = n
	}
}

// WithIncludePatterns sets glob patterns for files to include.
func WithIncludePatterns(patterns ...string) SearchToolOption {
	return func(c *searchToolConfig) {
		c.includePatterns = patterns
	}
}

// WithExcludePatterns sets glob patterns for files to exclude.
func WithExcludePatterns(patterns ...string) SearchToolOption {
	return func(c *searchToolConfig) {
		c.excludePatterns = patterns
	}
}

func applySearchOpts(opts []SearchToolOption) *searchToolConfig {
	cfg := &searchToolConfig{
		maxResults: 100,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.basePath == "" {
		cfg.basePath = "."
	}
	return cfg
}

func (c *searchToolConfig) shouldInclude(path string) bool {
	// Check exclude patterns first
	for _, pattern := range c.excludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return false
		}
	}

	// If no include patterns, include all
	if len(c.includePatterns) == 0 {
		return true
	}

	// Check include patterns
	for _, pattern := range c.includePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}

	return false
}

// searchArgs defines arguments for the search tool.
type searchArgs struct {
	Pattern     string `json:"pattern" desc:"Regex pattern to search for" required:"true"`
	Path        string `json:"path" desc:"Directory to search in (defaults to current)"`
	FilePattern string `json:"file_pattern" desc:"Glob pattern for file names (e.g., *.go)"`
}

// NewSearchTool creates a tool for searching file contents with regex.
func NewSearchTool(opts ...SearchToolOption) (ai.Tool, Handler) {
	cfg := applySearchOpts(opts)

	schema := MustSchemaFor[searchArgs]()

	t := ai.Tool{
		Name:        "search_files",
		Description: "Search for a pattern in file contents",
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args searchArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}

		// Compile regex
		re, err := regexp.Compile(args.Pattern)
		if err != nil {
			return "", err
		}

		// Determine search path
		searchPath := cfg.basePath
		if args.Path != "" {
			searchPath = filepath.Join(cfg.basePath, args.Path)
		}

		type match struct {
			File    string `json:"file"`
			Line    int    `json:"line"`
			Content string `json:"content"`
		}

		var matches []match
		resultCount := 0

		err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Check file pattern if specified
			if args.FilePattern != "" {
				if matched, _ := filepath.Match(args.FilePattern, info.Name()); !matched {
					return nil
				}
			}

			// Check include/exclude patterns from config
			if !cfg.shouldInclude(path) {
				return nil
			}

			// Skip binary files and very large files
			if info.Size() > 10*1024*1024 { // 10MB
				return nil
			}

			// Open and search file
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			lineNum := 0

			for scanner.Scan() {
				lineNum++
				line := scanner.Text()

				if re.MatchString(line) {
					relPath, _ := filepath.Rel(cfg.basePath, path)
					if relPath == "" {
						relPath = path
					}

					// Truncate long lines
					content := line
					if len(content) > 200 {
						content = content[:200] + "..."
					}
					content = strings.TrimSpace(content)

					matches = append(matches, match{
						File:    relPath,
						Line:    lineNum,
						Content: content,
					})

					resultCount++
					if resultCount >= cfg.maxResults {
						return filepath.SkipAll
					}
				}
			}

			return nil
		})

		if err != nil && err != filepath.SkipAll {
			return "", err
		}

		result := struct {
			Pattern   string  `json:"pattern"`
			Path      string  `json:"path"`
			Count     int     `json:"count"`
			Truncated bool    `json:"truncated,omitempty"`
			Matches   []match `json:"matches"`
		}{
			Pattern:   args.Pattern,
			Path:      searchPath,
			Count:     len(matches),
			Truncated: resultCount >= cfg.maxResults,
			Matches:   matches,
		}

		out, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	return t, handler
}

// SearchTools returns the search tool.
func SearchTools(opts ...SearchToolOption) []ToolPair {
	t, h := NewSearchTool(opts...)
	return []ToolPair{{Tool: t, Handler: h}}
}
