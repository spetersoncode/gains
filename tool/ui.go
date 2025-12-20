package tool

import (
	"encoding/json"

	ai "github.com/spetersoncode/gains"
)

// UIComponentType identifies the kind of UI component to generate.
type UIComponentType string

const (
	// UIComponentForm renders a form with input fields.
	UIComponentForm UIComponentType = "form"

	// UIComponentConfirmation renders a confirmation dialog.
	UIComponentConfirmation UIComponentType = "confirmation"

	// UIComponentCard renders an information card.
	UIComponentCard UIComponentType = "card"

	// UIComponentList renders a selectable list.
	UIComponentList UIComponentType = "list"

	// UIComponentProgress renders a progress indicator.
	UIComponentProgress UIComponentType = "progress"

	// UIComponentCustom renders a custom component (frontend-specific).
	UIComponentCustom UIComponentType = "custom"
)

// UIComponent defines a UI component for the frontend to render.
// This is the schema for the generateUserInterface tool arguments.
type UIComponent struct {
	// Type specifies the kind of component to render.
	Type UIComponentType `json:"type" desc:"Type of UI component" required:"true" enum:"form,confirmation,card,list,progress,custom"`

	// Title is the component's heading or title.
	Title string `json:"title,omitempty" desc:"Component title or heading"`

	// Description provides additional context.
	Description string `json:"description,omitempty" desc:"Additional description or context"`

	// Fields defines form fields (for form type).
	Fields []UIField `json:"fields,omitempty" desc:"Form fields for type=form"`

	// Items defines list items (for list type).
	Items []UIListItem `json:"items,omitempty" desc:"List items for type=list"`

	// Actions defines available actions/buttons.
	Actions []UIAction `json:"actions,omitempty" desc:"Action buttons"`

	// Progress is the progress value 0-100 (for progress type).
	Progress int `json:"progress,omitempty" desc:"Progress percentage 0-100 for type=progress"`

	// CustomType is the custom component identifier (for custom type).
	CustomType string `json:"customType,omitempty" desc:"Custom component type identifier"`

	// CustomData is arbitrary data for custom components.
	CustomData json.RawMessage `json:"customData,omitempty" desc:"Custom component data"`

	// Metadata is additional metadata for the component.
	Metadata map[string]any `json:"metadata,omitempty" desc:"Additional component metadata"`
}

// UIFieldType identifies the kind of form field.
type UIFieldType string

const (
	UIFieldText     UIFieldType = "text"
	UIFieldNumber   UIFieldType = "number"
	UIFieldEmail    UIFieldType = "email"
	UIFieldPassword UIFieldType = "password"
	UIFieldTextarea UIFieldType = "textarea"
	UIFieldSelect   UIFieldType = "select"
	UIFieldCheckbox UIFieldType = "checkbox"
	UIFieldRadio    UIFieldType = "radio"
	UIFieldDate     UIFieldType = "date"
	UIFieldFile     UIFieldType = "file"
)

// UIField defines a form field.
type UIField struct {
	// Name is the field identifier (used in result).
	Name string `json:"name" desc:"Field identifier" required:"true"`

	// Type is the field type.
	Type UIFieldType `json:"type" desc:"Field type" required:"true" enum:"text,number,email,password,textarea,select,checkbox,radio,date,file"`

	// Label is the field's display label.
	Label string `json:"label,omitempty" desc:"Display label"`

	// Placeholder is the placeholder text.
	Placeholder string `json:"placeholder,omitempty" desc:"Placeholder text"`

	// Default is the default value.
	Default string `json:"default,omitempty" desc:"Default value"`

	// Required indicates if the field is required.
	Required bool `json:"required,omitempty" desc:"Whether field is required"`

	// Options are the choices for select/radio fields.
	Options []UIOption `json:"options,omitempty" desc:"Options for select/radio fields"`

	// Validation contains validation rules.
	Validation *UIValidation `json:"validation,omitempty" desc:"Validation rules"`
}

// UIOption defines an option for select/radio fields.
type UIOption struct {
	Value string `json:"value" desc:"Option value" required:"true"`
	Label string `json:"label" desc:"Display label" required:"true"`
}

// UIValidation defines validation rules for a field.
type UIValidation struct {
	MinLength int    `json:"minLength,omitempty" desc:"Minimum text length"`
	MaxLength int    `json:"maxLength,omitempty" desc:"Maximum text length"`
	Min       int    `json:"min,omitempty" desc:"Minimum numeric value"`
	Max       int    `json:"max,omitempty" desc:"Maximum numeric value"`
	Pattern   string `json:"pattern,omitempty" desc:"Regex pattern for validation"`
	Message   string `json:"message,omitempty" desc:"Custom validation error message"`
}

// UIListItem defines an item in a list component.
type UIListItem struct {
	ID          string `json:"id" desc:"Item identifier" required:"true"`
	Title       string `json:"title" desc:"Item title" required:"true"`
	Description string `json:"description,omitempty" desc:"Item description"`
	Icon        string `json:"icon,omitempty" desc:"Icon identifier"`
	Selectable  bool   `json:"selectable,omitempty" desc:"Whether item is selectable"`
}

// UIAction defines an action button.
type UIAction struct {
	ID      string `json:"id" desc:"Action identifier" required:"true"`
	Label   string `json:"label" desc:"Button label" required:"true"`
	Primary bool   `json:"primary,omitempty" desc:"Whether this is the primary action"`
	Danger  bool   `json:"danger,omitempty" desc:"Whether this is a destructive action"`
}

// UIResult is the result returned from a UI component interaction.
type UIResult struct {
	// Action is the action that was triggered (for button clicks).
	Action string `json:"action,omitempty"`

	// Values contains form field values.
	Values map[string]any `json:"values,omitempty"`

	// SelectedItems contains selected list item IDs.
	SelectedItems []string `json:"selectedItems,omitempty"`

	// Cancelled indicates if the user dismissed the UI.
	Cancelled bool `json:"cancelled,omitempty"`
}

// GenerateUserInterfaceTool creates a client-side tool for generating UI.
// This tool is handled by the frontend - it renders the specified component
// and returns the user's interaction as the tool result.
//
// Example usage in agent:
//
//	registry.RegisterClientTool(tool.GenerateUserInterfaceTool())
//
// The agent can then call generateUserInterface to show dynamic UI:
//
//	{
//	    "type": "form",
//	    "title": "User Details",
//	    "fields": [
//	        {"name": "email", "type": "email", "label": "Email", "required": true}
//	    ],
//	    "actions": [
//	        {"id": "submit", "label": "Submit", "primary": true}
//	    ]
//	}
func GenerateUserInterfaceTool() ai.Tool {
	return ai.Tool{
		Name:        "generateUserInterface",
		Description: "Generate a dynamic UI component for the user to interact with. Use this to show forms, confirmations, cards, lists, or custom UI components. The result contains the user's input or action.",
		Parameters:  MustSchemaFor[UIComponent](),
	}
}

// ParseUIResult parses the tool result from a generateUserInterface call.
func ParseUIResult(result string) (*UIResult, error) {
	var r UIResult
	if err := json.Unmarshal([]byte(result), &r); err != nil {
		return nil, err
	}
	return &r, nil
}
