package mail

import (
	"cmp"
	"context"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	gomail "github.com/wneessen/go-mail"
)

var (
	ErrMailDisabled   = errors.New("mail is disabled")
	ErrInvalidMessage = errors.New("invalid mail message")
)

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
	To      string // a bare address, without a display name
	Subject string // a single line
	Body    string // plain text; lines end with "\n", never "\r"
}

// Validate rejects messages that could inject headers or break SMTP line
// rules. Messages are built by our own code, so a violation is a bug.
func (msg Message) Validate() error {
	if addr, err := netmail.ParseAddress(msg.To); err != nil || addr.Name != "" || addr.Address != msg.To {
		return errors.Wrapf(ErrInvalidMessage, "recipient must be a bare address: %q", msg.To)
	}
	if strings.ContainsAny(msg.Subject, "\r\n") {
		return errors.Wrap(ErrInvalidMessage, "subject must be a single line")
	}
	if strings.ContainsRune(msg.Body, '\r') {
		return errors.Wrap(ErrInvalidMessage, `body lines must end with "\n" only`)
	}
	return nil
}

// Sender delivers mail. It is an interface so tests can capture messages.
type Sender interface {
	Enabled() bool
	Send(ctx context.Context, msg Message) error
}

func New(conf *Config) (Sender, error) {
	c := conf.SMTP
	if c.Host == "" {
		return disabled{}, nil
	}
	if c.From == "" {
		return nil, errors.New("mail.smtp.from is required when mail.smtp.host is set")
	}
	from, err := netmail.ParseAddress(c.From)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid mail.smtp.from: %q", c.From)
	}

	opts := []gomail.Option{gomail.WithTimeout(30 * time.Second)}
	port := c.Port
	switch c.TLS {
	case "", "starttls":
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
		port = cmp.Or(port, 587)
	case "tls":
		opts = append(opts, gomail.WithSSL())
		port = cmp.Or(port, 465)
	case "none":
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
		port = cmp.Or(port, 25)
	default:
		return nil, errors.Errorf("unknown mail.smtp.tls: %q (use starttls, tls or none)", c.TLS)
	}
	opts = append(opts, gomail.WithPort(port))
	if c.Username != "" {
		// Pick the strongest mechanism the server offers; some servers
		// (e.g. Exchange / Office 365) accept LOGIN but not PLAIN.
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
			gomail.WithUsername(c.Username),
			gomail.WithPassword(c.Password),
		)
	}
	return &smtpSender{host: c.Host, from: from, opts: opts}, nil
}

type disabled struct{}

func (disabled) Enabled() bool                       { return false }
func (disabled) Send(context.Context, Message) error { return ErrMailDisabled }

type smtpSender struct {
	host string
	from *netmail.Address
	opts []gomail.Option
}

func (s *smtpSender) Enabled() bool { return true }

func (s *smtpSender) Send(ctx context.Context, msg Message) error {
	// Reject an invalid message before connecting.
	if err := msg.Validate(); err != nil {
		return err
	}

	m := gomail.NewMsg(gomail.WithCharset(gomail.CharsetUTF8))
	if err := m.FromFormat(s.from.Name, s.from.Address); err != nil {
		return errors.WithStack(err)
	}
	if err := m.To(msg.To); err != nil {
		return errors.WithStack(err)
	}
	m.Subject(msg.Subject)
	m.SetDate()
	m.SetBodyString(gomail.TypeTextPlain, msg.Body)

	// A client per send: go-mail clients hold connection state and sends may run concurrently.
	client, err := gomail.NewClient(s.host, s.opts...)
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.Wrap(client.DialAndSendWithContext(ctx, m), "send mail")
}
