package parser

import (
	"fmt"
	"sort"
	"strings"
)

// metaPriorityKeys defines the preferred key ordering for specific metadata types.
// Priority keys are listed first; remaining keys are sorted alphabetically.
var metaPriorityKeys = map[string][]string{
	"layout": {"x", "y", "w", "h"},
	"style":  {"font-color", "font-size", "text-align", "fill", "stroke"},
}

// Format converts an AST Document back to normalized .grim source text.
// Rules:
//   - Tab indentation
//   - Metadata (@) before children in blocks
//   - Blank lines preserved (multiple collapsed to one)
//   - Trailing blank lines stripped
func Format(doc *Document) string {
	var sb strings.Builder
	for _, stmt := range doc.Statements {
		writeStatement(&sb, stmt, 0)
	}
	// Strip trailing blank lines, keep one trailing newline
	result := strings.TrimRight(sb.String(), "\n")
	if result != "" {
		result += "\n"
	}
	return result
}

func writeStatement(sb *strings.Builder, stmt Statement, depth int) {
	switch s := stmt.(type) {
	case *BlankLine:
		sb.WriteByte('\n')
	case *Comment:
		writeIndent(sb, depth)
		sb.WriteString("# ")
		sb.WriteString(strings.TrimLeft(s.Text, " \t"))
		sb.WriteByte('\n')
	case *NodeDecl:
		writeNodeDecl(sb, s, depth)
	case *EdgeDecl:
		writeEdgeDecl(sb, s, depth)
	}
}

func writeNodeDecl(sb *strings.Builder, n *NodeDecl, depth int) {
	writeIndent(sb, depth)
	sb.WriteString(strings.Join(n.Path, "."))
	if n.Label != nil {
		sb.WriteString(": ")
		writeTextValue(sb, n.Label)
	}
	if n.Block != nil {
		sb.WriteByte(' ')
		writeBlock(sb, n.Block, depth)
	}
	sb.WriteByte('\n')
}

func writeEdgeDecl(sb *strings.Builder, e *EdgeDecl, depth int) {
	writeIndent(sb, depth)
	sb.WriteString(strings.Join(e.From, "."))
	switch e.Direction {
	case "none":
		sb.WriteString(" -- ")
	case "reverse":
		sb.WriteString(" <- ")
	case "bidirectional":
		sb.WriteString(" <-> ")
	default: // "" or "forward"
		sb.WriteString(" -> ")
	}
	sb.WriteString(strings.Join(e.To, "."))
	if e.Label != nil {
		sb.WriteString(": ")
		writeTextValue(sb, e.Label)
	}
	if e.Block != nil {
		sb.WriteByte(' ')
		writeBlock(sb, e.Block, depth)
	}
	sb.WriteByte('\n')
}

func writeBlock(sb *strings.Builder, b *Block, depth int) {
	sb.WriteString("{\n")
	childDepth := depth + 1
	for _, meta := range b.Metadata {
		writeMetadata(sb, meta, childDepth)
	}
	for _, child := range b.Children {
		writeStatement(sb, child, childDepth)
	}
	writeIndent(sb, depth)
	sb.WriteByte('}')
}

func writeMetadata(sb *strings.Builder, meta Metadata, depth int) {
	writeIndent(sb, depth)
	switch m := meta.(type) {
	case *LayoutMeta:
		sb.WriteString("@layout ")
		writeJSONObject(sb, m.Values, "layout")
	case *StyleMeta:
		sb.WriteString("@style ")
		writeJSONObject(sb, m.Values, "style")
	case *ShapeMeta:
		sb.WriteString("@shape ")
		sb.WriteString(m.Value)
	case *TextMeta:
		if m.Format == "" || m.Format == "plain" {
			sb.WriteString("@text ")
		} else {
			sb.WriteString("@text[")
			sb.WriteString(m.Format)
			sb.WriteString("] ")
		}
		writeTextContent(sb, m.Value, depth)
	case *EdgeMeta:
		sb.WriteString("@edge ")
		writeJSONObject(sb, m.Values, "edge")
	case *IconMeta:
		sb.WriteString("@icon ")
		writeJSONObject(sb, m.Values, "icon")
	}
	sb.WriteByte('\n')
}

