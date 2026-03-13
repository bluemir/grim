package parser

// Document is the root AST node containing all top-level statements.
type Document struct {
	Statements []Statement
}

// Statement is the interface for all top-level and nested declarations.
type Statement interface {
	statementNode()
	GetLine() int
}

// NodeDecl declares a node with an optional label and block.
type NodeDecl struct {
	Line  int
	Path  []string   // e.g. ["my-node", "inside-node"]
	Label *TextValue // shorthand label from colon syntax
	Block *Block     // optional block with metadata/children
}

func (*NodeDecl) statementNode() {}
func (n *NodeDecl) GetLine() int { return n.Line }

// EdgeDecl declares an edge between two nodes.
type EdgeDecl struct {
	Line      int
	From      []string   // source path
	To        []string   // target path
	Direction string     // "" or "forward" = ->, "reverse" = <-, "bidirectional" = <->
	Label     *TextValue // shorthand label
	Block     *Block     // optional block with metadata/children
}

func (*EdgeDecl) statementNode() {}
func (e *EdgeDecl) GetLine() int { return e.Line }

// Comment represents a # comment line.
type Comment struct {
	Line int
	Text string
}

func (*Comment) statementNode() {}
func (c *Comment) GetLine() int { return c.Line }

// Block holds metadata directives and child statements within { }.
type Block struct {
	Line     int
	Metadata []Metadata
	Children []Statement
}

// TextValue represents a text/label value.
type TextValue struct {
	Format string // "plain", "markdown", "latex" (default: "plain")
	Value  string
}

// Metadata is the interface for @ directives.
type Metadata interface {
	metadataNode()
	GetLine() int
}

// LayoutMeta holds @layout data.
type LayoutMeta struct {
	Line   int
	Values map[string]interface{}
}

func (*LayoutMeta) metadataNode()  {}
func (m *LayoutMeta) GetLine() int { return m.Line }

// StyleMeta holds @style data.
type StyleMeta struct {
	Line   int
	Values map[string]interface{}
}

func (*StyleMeta) metadataNode()  {}
func (m *StyleMeta) GetLine() int { return m.Line }

// ShapeMeta holds @shape data.
type ShapeMeta struct {
	Line  int
	Value string
}

func (*ShapeMeta) metadataNode()  {}
func (m *ShapeMeta) GetLine() int { return m.Line }

// TextMeta holds @text data.
type TextMeta struct {
	Line   int
	Format string // "plain", "markdown", "latex"
	Value  string
}

func (*TextMeta) metadataNode()  {}
func (m *TextMeta) GetLine() int { return m.Line }

// EdgeMeta holds @edge routing data.
type EdgeMeta struct {
	Line   int
	Values map[string]interface{}
}

func (*EdgeMeta) metadataNode()  {}
func (m *EdgeMeta) GetLine() int { return m.Line }

// IconMeta holds @icon data.
type IconMeta struct {
	Line   int
	Values map[string]interface{}
}

func (*IconMeta) metadataNode()  {}
func (m *IconMeta) GetLine() int { return m.Line }
