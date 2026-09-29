package mail

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDisabledWithoutHost(t *testing.T) {
	s, err := New(&Config{})
	require.NoError(t, err)
	assert.False(t, s.Enabled())
}

func TestNewValidates(t *testing.T) {
	_, err := New(&Config{SMTP: SMTPConfig{Host: "smtp.example.com"}})
	assert.ErrorContains(t, err, "from")
	_, err = New(&Config{SMTP: SMTPConfig{Host: "smtp.example.com", From: "a@example.com", TLS: "ssl"}})
	assert.ErrorContains(t, err, "tls")
}

func TestEnvelopeAddress(t *testing.T) {
	for in, want := range map[string]string{
		"grim <grim@example.com>": "grim@example.com",
		" grim@example.com ":      "grim@example.com",
	} {
		got, err := envelopeAddress(in)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

func TestFormat(t *testing.T) {
	msg := string(format("grim <grim@example.com>", Message{To: "a@example.com", Subject: "[grim] 인증", Body: "첫 줄\n둘째 줄"}))
	head, body, ok := strings.Cut(msg, "\r\n\r\n")
	require.True(t, ok)
	assert.Contains(t, head, "To: a@example.com\r\n")
	assert.Contains(t, head, "Subject: =?UTF-8?b?")
	assert.Contains(t, head, "Content-Type: text/plain; charset=UTF-8")
	assert.Equal(t, "첫 줄\r\n둘째 줄", body)
}
