package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

var ErrMailDisabled = errors.New("mail is disabled")

type Config struct {
	SMTP SMTPConfig `yaml:"smtp"`
}

type SMTPConfig struct {
	// Host enables mail when set.
	Host     string
	Port     int
	Username string
	Password string
	// From is the sender address, e.g. "grim <grim@example.com>".
	From string
	// TLS is "starttls" (default), "tls" (implicit TLS, usually port 465) or "none".
	TLS string `yaml:"tls"`
}

type Message struct {
	To      string
	Subject string
	Body    string // plain text
}

// Sender delivers mail. It is an interface so tests can capture messages.
type Sender interface {
	Enabled() bool
	Send(ctx context.Context, msg Message) error
}

func New(conf *Config) (Sender, error) {
	if conf.SMTP.Host == "" {
		return disabled{}, nil
	}
	switch conf.SMTP.TLS {
	case "", "starttls", "tls", "none":
	default:
		return nil, errors.Errorf("unknown mail.smtp.tls: %q (use starttls, tls or none)", conf.SMTP.TLS)
	}
	if conf.SMTP.From == "" {
		return nil, errors.New("mail.smtp.from is required when mail.smtp.host is set")
	}
	return &smtpSender{conf: conf.SMTP}, nil
}

type disabled struct{}

func (disabled) Enabled() bool                       { return false }
func (disabled) Send(context.Context, Message) error { return ErrMailDisabled }

type smtpSender struct {
	conf SMTPConfig
}

func (s *smtpSender) Enabled() bool { return true }

func (s *smtpSender) Send(ctx context.Context, msg Message) error {
	port := s.conf.Port
	if port == 0 {
		port = map[string]int{"tls": 465, "none": 25}[s.conf.TLS]
		if port == 0 {
			port = 587
		}
	}
	addr := net.JoinHostPort(s.conf.Host, strconv.Itoa(port))
	tlsConf := &tls.Config{ServerName: s.conf.Host}

	dialer := &net.Dialer{Timeout: 30 * time.Second}
	var conn net.Conn
	var err error
	if s.conf.TLS == "tls" {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConf}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return errors.Wrapf(err, "dial smtp %s", addr)
	}
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	} else {
		conn.SetDeadline(time.Now().Add(time.Minute))
	}

	c, err := smtp.NewClient(conn, s.conf.Host)
	if err != nil {
		conn.Close()
		return errors.WithStack(err)
	}
	defer c.Close()

	if s.conf.TLS == "" || s.conf.TLS == "starttls" {
		if err := c.StartTLS(tlsConf); err != nil {
			return errors.Wrap(err, "smtp starttls")
		}
	}
	if s.conf.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", s.conf.Username, s.conf.Password, s.conf.Host)); err != nil {
			return errors.Wrap(err, "smtp auth")
		}
	}

	from, err := envelopeAddress(s.conf.From)
	if err != nil {
		return err
	}
	if err := c.Mail(from); err != nil {
		return errors.WithStack(err)
	}
	if err := c.Rcpt(msg.To); err != nil {
		return errors.WithStack(err)
	}
	w, err := c.Data()
	if err != nil {
		return errors.WithStack(err)
	}
	if _, err := w.Write(format(s.conf.From, msg)); err != nil {
		return errors.WithStack(err)
	}
	if err := w.Close(); err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(c.Quit())
}

// envelopeAddress extracts the bare address from `name <addr>`.
func envelopeAddress(from string) (string, error) {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		j := strings.LastIndex(from, ">")
		if j < i {
			return "", errors.Errorf("invalid mail.smtp.from: %q", from)
		}
		return from[i+1 : j], nil
	}
	return strings.TrimSpace(from), nil
}

func format(from string, msg Message) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", msg.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.BEncoding.Encode("UTF-8", msg.Subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(strings.ReplaceAll(msg.Body, "\r\n", "\n"), "\n", "\r\n"))
	return []byte(b.String())
}
