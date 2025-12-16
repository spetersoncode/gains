// Package tool provides tool infrastructure for the gains library.
//
// This package includes:
//   - Registry and Handler types for tool management
//   - Function binding with automatic schema generation from struct tags
//   - Built-in tools for common operations (file, HTTP, search)
//   - Client tools wrapping gains client capabilities (image, embedding, chat)
//
// # Basic Usage
//
// Define tool arguments as a struct with tags, then use Bind or BindTo:
//
//	type WeatherArgs struct {
//	    Location string `json:"location" desc:"City name" required:"true"`
//	    Unit     string `json:"unit" desc:"Temperature unit" enum:"celsius,fahrenheit"`
//	}
//
//	// Create tool and handler
//	t, h := tool.MustBind("get_weather", "Get current weather",
//	    func(ctx context.Context, args WeatherArgs) (string, error) {
//	        return fmt.Sprintf(`{"temp": 72, "location": %q}`, args.Location), nil
//	    })
//
//	// Register to a registry
//	registry := tool.NewRegistry()
//	registry.MustRegister(t, h)
//
// # Supported Struct Tags
//
// The following tags are supported for schema generation:
//
//	json:"name"      - Property name (required for inclusion)
//	desc:"text"      - Description for the model
//	required:"true"  - Mark field as required
//	enum:"a,b,c"     - Allowed values (comma-separated)
//	min:"0"          - Minimum value (numbers)
//	max:"100"        - Maximum value (numbers)
//	minLength:"1"    - Minimum string length
//	maxLength:"100"  - Maximum string length
//	pattern:"regex"  - String pattern
//	default:"value"  - Default value
//	minItems:"1"     - Minimum array items
//	maxItems:"10"    - Maximum array items
//
// # Built-in Tools
//
// The package provides several built-in tools:
//
// File Tools:
//   - read_file: Read file contents
//   - write_file: Write content to a file
//   - list_directory: List directory contents
//
// HTTP Tool:
//   - http_request: Make HTTP requests
//
// Search Tool:
//   - search_files: Search for patterns in files
//
// Client Tools (require gains client):
//   - generate_image: Generate images from text
//   - embed_text: Generate text embeddings
//   - ask_assistant: Make LLM calls (sub-agent pattern)
//
// # Using Built-in Tools
//
//	registry := tool.NewRegistry()
//
//	// Register file tools with restrictions
//	tool.RegisterAll(registry, tool.FileTools(
//	    tool.WithBasePath("/workspace"),
//	    tool.WithAllowedExtensions(".go", ".md"),
//	))
//
//	// Register all standard tools
//	tool.RegisterAll(registry, tool.AllTools(client,
//	    tool.WithFileOptions(tool.WithBasePath("/workspace")),
//	    tool.WithHTTPOptions(tool.WithAllowedHosts("api.example.com")),
//	))
//
// # Tool Sets
//
// Pre-configured tool collections are available:
//
//   - FileTools(): read, write, list directory
//   - WebTools(): HTTP requests
//   - SearchTools(): file search
//   - ClientTools(): image, embedding, chat (requires client)
//   - StandardTools(): file, HTTP, search (no client required)
//   - AllTools(): all tools including client tools
package tool
