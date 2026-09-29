package util

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseDuration(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"90d":   90 * 24 * time.Hour,
		"1d12h": 36 * time.Hour,
		"720h":  720 * time.Hour,
		"30m":   30 * time.Minute,
		"3600":  time.Hour,
		"0":     0,
	} {
		got, err := ParseDuration(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got.Std(), in)
	}
	_, err := ParseDuration("soon")
	assert.Error(t, err)
}

func TestParseSize(t *testing.T) {
	for in, want := range map[string]Size{
		"1MiB":   1 << 20,
		"512KiB": 512 << 10,
		"64KB":   64 << 10,
		"2M":     2 << 20,
		"100":    100,
	} {
		got, err := ParseSize(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	_, err := ParseSize("big")
	assert.Error(t, err)
}

func TestUnitsFromConfigFormats(t *testing.T) {
	type conf struct {
		Retention Duration
		MaxSize   Size `yaml:"maxSize"`
	}

	var h conf
	require.NoError(t, hjson.Unmarshal([]byte("{\n  retention: 30d\n  maxSize: 1MiB\n}"), &h))
	assert.Equal(t, 30*24*time.Hour, h.Retention.Std())
	assert.Equal(t, Size(1<<20), h.MaxSize)

	var y conf
	require.NoError(t, yaml.Unmarshal([]byte("retention: 30d\nmaxSize: 1MiB\n"), &y))
	assert.Equal(t, h, y)

	var j conf
	require.NoError(t, json.Unmarshal([]byte(`{"retention": 86400, "maxSize": 2048}`), &j))
	assert.Equal(t, 24*time.Hour, j.Retention.Std())
	assert.Equal(t, Size(2048), j.MaxSize)
}
