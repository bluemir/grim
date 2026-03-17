package guide

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/alecthomas/kingpin/v2"

	"github.com/bluemir/grim/assets"
)

type manifest struct {
	Sections []section `json:"sections"`
}

type section struct {
	Title string `json:"title"`
	Items []item `json:"items"`
}

type item struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

func Register(cmd *kingpin.CmdClause) {
	cmd.Action(func(*kingpin.ParseContext) error {
		return run()
	})
}

func run() error {
	data, err := fs.ReadFile(assets.GuideFS, "guide/manifest.json")
	if err != nil {
		return err
	}

	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	var parts []string
	for _, sec := range m.Sections {
		for _, it := range sec.Items {
			content, err := fs.ReadFile(assets.GuideFS, "guide/"+it.Slug+".md")
			if err != nil {
				return err
			}
			parts = append(parts, strings.TrimSpace(string(content)))
		}
	}

	fmt.Println(strings.Join(parts, "\n\n---\n\n"))
	return nil
}
