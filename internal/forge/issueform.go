// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// IssueForm is a GitHub issue form (.github/ISSUE_TEMPLATE/*.yml) as the portal renders it.
// Changing a form in the repository changes the portal (PRD §5, V1).
type IssueForm struct {
	Name        string
	Description string
	TitlePrefix string
	Labels      []string
	Type        string
	Fields      []Field
}

type Field struct {
	ID          string
	Type        string // markdown, input, textarea, dropdown, checkboxes
	Label       string
	Description string
	Placeholder string
	Value       string // markdown text, or default value of an input/textarea
	Options     []string
	// RequiredOptions lists the checkboxes that must be ticked.
	RequiredOptions []string
	Multiple        bool
	Required        bool
}

type rawForm struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Title       string    `yaml:"title"`
	Labels      rawLabels `yaml:"labels"`
	Type        string    `yaml:"type"`
	Body        []struct {
		Type       string `yaml:"type"`
		ID         string `yaml:"id"`
		Attributes struct {
			Label       string    `yaml:"label"`
			Description string    `yaml:"description"`
			Placeholder string    `yaml:"placeholder"`
			Value       string    `yaml:"value"`
			Multiple    bool      `yaml:"multiple"`
			Options     yaml.Node `yaml:"options"`
		} `yaml:"attributes"`
		Validations struct {
			Required bool `yaml:"required"`
		} `yaml:"validations"`
	} `yaml:"body"`
}

// rawLabels accepts both a YAML list and GitHub's comma-separated string.
type rawLabels []string

func (l *rawLabels) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		for _, s := range strings.Split(n.Value, ",") {
			if s = strings.TrimSpace(s); s != "" {
				*l = append(*l, s)
			}
		}
		return nil
	}
	var list []string
	if err := n.Decode(&list); err != nil {
		return err
	}
	*l = list
	return nil
}

var fieldTypes = map[string]bool{"markdown": true, "input": true, "textarea": true, "dropdown": true, "checkboxes": true}

// ParseIssueForm parses a form file. Fields without an id get a stable positional one.
func ParseIssueForm(data []byte) (IssueForm, error) {
	var raw rawForm
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return IssueForm{}, fmt.Errorf("issue form: %w", err)
	}
	if raw.Name == "" || len(raw.Body) == 0 {
		return IssueForm{}, fmt.Errorf("issue form: name and body are required")
	}
	form := IssueForm{
		Name:        raw.Name,
		Description: raw.Description,
		TitlePrefix: raw.Title,
		Labels:      raw.Labels,
		Type:        raw.Type,
	}
	for i, b := range raw.Body {
		if !fieldTypes[b.Type] {
			return IssueForm{}, fmt.Errorf("issue form: field %d has unknown type %q", i+1, b.Type)
		}
		f := Field{
			ID:          b.ID,
			Type:        b.Type,
			Label:       b.Attributes.Label,
			Description: b.Attributes.Description,
			Placeholder: b.Attributes.Placeholder,
			Value:       b.Attributes.Value,
			Multiple:    b.Attributes.Multiple,
			Required:    b.Validations.Required,
		}
		if f.ID == "" {
			f.ID = fmt.Sprintf("field-%d", i+1)
		}
		if err := decodeOptions(&f, &b.Attributes.Options); err != nil {
			return IssueForm{}, fmt.Errorf("issue form: field %s: %w", f.ID, err)
		}
		if f.Type != "markdown" && f.Label == "" {
			return IssueForm{}, fmt.Errorf("issue form: field %s has no label", f.ID)
		}
		form.Fields = append(form.Fields, f)
	}
	return form, nil
}

// Dropdown options are strings; checkbox options are {label, required} objects.
func decodeOptions(f *Field, n *yaml.Node) error {
	if n.Kind == 0 {
		return nil
	}
	if f.Type == "checkboxes" {
		var opts []struct {
			Label    string `yaml:"label"`
			Required bool   `yaml:"required"`
		}
		if err := n.Decode(&opts); err != nil {
			return err
		}
		for _, o := range opts {
			f.Options = append(f.Options, o.Label)
			if o.Required {
				f.RequiredOptions = append(f.RequiredOptions, o.Label)
				f.Required = true
			}
		}
		return nil
	}
	return n.Decode(&f.Options)
}
