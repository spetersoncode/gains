package tool

import (
	"encoding/json"
	"testing"
)

func TestGenerateUserInterfaceTool(t *testing.T) {
	tool := GenerateUserInterfaceTool()

	if tool.Name != "generateUserInterface" {
		t.Errorf("expected name 'generateUserInterface', got %q", tool.Name)
	}

	if tool.Description == "" {
		t.Error("expected non-empty description")
	}

	if len(tool.Parameters) == 0 {
		t.Error("expected non-empty parameters schema")
	}
}

func TestUIComponent_FormJSON(t *testing.T) {
	component := UIComponent{
		Type:        UIComponentForm,
		Title:       "Contact Form",
		Description: "Please fill out the form",
		Fields: []UIField{
			{
				Name:        "name",
				Type:        UIFieldText,
				Label:       "Full Name",
				Required:    true,
				Placeholder: "Enter your name",
			},
			{
				Name:  "email",
				Type:  UIFieldEmail,
				Label: "Email Address",
				Validation: &UIValidation{
					Pattern: "^[a-zA-Z0-9+_.-]+@[a-zA-Z0-9.-]+$",
					Message: "Please enter a valid email",
				},
			},
			{
				Name:  "country",
				Type:  UIFieldSelect,
				Label: "Country",
				Options: []UIOption{
					{Value: "us", Label: "United States"},
					{Value: "uk", Label: "United Kingdom"},
					{Value: "ca", Label: "Canada"},
				},
			},
		},
		Actions: []UIAction{
			{ID: "submit", Label: "Submit", Primary: true},
			{ID: "cancel", Label: "Cancel"},
		},
	}

	data, err := json.Marshal(component)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed UIComponent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Type != UIComponentForm {
		t.Errorf("expected type 'form', got %q", parsed.Type)
	}
	if len(parsed.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(parsed.Fields))
	}
	if len(parsed.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(parsed.Actions))
	}
}

func TestUIComponent_ConfirmationJSON(t *testing.T) {
	component := UIComponent{
		Type:        UIComponentConfirmation,
		Title:       "Delete Item?",
		Description: "This action cannot be undone.",
		Actions: []UIAction{
			{ID: "confirm", Label: "Delete", Primary: true, Danger: true},
			{ID: "cancel", Label: "Cancel"},
		},
	}

	data, err := json.Marshal(component)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed UIComponent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Type != UIComponentConfirmation {
		t.Errorf("expected type 'confirmation', got %q", parsed.Type)
	}
	if !parsed.Actions[0].Danger {
		t.Error("expected first action to be danger")
	}
}

func TestUIComponent_ListJSON(t *testing.T) {
	component := UIComponent{
		Type:  UIComponentList,
		Title: "Select Items",
		Items: []UIListItem{
			{ID: "1", Title: "Item 1", Description: "First item", Selectable: true},
			{ID: "2", Title: "Item 2", Description: "Second item", Selectable: true},
			{ID: "3", Title: "Item 3", Description: "Third item", Selectable: false},
		},
	}

	data, err := json.Marshal(component)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed UIComponent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(parsed.Items))
	}
}

func TestUIComponent_ProgressJSON(t *testing.T) {
	component := UIComponent{
		Type:        UIComponentProgress,
		Title:       "Processing",
		Description: "Please wait...",
		Progress:    75,
	}

	data, err := json.Marshal(component)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed UIComponent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Progress != 75 {
		t.Errorf("expected progress 75, got %d", parsed.Progress)
	}
}

func TestUIComponent_CustomJSON(t *testing.T) {
	customData := map[string]any{
		"chartType": "bar",
		"data":      []int{1, 2, 3, 4, 5},
	}
	customDataJSON, _ := json.Marshal(customData)

	component := UIComponent{
		Type:       UIComponentCustom,
		Title:      "Custom Chart",
		CustomType: "chart",
		CustomData: customDataJSON,
		Metadata: map[string]any{
			"theme": "dark",
		},
	}

	data, err := json.Marshal(component)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed UIComponent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.CustomType != "chart" {
		t.Errorf("expected customType 'chart', got %q", parsed.CustomType)
	}
	if parsed.Metadata["theme"] != "dark" {
		t.Errorf("expected metadata theme 'dark', got %v", parsed.Metadata["theme"])
	}
}

func TestParseUIResult(t *testing.T) {
	t.Run("form submission", func(t *testing.T) {
		json := `{"action":"submit","values":{"name":"John","email":"john@example.com"}}`
		result, err := ParseUIResult(json)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Action != "submit" {
			t.Errorf("expected action 'submit', got %q", result.Action)
		}
		if result.Values["name"] != "John" {
			t.Errorf("expected name 'John', got %v", result.Values["name"])
		}
	})

	t.Run("list selection", func(t *testing.T) {
		json := `{"selectedItems":["1","3"]}`
		result, err := ParseUIResult(json)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.SelectedItems) != 2 {
			t.Errorf("expected 2 selected items, got %d", len(result.SelectedItems))
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		json := `{"cancelled":true}`
		result, err := ParseUIResult(json)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Cancelled {
			t.Error("expected cancelled to be true")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := ParseUIResult(`{invalid}`)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}
