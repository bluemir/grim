package util

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"gopkg.in/yaml.v3"
)

// Duration is a time.Duration that reads human-friendly strings from config
// files: everything time.ParseDuration accepts, plus a "d" (day) unit
// such as "90d" or "1d12h". A bare number is taken as seconds.
type Duration time.Duration

var dayUnit = regexp.MustCompile(`(\d+)d`)

func ParseDuration(s string) (Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return Duration(time.Duration(n) * time.Second), nil
	}
	expanded := dayUnit.ReplaceAllStringFunc(s, func(m string) string {
		n, _ := strconv.Atoi(strings.TrimSuffix(m, "d"))
		return strconv.Itoa(n*24) + "h"
	})
	d, err := time.ParseDuration(expanded)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid duration: %q", s)
	}
	return Duration(d), nil
}

func (d Duration) Std() time.Duration { return time.Duration(d) }

// Days returns the whole number of days, rounded down.
func (d Duration) Days() int { return int(time.Duration(d) / (24 * time.Hour)) }

func (d Duration) String() string {
	td := time.Duration(d)
	if td != 0 && td%(24*time.Hour) == 0 {
		return strconv.Itoa(d.Days()) + "d"
	}
	return td.String()
}

func (d Duration) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

func (d *Duration) UnmarshalText(b []byte) error {
	v, err := ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// UnmarshalJSON accepts both a string ("90d") and a number (seconds).
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	return d.UnmarshalText([]byte(s))
}

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	return d.UnmarshalText([]byte(node.Value))
}
