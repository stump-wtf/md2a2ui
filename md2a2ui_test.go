package md2a2ui

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConvert_Heading(t *testing.T) {
	tests := []struct {
		name    string
		md      string
		wantVar string
	}{
		{"h1", "# Title", "h1"},
		{"h2", "## Section", "h2"},
		{"h3", "### Subsection", "h3"},
		{"h4", "#### Deep", "h4"},
		{"h5", "##### Deeper", "h5"},
		{"h6", "###### Deepest", "caption"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := Convert(tt.md)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			found := false
			for _, c := range msg.UpdateComponents.Components {
				if c.Component == "Text" && c.Variant == tt.wantVar {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected a Text with variant %q", tt.wantVar)
			}
		})
	}
}

func TestConvert_Paragraph(t *testing.T) {
	msg, err := Convert("Hello world")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	var textComp *Component
	for i := range msg.UpdateComponents.Components {
		c := &msg.UpdateComponents.Components[i]
		if c.Component == "Text" && c.Text == "Hello world" {
			textComp = c
			break
		}
	}
	if textComp == nil {
		t.Fatal("expected a Text component with 'Hello world'")
	}
	if textComp.Variant != "body" {
		t.Errorf("expected variant 'body', got %q", textComp.Variant)
	}
}

func TestConvert_HasRoot(t *testing.T) {
	msg, err := Convert("# Test\n\nSome text")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if err := msg.EnsureRootValid(); err != nil {
		t.Errorf("root validation failed: %v", err)
	}

	for _, c := range msg.UpdateComponents.Components {
		if c.ID == "root" {
			if c.Component != "Column" {
				t.Errorf("root should be a Column, got %s", c.Component)
			}
			if len(c.Children) == 0 {
				t.Error("root should have children")
			}
			return
		}
	}
	t.Fatal("no root component found")
}

func TestConvert_InlineMarkdown(t *testing.T) {
	msg, err := Convert("This is **bold** and *italic*")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	for _, c := range msg.UpdateComponents.Components {
		if c.Component == "Text" && strings.Contains(c.Text, "**bold**") {
			return
		}
	}
	t.Fatal("expected inline markdown to be preserved in Text")
}

func TestConvert_CodeSpan(t *testing.T) {
	msg, err := Convert("Use `code` here")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	for _, c := range msg.UpdateComponents.Components {
		if c.Component == "Text" && strings.Contains(c.Text, "`code`") {
			return
		}
	}
	t.Fatal("expected code span markdown to be preserved")
}

func TestConvert_Link(t *testing.T) {
	msg, err := Convert("[example](https://example.com)")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	for _, c := range msg.UpdateComponents.Components {
		if c.Component == "Text" && strings.Contains(c.Text, "[example](https://example.com)") {
			return
		}
	}
	t.Fatal("expected link markdown to be preserved")
}

func TestConvert_List(t *testing.T) {
	msg, err := Convert("- Item one\n- Item two\n- Item three")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	var listComp *Component
	for i := range msg.UpdateComponents.Components {
		c := &msg.UpdateComponents.Components[i]
		if c.Component == "List" {
			listComp = c
			break
		}
	}
	if listComp == nil {
		t.Fatal("expected a List component")
	}
	if len(listComp.Children) != 3 {
		t.Errorf("expected 3 list items, got %d", len(listComp.Children))
	}
	if listComp.Direction != "vertical" {
		t.Errorf("expected direction 'vertical', got %q", listComp.Direction)
	}
}

func TestConvert_NumberedList(t *testing.T) {
	msg, err := Convert("1. First\n2. Second")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	var listComp *Component
	for i := range msg.UpdateComponents.Components {
		c := &msg.UpdateComponents.Components[i]
		if c.Component == "List" {
			listComp = c
			break
		}
	}
	if listComp == nil {
		t.Fatal("expected a List component for ordered list")
	}
	if len(listComp.Children) != 2 {
		t.Errorf("expected 2 list items, got %d", len(listComp.Children))
	}
}

func TestConvert_CodeBlock(t *testing.T) {
	msg, err := Convert("```go\nfmt.Println(\"hello\")\n```")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	var cardComp *Component
	for i := range msg.UpdateComponents.Components {
		c := &msg.UpdateComponents.Components[i]
		if c.Component == "Card" {
			cardComp = c
			break
		}
	}
	if cardComp == nil {
		t.Fatal("expected a Card component for code block")
	}
}