func writeTextValue(sb *strings.Builder, tv *TextValue) {
	if tv.Format != "" && tv.Format != "plain" {
		sb.WriteString("[")
		sb.WriteString(tv.Format)
		sb.WriteString("] ")
	}
	sb.WriteString(`"`)
	sb.WriteString(escapeString(tv.Value))
	sb.WriteString(`"`)
}

// writeTextContent writes @text value: raw block for multiline, quoted string otherwise.
// Multiline output format:
//
//	{|
//	<depth+1 tabs>line1
//	<depth+1 tabs>line2
//	<depth tabs>|}
func writeTextContent(sb *strings.Builder, value string, depth int) {
	if strings.Contains(value, "\n") {
		sb.WriteString("{|\n")
		for _, line := range strings.Split(value, "\n") {
			if strings.TrimSpace(line) != "" {
				writeIndent(sb, depth+1)
				sb.WriteString(line)
			}
			sb.WriteByte('\n')
		}
		writeIndent(sb, depth)
		sb.WriteString("|}")
	} else {
		sb.WriteString(`"`)
		sb.WriteString(escapeString(value))
		sb.WriteString(`"`)
	}
}

// isSimpleIdent returns true if s can be safely output as an unquoted identifier
// in a JSON-like object value (letters, digits, hyphens, underscores; not a keyword).
func isSimpleIdent(s string) bool {
	if s == "" {
		return false
	}
	switch s {
	case "true", "false", "null", "nil":
		return false // reserved as JSON keywords
	}
	for _, ch := range s {
		if !('a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || '0' <= ch && ch <= '9' || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func writeIndent(sb *strings.Builder, depth int) {
	for i := 0; i < depth; i++ {
		sb.WriteByte('\t')
	}
}

func sortedKeys(m map[string]interface{}, priority []string) []string {
	seen := make(map[string]bool, len(priority))
	result := make([]string, 0, len(m))
	for _, k := range priority {
		if _, ok := m[k]; ok {
			result = append(result, k)
			seen[k] = true
		}
	}
	remaining := make([]string, 0, len(m)-len(result))
	for k := range m {
		if !seen[k] {
			remaining = append(remaining, k)
		}
	}
	sort.Strings(remaining)
	return append(result, remaining...)
}

func writeJSONObject(sb *strings.Builder, m map[string]interface{}, metaType string) {
	priority := metaPriorityKeys[metaType]
	keys := sortedKeys(m, priority)
	sb.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(k)
		sb.WriteString(": ")
		writeJSONValue(sb, m[k])
	}
	sb.WriteByte('}')
}

func writeJSONValue(sb *strings.Builder, v interface{}) {
	switch val := v.(type) {
	case nil:
		sb.WriteString("null")
	case bool:
		if val {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case int64:
		sb.WriteString(fmt.Sprintf("%d", val))
	case float64:
		// Render whole numbers without decimal point
		if val == float64(int64(val)) && val >= -1e15 && val <= 1e15 {
			sb.WriteString(fmt.Sprintf("%d", int64(val)))
		} else {
			sb.WriteString(fmt.Sprintf("%g", val))
		}
	case string:
		// Output unquoted if it looks like a valid identifier (no spaces, not a keyword)
		if isSimpleIdent(val) {
			sb.WriteString(val)
		} else {
			sb.WriteString(`"`)
			sb.WriteString(escapeString(val))
			sb.WriteString(`"`)
		}
	case map[string]interface{}:
		writeJSONObject(sb, val, "")
	case []interface{}:
		sb.WriteByte('[')
		for i, item := range val {
			if i > 0 {
				sb.WriteString(", ")
			}
			writeJSONValue(sb, item)
		}
		sb.WriteByte(']')
	default:
		sb.WriteString(fmt.Sprintf("%v", val))
	}
}
