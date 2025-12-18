package google

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	ai "github.com/spetersoncode/gains"
	"google.golang.org/genai"
)

func convertMessages(messages []ai.Message) ([]*genai.Content, error) {
	var contents []*genai.Content

	for _, msg := range messages {
		role := "user"
		switch msg.Role {
		case ai.RoleUser:
			role = "user"
		case ai.RoleAssistant:
			role = "model"
		case ai.RoleSystem:
			// Gemini handles system prompts differently - prepend to first user message
			// For simplicity, treat as user message with context
			role = "user"
		case ai.RoleTool:
			// Tool results are sent as user messages with FunctionResponse parts
			role = "user"
		}

		var parts []*genai.Part

		// Handle multimodal content
		if msg.HasParts() {
			convertedParts, err := convertPartsToGoogleParts(msg.Parts)
			if err != nil {
				return nil, err
			}
			parts = convertedParts
		} else if msg.Content != "" {
			parts = append(parts, &genai.Part{Text: msg.Content})
		}

		// Handle tool calls (assistant messages)
		for _, tc := range msg.ToolCalls {
			var args map[string]any
			json.Unmarshal([]byte(tc.Arguments), &args)
			parts = append(parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					Name: tc.Name,
					Args: args,
				},
			})
		}

		// Handle tool results
		for _, tr := range msg.ToolResults {
			// Parse the content as JSON if possible, otherwise wrap as text
			var result map[string]any
			if err := json.Unmarshal([]byte(tr.Content), &result); err != nil {
				result = map[string]any{"result": tr.Content}
			}
			parts = append(parts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     tr.ToolCallID, // Google uses the function name, but we store ID; handle in extractToolCalls
					Response: result,
				},
			})
		}

		if len(parts) > 0 {
			contents = append(contents, &genai.Content{
				Role:  role,
				Parts: parts,
			})
		}
	}

	return contents, nil
}

func convertPartsToGoogleParts(parts []ai.ContentPart) ([]*genai.Part, error) {
	var result []*genai.Part
	for _, part := range parts {
		switch part.Type {
		case ai.ContentPartTypeText:
			if part.Text != "" {
				result = append(result, &genai.Part{Text: part.Text})
			}
		case ai.ContentPartTypeImage:
			if part.Base64 != "" {
				// Decode base64 to bytes
				data, err := base64.StdEncoding.DecodeString(part.Base64)
				if err != nil {
					return nil, &ai.ImageError{Op: "decode", URL: "base64", Err: err}
				}
				mimeType := part.MimeType
				if mimeType == "" {
					mimeType = "image/jpeg" // Default
				}
				result = append(result, &genai.Part{
					InlineData: &genai.Blob{
						Data:     data,
						MIMEType: mimeType,
					},
				})
			} else if part.ImageURL != "" {
				// Google supports GCS URIs directly
				if strings.HasPrefix(part.ImageURL, "gs://") {
					mimeType := part.MimeType
					if mimeType == "" {
						mimeType = inferMimeTypeFromURL(part.ImageURL)
					}
					result = append(result, &genai.Part{
						FileData: &genai.FileData{
							FileURI:  part.ImageURL,
							MIMEType: mimeType,
						},
					})
				} else {
					// HTTP/HTTPS URLs need to be fetched and converted to inline data
					data, mimeType, err := fetchImageFromURL(part.ImageURL)
					if err != nil {
						return nil, &ai.ImageError{Op: "fetch", URL: part.ImageURL, Err: err}
					}
					if part.MimeType != "" {
						mimeType = part.MimeType
					}
					result = append(result, &genai.Part{
						InlineData: &genai.Blob{
							Data:     data,
							MIMEType: mimeType,
						},
					})
				}
			}
		}
	}
	return result, nil
}

func fetchImageFromURL(url string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	// Set User-Agent header - many servers (including Wikipedia) require this
	req.Header.Set("User-Agent", "gains/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Get MIME type from Content-Type header or infer from URL
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = inferMimeTypeFromURL(url)
	}

	return data, mimeType, nil
}

func inferMimeTypeFromURL(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	default:
		return "image/jpeg" // Default fallback
	}
}

// BlockedError indicates the request was blocked by content filtering.
type BlockedError struct {
	Reason string
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("request blocked: %s", e.Reason)
}
