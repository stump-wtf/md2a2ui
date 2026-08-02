// Package md2a2ui converts Markdown text into A2UI component trees.
//
// It uses goldmark to parse CommonMark markdown and walks the resulting AST,
// emitting a flat list of A2UI components connected by ID references in the
// adjacency-list model the A2UI protocol expects.
//
// The output is a complete updateComponents message, ready to marshal as JSON
// and send to any A2UI renderer that uses the v0.9 basic catalog.
package md2a2ui

import (
	"fmt"
	"strconv"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extensionAST "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Version is the A2UI protocol version this library targets.
const Version = "v0.9"

// Message is a top-level A2UI agent-to-renderer message containing an
// updateComponents payload. Marshal it to JSON and send it to a renderer
// that already has a surface created.
type Message struct {
	Version          string            `json:"version"`
	UpdateComponents *UpdateComponents `json:"updateComponents"`
}

// UpdateComponents is the A2UI message body that carries a flat list of
// component definitions for a specific surface.
type UpdateComponents struct {
	SurfaceID  string      `json:"surfaceId"`
	Components []Component `json:"components"`
}

// Component is a single A2UI component in the adjacency-list model.
// Only the fields relevant to the markdown→A2UI mapping are included;
// the struct uses omitempty on optional fields so the JSON stays clean.
type Component struct {
	ID        string   `json:"id"`
	Component string   `json:"component"`
	Text      string   `json:"text,omitempty"`
	Variant   string   `json:"variant,omitempty"`
	Children  []string `json:"children,omitempty"`
	Child     string   `json:"child,omitempty"`
	Axis      string   `json:"axis,omitempty"`
	URL       string   `json:"url,omitempty"`
	AltText   string   `json:"altText,omitempty"`
	Direction string   `json:"direction,omitempty"`
}

// Convert parses markdown and returns a complete updateComponents A2UI
// message. The component tree is rooted in a Column whose id is "root".
//
// A custom surfaceId can be set via the ConvertWithSurface function.
func Convert(markdown string) (*Message, error) {
	return ConvertWithSurface(markdown, "markdown")
}

// ConvertWithSurface is like Convert but lets the caller choose the
// surfaceId that appears in the updateComponents message.
func ConvertWithSurface(markdown, surfaceID string) (*Message, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)

	reader := text.NewReader([]byte(markdown))
	doc := md.Parser().Parse(reader)

	b := &builder{
		idCounter: 0,
	}

	rootColID := "root"
	rootChildIDs := b.convertBlockChildren(doc, reader)

	// If the document produced no children, emit an empty Text so the
	// Column has at least one valid child.
	if len(rootChildIDs) == 0 {
		textID := b.nextID("text")
		b.add(Component{
			ID:        textID,
			Component: "Text",
			Text:      "",
			Variant:   "body",
		})
		rootChildIDs = []string{textID}
	}

	b.add(Component{
		ID:        rootColID,
		Component: "Column",
		Children:  rootChildIDs,
	})

	return &Message{
		Version: Version,
		UpdateComponents: &UpdateComponents{
			SurfaceID:  surfaceID,
			Components: b.components,
		},
	}, nil
}

// builder holds state during AST traversal: a growing flat list of components
// and a monotonic counter for unique IDs.
type builder struct {
	components []Component
	idCounter  int
}

func (b *builder) add(c Component) {
	b.components = append(b.components, c)
}

func (b *builder) nextID(prefix string) string {
	b.idCounter++
	return prefix + "-" + strconv.Itoa(b.idCounter)
}

