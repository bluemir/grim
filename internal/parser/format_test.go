package parser

import (
	"testing"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// --- Node declarations ---
		{
			name:  "simple node",
			input: "myNode",
			want:  "myNode\n",
		},
		{
			name:  "dotted path node",
			input: "parent.child",
			want:  "parent.child\n",
		},
		{
			name:  "node with quoted label",
			input: `myNode: "hello"`,
			want:  "myNode: \"hello\"\n",
		},
		{
			name:  "node with plain text label (normalized to quoted)",
			input: "myNode: hello world",
			want:  "myNode: \"hello world\"\n",
		},
		{
			name:  "node with empty block",
			input: "myNode {}",
			want:  "myNode {\n}\n",
		},

		// --- Edge declarations ---
		{
			name:  "forward edge",
			input: "A -> B",
			want:  "A -> B\n",
		},
		{
			name:  "reverse edge",
			input: "A <- B",
			want:  "A <- B\n",
		},
		{
			name:  "bidirectional edge",
			input: "A <-> B",
			want:  "A <-> B\n",
		},
		{
			name:  "edge with label",
			input: `A -> B: "calls"`,
			want:  "A -> B: \"calls\"\n",
		},
		{
			name:  "dotted path edge",
			input: "parent.A -> parent.B",
			want:  "parent.A -> parent.B\n",
		},

		// --- Comments ---
		{
			name:  "comment preserved",
			input: "# this is a comment",
			want:  "# this is a comment\n",
		},
		{
			name:  "comment whitespace normalized",
			input: "#   extra spaces",
			want:  "# extra spaces\n",
		},
		{
			name:  "comment with no space",
			input: "#noSpace",
			want:  "# noSpace\n",
		},

		// --- Metadata ---
		{
			name:  "shape metadata",
			input: "n {\n\t@shape diamond\n}",
			want:  "n {\n\t@shape diamond\n}\n",
		},
		{
			name:  "layout metadata key order",
			input: "n {\n\t@layout {h: 50, w: 100, y: 200, x: 100}\n}",
			want:  "n {\n\t@layout {x: 100, y: 200, w: 100, h: 50}\n}\n",
		},
		{
			name:  "style metadata key order",
			input: "n {\n\t@style {stroke: black, fill: blue, font-size: 14, font-color: white}\n}",
			want:  "n {\n\t@style {font-color: white, font-size: 14, fill: blue, stroke: black}\n}\n",
		},
		{
			name:  "edge metadata",
			input: "A -> B {\n\t@edge {gap: 10, anchors: [\"top\", \"bottom\"]}\n}",
			want:  "A -> B {\n\t@edge {anchors: [top, bottom], gap: 10}\n}\n",
		},
		{
			name: "text metadata plain",
			input: `n {
	@text "hello world"
}`,
			want: "n {\n\t@text \"hello world\"\n}\n",
		},
		{
			name: "text metadata markdown",
			input: `n {
	@text[markdown] "**bold**"
}`,
			want: "n {\n\t@text[markdown] \"**bold**\"\n}\n",
		},
		{
			name: "text metadata multiline raw block",
			input: `n {
	@text[markdown] {|
		## header

		- list
			- list2
	|}
}`,
			want: "n {\n\t@text[markdown] {|\n\t\t## header\n\n\t\t- list\n\t\t\t- list2\n\t|}\n}\n",
		},

		// --- Nested blocks ---
		{
			name: "nested node in block",
			input: `parent {
	child {
		@shape circle
	}
}`,
			want: "parent {\n\tchild {\n\t\t@shape circle\n\t}\n}\n",
		},
		{
			name: "metadata before children",
			input: `parent {
	child
	@shape diamond
}`,
			want: "parent {\n\t@shape diamond\n\tchild\n}\n",
		},

		// --- Blank lines ---
		{
			name:  "blank line between nodes",
			input: "A\n\nB",
			want:  "A\n\nB\n",
		},
		{
			name:  "multiple blank lines collapsed",
			input: "A\n\n\n\nB",
			want:  "A\n\nB\n",
		},
		{
			name:  "blank lines not added at document start",
			input: "\n\nA",
			want:  "A\n",
		},
		{
			name:  "trailing blank lines stripped",
			input: "A\n\n",
			want:  "A\n",
		},

		// --- Mixed ---
		{
			name: "comment and node and edge",
			input: `# section header
server
client
server -> client: "request"`,
			want: "# section header\nserver\nclient\nserver -> client: \"request\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			got := Format(doc)
			if got != tt.want {
				t.Errorf("Format() mismatch:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestFormatIdempotent(t *testing.T) {
	inputs := []string{
		"myNode",
		`myNode: "label"`,
		"A -> B",
		"A <- B",
		"A <-> B",
		`A -> B: "label"`,
		"# comment",
		"n {\n\t@shape diamond\n}",
		"n {\n\t@layout {x: 100, y: 200, w: 300, h: 150}\n}",
		"n {\n\t@style {font-color: white, fill: blue}\n}",
		"n {\n\t@text \"hello\"\n}",
		"n {\n\t@text[markdown] \"**bold**\"\n}",
		"n {\n\t@text[markdown] {|\n\t\t## header\n\n\t\t- list\n\t\t\t- list2\n\t|}\n}\n",
		"parent {\n\tchild\n}\n",
		"A\n\nB",
		"# comment\nA\n\nB -> C\n",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			doc1, _ := Parse(input)
			formatted1 := Format(doc1)

			doc2, _ := Parse(formatted1)
			formatted2 := Format(doc2)

			if formatted1 != formatted2 {
				t.Errorf("Not idempotent for %q:\nfirst:  %q\nsecond: %q", input, formatted1, formatted2)
			}
		})
	}
}
