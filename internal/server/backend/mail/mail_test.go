package mail

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	netmail "net/mail"
	"strings"
	"sync"
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

func TestNewRejectsInvalidFrom(t *testing.T) {
	_, err := New(&Config{SMTP: SMTPConfig{Host: "smtp.example.com", From: "grim <not an address"}})
	assert.ErrorContains(t, err, "mail.smtp.from")
}

func TestValidate(t *testing.T) {
	ok := Message{To: "a@example.com", Subject: "[grim] 인증", Body: "첫 줄\n둘째 줄\n"}
	assert.NoError(t, ok.Validate())

	for name, msg := range map[string]Message{
		"recipient with name":  {To: "A <a@example.com>", Subject: "s", Body: "b"},
		"recipient injection":  {To: "a@example.com\r\nBcc: x@example.com", Subject: "s", Body: "b"},
		"recipient not parsed": {To: "not-an-address", Subject: "s", Body: "b"},
		"recipient padded":     {To: " a@example.com", Subject: "s", Body: "b"},
		"subject injection":    {To: "a@example.com", Subject: "s\r\nBcc: x@example.com", Body: "b"},
		"subject newline":      {To: "a@example.com", Subject: "s\nx", Body: "b"},
		"body CRLF":            {To: "a@example.com", Subject: "s", Body: "a\r\nb"},
		"body lone CR":         {To: "a@example.com", Subject: "s", Body: "a\rb"},
	} {
		assert.ErrorIs(t, msg.Validate(), ErrInvalidMessage, name)
	}
}

func TestSendValidatesBeforeConnecting(t *testing.T) {
	// Port 1 on localhost refuses connections, so reaching the network would
	// fail with a dial error instead of ErrInvalidMessage.
	s, err := New(&Config{SMTP: SMTPConfig{Host: "127.0.0.1", Port: 1, From: "grim@example.com", TLS: "none"}})
	require.NoError(t, err)
	err = s.Send(context.Background(), Message{To: "a@example.com", Subject: "s\nBcc: x@example.com", Body: "b"})
	assert.ErrorIs(t, err, ErrInvalidMessage)
}

// smtpSink is a minimal in-process SMTP server that records what it receives.
type smtpSink struct {
	addr     *net.TCPAddr
	mu       sync.Mutex
	commands []string
	data     []string // raw DATA payloads, as sent on the wire
}

func newSMTPSink(t *testing.T) *smtpSink {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	sink := &smtpSink{addr: ln.Addr().(*net.TCPAddr)}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go sink.serve(conn)
		}
	}()
	return sink
}

func (s *smtpSink) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	reply := func(line string) { io.WriteString(conn, line+"\r\n") }

	reply("220 sink ready")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.TrimRight(line, "\r\n")
		s.mu.Lock()
		s.commands = append(s.commands, cmd)
		s.mu.Unlock()

		switch verb := strings.ToUpper(strings.SplitN(cmd, " ", 2)[0]); verb {
		case "EHLO":
			reply("250-sink")
			reply("250 8BITMIME")
		case "DATA":
			reply("354 end with <CRLF>.<CRLF>")
			var b strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				b.WriteString(l)
			}
			s.mu.Lock()
			s.data = append(s.data, b.String())
			s.mu.Unlock()
			reply("250 queued")
		case "QUIT":
			reply("221 bye")
			return
		default:
			reply("250 ok")
		}
	}
}

func (s *smtpSink) received() ([]string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...), append([]string(nil), s.data...)
}

func TestSendDeliversOverSMTP(t *testing.T) {
	sink := newSMTPSink(t)
	s, err := New(&Config{SMTP: SMTPConfig{
		Host: "127.0.0.1",
		Port: sink.addr.Port,
		From: "그림 <grim@example.com>",
		TLS:  "none",
	}})
	require.NoError(t, err)

	err = s.Send(context.Background(), Message{
		To:      "a@example.com",
		Subject: "[grim] 다이어그램 보관 메일 인증",
		Body:    "첫 줄\n.점으로 시작하는 줄\n마지막 줄\n",
	})
	require.NoError(t, err)

	commands, data := sink.received()
	assert.Contains(t, commands, "MAIL FROM:<grim@example.com> BODY=8BITMIME")
	assert.Contains(t, commands, "RCPT TO:<a@example.com>")
	require.Len(t, data, 1)
	wire := data[0]

	// SMTP line rules: every line ends with CRLF, no bare CR or LF.
	assert.NotContains(t, strings.ReplaceAll(wire, "\r\n", ""), "\n", "no bare LF on the wire")
	assert.NotContains(t, strings.ReplaceAll(wire, "\r\n", ""), "\r", "no bare CR on the wire")

	// Undo dot-stuffing and parse what the recipient's server would store.
	parsed, err := netmail.ReadMessage(strings.NewReader(strings.ReplaceAll(wire, "\r\n..", "\r\n.")))
	require.NoError(t, err)

	from, err := parsed.Header.AddressList("From") // decodes RFC 2047 names
	require.NoError(t, err)
	assert.Equal(t, []*netmail.Address{{Name: "그림", Address: "grim@example.com"}}, from)
	to, err := parsed.Header.AddressList("To")
	require.NoError(t, err)
	assert.Equal(t, []*netmail.Address{{Address: "a@example.com"}}, to)
	subject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	require.NoError(t, err)
	assert.Equal(t, "[grim] 다이어그램 보관 메일 인증", subject)
	assert.NotEmpty(t, parsed.Header.Get("Date"))
	assert.Contains(t, parsed.Header.Get("Content-Type"), "text/plain")

	body, err := io.ReadAll(parsed.Body)
	require.NoError(t, err)
	decodedBody := decodeTransfer(t, parsed.Header.Get("Content-Transfer-Encoding"), string(body))
	assert.Equal(t, "첫 줄\r\n.점으로 시작하는 줄\r\n마지막 줄\r\n", decodedBody)
}

func TestSendFailsWhenServerUnreachable(t *testing.T) {
	s, err := New(&Config{SMTP: SMTPConfig{Host: "127.0.0.1", Port: 1, From: "grim@example.com", TLS: "none"}})
	require.NoError(t, err)
	err = s.Send(context.Background(), Message{To: "a@example.com", Subject: "s", Body: "b"})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrInvalidMessage)
}

// decodeTransfer undoes the Content-Transfer-Encoding go-mail chose for the body.
func decodeTransfer(t *testing.T, encoding, body string) string {
	t.Helper()
	switch strings.ToLower(encoding) {
	case "quoted-printable":
		b, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(body)))
		require.NoError(t, err)
		return string(b)
	case "base64":
		b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(body, "\r\n", ""))
		require.NoError(t, err)
		return string(b)
	default:
		return body
	}
}