// convertBlockChildren walks the direct block children of a container node
// and returns the IDs of the top-level components produced for each child.
func (b *builder) convertBlockChildren(parent ast.Node, reader text.Reader) []string {
	var ids []string
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if id, ok := b.convertBlock(child, reader); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// convertBlock converts a single block-level AST node into one or more A2UI
// components. It returns the ID of the top-level component produced (if any).
func (b *builder) convertBlock(node ast.Node, reader text.Reader) (string, bool) {
	switch n := node.(type) {
	case *ast.Heading:
		return b.convertHeading(n, reader), true

	case *ast.Paragraph:
		textContent := b.renderInlineChildren(n, reader)
		id := b.nextID("text")
		b.add(Component{
			ID:        id,
			Component: "Text",
			Text:      textContent,
			Variant:   "body",
		})
		return id, true

	case *ast.Blockquote:
		textContent := b.renderBlockText(n, reader)
		cardColID := b.nextID("col")
		cardTextID := b.nextID("text")
		b.add(Component{
			ID:        cardTextID,
			Component: "Text",
			Text:      textContent,
			Variant:   "body",
		})
		b.add(Component{
			ID:        cardColID,
			Component: "Column",
			Children:  []string{cardTextID},
		})
		cardID := b.nextID("card")
		b.add(Component{
			ID:        cardID,
			Component: "Card",
			Child:     cardColID,
		})
		return cardID, true

	case *ast.FencedCodeBlock:
		return b.convertCodeBlock(n, reader), true

	case *ast.CodeBlock:
		return b.convertCodeBlock(n, reader), true

	case *ast.List:
		return b.convertList(n, reader), true

	case *ast.ThematicBreak:
		id := b.nextID("divider")
		b.add(Component{
			ID:        id,
			Component: "Divider",
			Axis:      "horizontal",
		})
		return id, true

	case *extensionAST.Table:
		return b.convertTable(n, reader), true

	case *ast.HTMLBlock:
		// Render raw HTML as plain text in a body Text component.
		segments := n.BaseBlock.Lines()
		textContent := string(segments.Value(reader.Source()))
		id := b.nextID("text")
		b.add(Component{
			ID:        id,
			Component: "Text",
			Text:      textContent,
			Variant:   "body",
		})
		return id, true

	default:
		// Fallback: render any text content from the node.
		textContent := string(node.Text(reader.Source()))
		if textContent == "" {
			return "", false
		}
		id := b.nextID("text")
		b.add(Component{
			ID:        id,
			Component: "Text",
			Text:      textContent,
			Variant:   "body",
		})
		return id, true
	}
}

func (b *builder) convertHeading(n *ast.Heading, reader text.Reader) string {
	textContent := b.renderInlineChildren(n, reader)
	variant := headingVariant(n.Level)
	id := b.nextID("text")
	b.add(Component{
		ID:        id,
		Component: "Text",
		Text:      textContent,
		Variant:   variant,
	})
	return id
}

func headingVariant(level int) string {
	switch level {
	case 1:
		return "h1"
	case 2:
		return "h2"
	case 3:
		return "h3"
	case 4:
		return "h4"
	case 5:
		return "h5"
	default:
		return "caption"
	}
}

func (b *builder) convertCodeBlock(n ast.Node, reader text.Reader) string {
	var code string
	if lines := n.Lines(); lines.Len() > 0 {
		code = string(lines.Value(reader.Source()))
	}

	cardTextID := b.nextID("text")
	b.add(Component{
		ID:        cardTextID,
		Component: "Text",
		Text:      code,
		Variant:   "body",
	})

	cardColID := b.nextID("col")
	b.add(Component{
		ID:        cardColID,
		Component: "Column",
		Children:  []string{cardTextID},
	})

	cardID := b.nextID("card")
	b.add(Component{
		ID:        cardID,
		Component: "Card",
		Child:     cardColID,
	})
	return cardID
}

func (b *builder) convertList(n *ast.List, reader text.Reader) string {
	var itemIDs []string
	for item := n.FirstChild(); item != nil; item = item.NextSibling() {
		textContent := b.renderInlineChildren(item, reader)
		id := b.nextID("text")
		b.add(Component{
			ID:        id,
			Component: "Text",
			Text:      textContent,
			Variant:   "body",
		})
		itemIDs = append(itemIDs, id)
	}

	listID := b.nextID("list")
	b.add(Component{
		ID:        listID,
		Component: "List",
		Children:  itemIDs,
		Direction: "vertical",
	})
	return listID
}

func (b *builder) convertTable(n *extensionAST.Table, reader text.Reader) string {
	var rowIDs []string

	for row := n.FirstChild(); row != nil; row = row.NextSibling() {
		var cellIDs []string
		isHeader := false

		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if _, ok := cell.(*extensionAST.TableCell); ok {
				// TableCell is handled by rendering its inline children
			}
			textContent := b.renderInlineChildren(cell, reader)
			if textContent == "" {
				textContent = " "
			}
			variant := "body"
			cellID := b.nextID("text")
			b.add(Component{
				ID:        cellID,
				Component: "Text",
				Text:      textContent,
				Variant:   variant,
			})
			cellIDs = append(cellIDs, cellID)
		}

		rowID := b.nextID("row")
		b.add(Component{
			ID:        rowID,
			Component: "Row",
			Children:  cellIDs,
		})

		// Mark first row as header by checking if it's inside a thead.
		// goldmark doesn't expose thead/tbody directly, so we treat the
		// first row as header for styling purposes.
		if len(rowIDs) == 0 {
			isHeader = true
		}
		_ = isHeader

		rowIDs = append(rowIDs, rowID)
	}

	// Style the first row's cells as caption (header look).
	// We retroactively find the first row's children and update their variant.
	if len(rowIDs) > 0 {
		firstRowChildren := b.findComponent(rowIDs[0])
		if firstRowChildren != nil {
			for _, childID := range firstRowChildren.Children {
				if comp := b.findComponent(childID); comp != nil {
					comp.Variant = "caption"
				}
			}
		}
	}

	tableColID := b.nextID("col")
	b.add(Component{
		ID:        tableColID,
		Component: "Column",
		Children:  rowIDs,
	})

	return tableColID
}

