package renderer

import (
	_ "embed"
	"strings"

	hjson "github.com/hjson/hjson-go/v4"
)

//go:embed material_colors.hjson
var materialColorsHJSON []byte

var materialColors map[string]string

func init() {
	if err := hjson.Unmarshal(materialColorsHJSON, &materialColors); err != nil {
		panic("failed to parse material_colors.hjson: " + err.Error())
	}
}

// resolveColor converts a color name (e.g. "blue-gray-200") to its hex value.
// If the color is already a hex/rgb/rgba value or a standard CSS color, it is returned as-is.
func resolveColor(name string) string {
	if name == "" {
		return name
	}
	// Already a hex, rgb, or rgba value
	if name[0] == '#' || strings.HasPrefix(name, "rgb") {
		return name
	}
	if hex, ok := materialColors[name]; ok {
		return hex
	}
	// Standard CSS color names (black, white, transparent, red, blue, etc.)
	return name
}