func TestConvert_Blockquote(t *testing.T) {
	msg, err := Convert("> This is a quote")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	var cardComp *Component
	for i := range msg.UpdateComponents.Components {
		c := &msg.UpdateComponents.Components[i]
		if c.Component == "Card" {
			cardComp = c
			break
		}
	}
	if cardComp == nil {
		t.Fatal("expected a Card component for blockquote")
	}
}

func TestConvert_ThematicBreak(t *testing.T) {
	msg, err := Convert("Above\n\n---\n\nBelow")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	var dividerComp *Component
	for i := range msg.UpdateComponents.Components {
		c := &msg.UpdateComponents.Components[i]
		if c.Component == "Divider" {
			dividerComp = c
			break
		}
	}
	if dividerComp == nil {
		t.Fatal("expected a Divider component")
	}
	if dividerComp.Axis != "horizontal" {
		t.Errorf("expected axis 'horizontal', got %q", dividerComp.Axis)
	}
}

func TestConvert_Table(t *testing.T) {
	msg, err := Convert("| A | B |\n|---|---|\n| 1 | 2 |")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	var colComp *Component
	for i := range msg.UpdateComponents.Components {
		c := &msg.UpdateComponents.Components[i]
		if c.Component == "Column" && c.ID != "root" && len(c.Children) > 0 {
			for _, childID := range c.Children {
				child := msg.UpdateComponents.Components
				for j := range child {
					if child[j].ID == childID && child[j].Component == "Row" {
						colComp = c
						break
					}
				}
			}
			if colComp != nil {
				break
			}
		}
	}
	if colComp == nil {
		t.Fatal("expected a Column containing Row children for the table")
	}
}

func TestConvert_EmptyString(t *testing.T) {
	msg, err := Convert("")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if err := msg.EnsureRootValid(); err != nil {
		t.Errorf("root validation failed for empty input: %v", err)
	}
	if len(msg.UpdateComponents.Components) < 2 {
		t.Errorf("expected at least 2 components for empty input (root + child), got %d", len(msg.UpdateComponents.Components))
	}
}

func TestConvert_SurfaceID(t *testing.T) {
	msg, err := ConvertWithSurface("# Test", "my-surface")
	if err != nil {
		t.Fatalf("ConvertWithSurface() error = %v", err)
	}
	if msg.UpdateComponents.SurfaceID != "my-surface" {
		t.Errorf("expected surfaceId 'my-surface', got %q", msg.UpdateComponents.SurfaceID)
	}
}

func TestConvert_Version(t *testing.T) {
	msg, err := Convert("# Test")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if msg.Version != Version {
		t.Errorf("expected version %q, got %q", Version, msg.Version)
	}
}

func TestConvert_JSONMarshalRoundTrip(t *testing.T) {
	msg, err := Convert("# Hello\n\nWorld")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var back Message
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if back.Version != msg.Version {
		t.Errorf("version mismatch after round-trip: %q vs %q", back.Version, msg.Version)
	}
	if len(back.UpdateComponents.Components) != len(msg.UpdateComponents.Components) {
		t.Errorf("component count mismatch after round-trip: %d vs %d",
			len(back.UpdateComponents.Components), len(msg.UpdateComponents.Components))
	}
}

func TestConvert_ComplexDocument(t *testing.T) {
	md := `# Title

Some paragraph with **bold** text.

## Section

- Item 1
- Item 2

| Col A | Col B |
|-------|-------|
| Cell1 | Cell2 |

> A quote

---

End paragraph.
`
	msg, err := Convert(md)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if err := msg.EnsureRootValid(); err != nil {
		t.Errorf("root validation failed: %v", err)
	}

	componentTypes := make(map[string]int)
	for _, c := range msg.UpdateComponents.Components {
		componentTypes[c.Component]++
	}

	expectedTypes := []string{"Text", "Column", "List", "Row", "Card", "Divider"}
	for _, expected := range expectedTypes {
		if componentTypes[expected] == 0 {
			t.Errorf("expected at least one %s component in complex document", expected)
		}
	}
}

func TestConvert_AllChildIDsValid(t *testing.T) {
	msg, err := Convert("# Title\n\nPara\n\n- a\n- b")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	ids := make(map[string]bool)
	for _, c := range msg.UpdateComponents.Components {
		ids[c.ID] = true
	}

	for _, c := range msg.UpdateComponents.Components {
		for _, childID := range c.Children {
			if !ids[childID] {
				t.Errorf("component %q references non-existent child %q", c.ID, childID)
			}
		}
		if c.Child != "" && !ids[c.Child] {
			t.Errorf("component %q references non-existent child %q", c.ID, c.Child)
		}
	}
}
