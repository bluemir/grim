package util

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v3"
)

// Size is a byte count that reads strings like "1MiB", "512KiB", "64KB" or a
// bare number of bytes from config files. KB/MB are treated as binary units
// too, since config authors rarely mean the decimal ones.
type Size int64

var sizePattern = regexp.MustCompile(`^(\d+)\s*([KMG]?)(I?B)?$`)

func ParseSize(s string) (Size, error) {
	m := sizePattern.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(s)))
	if m == nil {
		return 0, errors.Errorf("invalid size: %q", s)
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid size: %q", s)
	}
	switch m[2] {
	case "K":
		n <<= 10
	case "M":
		n <<= 20
	case "G":
		n <<= 30
	}
	return Size(n), nil
}

func (s Size) String() string {
	switch {
	case s != 0 && s%(1<<30) == 0:
		return strconv.FormatInt(int64(s>>30), 10) + "GiB"
	case s != 0 && s%(1<<20) == 0:
		return strconv.FormatInt(int64(s>>20), 10) + "MiB"
	case s != 0 && s%(1<<10) == 0:
		return strconv.FormatInt(int64(s>>10), 10) + "KiB"
	}
	return strconv.FormatInt(int64(s), 10) + "B"
}

func (s Size) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

func (s *Size) UnmarshalText(b []byte) error {
	v, err := ParseSize(string(b))
	if err != nil {
		return err
	}
	*s = v
	return nil
}

func (s *Size) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		str = string(b)
	}
	return s.UnmarshalText([]byte(str))
}

func (s *Size) UnmarshalYAML(node *yaml.Node) error {
	return s.UnmarshalText([]byte(node.Value))
}