// renderInlineChildren walks the inline children of a node and returns
// their concatenated text as a markdown string (since A2UI Text supports
// simple inline markdown).
func (b *builder) renderInlineChildren(node ast.Node, reader text.Reader) string {
	var buf []byte
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		buf = append(buf, b.renderInlineNode(child, reader)...)
	}
	return string(buf)
}

// renderInlineNode renders a single inline AST node back to markdown text.
// Since A2UI Text components support simple markdown (bold, italic, code,
// links), we reconstruct the markdown syntax from the AST.
func (b *builder) renderInlineNode(node ast.Node, reader text.Reader) []byte {
	switch n := node.(type) {
	case *ast.Text:
		return n.Value(reader.Source())

	case *ast.Emphasis:
		inner := b.renderInlineChildren(n, reader)
		if n.Level == 2 {
			return []byte("**" + string(inner) + "**")
		}
		return []byte("*" + string(inner) + "*")

	case *ast.CodeSpan:
		inner := string(n.Text(reader.Source()))
		return []byte("`" + inner + "`")

	case *ast.Link:
		inner := b.renderInlineChildren(n, reader)
		return []byte("[" + string(inner) + "](" + string(n.Destination) + ")")

	case *ast.Image:
		alt := string(n.Text(reader.Source()))
		return []byte("!(" + alt + ")(" + string(n.Destination) + ")")

	case *ast.AutoLink:
		return n.URL(reader.Source())

	default:
		return node.Text(reader.Source())
	}
}

// renderBlockText extracts all text content from a block node (for
// cases like blockquotes where we want the inner text as a string).
func (b *builder) renderBlockText(node ast.Node, reader text.Reader) string {
	var parts []string
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		text := b.renderInlineChildren(child, reader)
		if text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += "\n\n" + p
	}
	return result
}

// findComponent returns a pointer to a component in the flat list by ID,
// or nil if not found.
func (b *builder) findComponent(id string) *Component {
	for i := range b.components {
		if b.components[i].ID == id {
			return &b.components[i]
		}
	}
	return nil
}

// String returns a human-readable summary of the message for debugging.
func (m *Message) String() string {
	if m == nil || m.UpdateComponents == nil {
		return "<nil>"
	}
	return fmt.Sprintf("Message{version=%s, surface=%s, components=%d}",
		m.Version, m.UpdateComponents.SurfaceID, len(m.UpdateComponents.Components))
}

// EnsureRootValid verifies that the component list has exactly one component
// with id "root". A2UI requires this.
func (m *Message) EnsureRootValid() error {
	if m == nil || m.UpdateComponents == nil {
		return fmt.Errorf("message or updateComponents is nil")
	}
	rootCount := 0
	for _, c := range m.UpdateComponents.Components {
		if c.ID == "root" {
			rootCount++
		}
	}
	if rootCount == 0 {
		return fmt.Errorf("no component with id \"root\" found")
	}
	if rootCount > 1 {
		return fmt.Errorf("multiple components with id \"root\" found: %d", rootCount)
	}
	return nil
}
