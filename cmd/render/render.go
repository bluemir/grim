package render

import (
	"fmt"
	"os"

	"github.com/alecthomas/kingpin/v2"

	"github.com/bluemir/grim/internal/parser"
	"github.com/bluemir/grim/internal/renderer"
)

func Register(cmd *kingpin.CmdClause) {
	var (
		filePath   string
		outputPath string
	)

	cmd.Arg("file", ".grim file path").
		Required().
		StringVar(&filePath)
	cmd.Flag("output", "output file path (default: stdout)").
		Short('o').
		StringVar(&outputPath)

	cmd.Action(func(*kingpin.ParseContext) error {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		doc, err := parser.Parse(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse: %w", err)
		}

		svg := renderer.Render(doc)

		if outputPath == "" {
			fmt.Print(svg)
			return nil
		}

		if err := os.WriteFile(outputPath, []byte(svg), 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	})
}
