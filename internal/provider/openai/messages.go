package openai

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/openai/openai-go"
	ai "github.com/spetersoncode/gains"
)

func convertMessages(messages []ai.Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	var result []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		switch msg.Role {
		case ai.RoleUser:
			if msg.HasParts() {
				contentParts, err := convertPartsToOpenAIParts(msg.Parts)
				if err != nil {
					return nil, err
				}
				if len(contentParts) > 0 {
					result = append(result, openai.ChatCompletionMessageParamUnion{
						OfUser: &openai.ChatCompletionUserMessageParam{
							Content: openai.ChatCompletionUserMessageParamContentUnion{
								OfArrayOfContentParts: contentParts,
							},
						},
					})
				}
			} else if msg.Content != "" {
				result = append(result, openai.UserMessage(msg.Content))
			}
		case ai.RoleAssistant:
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
				if msg.Content != "" {
					assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					}
				}
				result = append(result, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &assistantMsg,
				})
			} else if msg.Content != "" {
				result = append(result, openai.AssistantMessage(msg.Content))
			}
		case ai.RoleSystem:
			if msg.Content != "" {
				result = append(result, openai.SystemMessage(msg.Content))
			}
		case ai.RoleTool:
			// Tool result messages - one message per tool result
			for _, tr := range msg.ToolResults {
				result = append(result, openai.ToolMessage(tr.Content, tr.ToolCallID))
			}
		default:
			if msg.Content != "" {
				result = append(result, openai.UserMessage(msg.Content))
			}
		}
	}
	return result, nil
}

func convertPartsToOpenAIParts(parts []ai.ContentPart) ([]openai.ChatCompletionContentPartUnionParam, error) {
	var result []openai.ChatCompletionContentPartUnionParam
	for _, part := range parts {
		switch part.Type {
		case ai.ContentPartTypeText:
			if part.Text != "" {
				result = append(result, openai.TextContentPart(part.Text))
			}
		case ai.ContentPartTypeImage:
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
					if err != nil {
						return nil, &ai.ImageError{Op: "fetch", URL: part.ImageURL, Err: err}
					}
					if part.MimeType != "" {
						mimeType = part.MimeType
					}
					b64 := base64.StdEncoding.EncodeToString(data)
					imageURL = fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
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
