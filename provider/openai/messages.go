package openai

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/openai/openai-go"
	"github.com/spetersoncode/gains"
)

func convertMessages(messages []gains.Message) []openai.ChatCompletionMessageParamUnion {
	var result []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		switch msg.Role {
		case gains.RoleUser:
			if msg.HasParts() {
				contentParts := convertPartsToOpenAIParts(msg.Parts)
				result = append(result, openai.ChatCompletionMessageParamUnion{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfArrayOfContentParts: contentParts,
						},
					},
				})
			} else {
				result = append(result, openai.UserMessage(msg.Content))
			}
		case gains.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				// Assistant message with tool calls
				toolCalls := make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls))
				for i, tc := range msg.ToolCalls {
					toolCalls[i] = openai.ChatCompletionMessageToolCallParam{
						ID: tc.ID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Name,
							Arguments: tc.Arguments,
						},
					}
				}
				assistantMsg := openai.ChatCompletionAssistantMessageParam{
					ToolCalls: toolCalls,
				}
				result = append(result, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &assistantMsg,
				})
			} else {
				result = append(result, openai.AssistantMessage(msg.Content))
			}
		case gains.RoleSystem:
			result = append(result, openai.SystemMessage(msg.Content))
		case gains.RoleTool:
			// Tool result messages - one message per tool result
			for _, tr := range msg.ToolResults {
				result = append(result, openai.ToolMessage(tr.Content, tr.ToolCallID))
			}
		default:
			result = append(result, openai.UserMessage(msg.Content))
		}
	}
	return result
}

func convertPartsToOpenAIParts(parts []gains.ContentPart) []openai.ChatCompletionContentPartUnionParam {
	var result []openai.ChatCompletionContentPartUnionParam
	for _, part := range parts {
		switch part.Type {
		case gains.ContentPartTypeText:
			result = append(result, openai.TextContentPart(part.Text))
		case gains.ContentPartTypeImage:
			var imageURL string
			if part.Base64 != "" {
				// Convert to data URI format
				mimeType := part.MimeType
				if mimeType == "" {
					mimeType = "image/jpeg" // Default
				}
				imageURL = fmt.Sprintf("data:%s;base64,%s", mimeType, part.Base64)
			} else if part.ImageURL != "" {
				// Fetch HTTP/HTTPS URLs client-side and convert to base64
				// This avoids issues where OpenAI's servers can't fetch certain URLs
				if strings.HasPrefix(part.ImageURL, "http://") || strings.HasPrefix(part.ImageURL, "https://") {
					data, mimeType, err := fetchImageFromURL(part.ImageURL)
					if err == nil {
						if part.MimeType != "" {
							mimeType = part.MimeType
						}
						b64 := base64.StdEncoding.EncodeToString(data)
						imageURL = fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
					} else {
						// Fall back to passing the URL directly if fetching fails
						imageURL = part.ImageURL
					}
				} else {
					// Non-HTTP URLs (data URIs, etc.) pass through directly
					imageURL = part.ImageURL
				}
			}
			if imageURL != "" {
				result = append(result, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL: imageURL,
				}))
			}
		}
	}
	return result
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
