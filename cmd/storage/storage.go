package storage

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/cockroachdb/errors"
	"gorm.io/gorm"

	"github.com/bluemir/grim/internal/server"
	"github.com/bluemir/grim/internal/server/backend/mail"
	"github.com/bluemir/grim/internal/server/backend/storage"
	"github.com/bluemir/grim/internal/server/store"
	"github.com/bluemir/grim/internal/util"
)

// Register adds admin commands that operate directly on the server's
// database file, so they work without any web authentication.
func Register(cmd *kingpin.CmdClause) {
	var (
		dbPath     string
		configPath string
	)
	cmd.Flag("db-path", "db path (same as the server's --db-path)").
		Required().StringVar(&dbPath)
	cmd.Flag("config", "server config file; used for the retention period when showing expiry").
		Short('c').StringVar(&configPath)

	open := func() (*storage.Manager, error) {
		conf := server.DefaultConfig().Backend.Storage
		if configPath != "" {
			c, err := server.ReadConfigFile(configPath)
			if err != nil {
				return nil, err
			}
			conf = c.Backend.Storage
		}
		conf.Enabled = true
		conf.RateLimit.Create = 0

		db, err := store.Initialize(context.Background(), dbPath)
		if err != nil {
			return nil, err
		}
		mailer, _ := mail.New(&mail.Config{}) // admin commands never send mail
		return storage.New(context.Background(), &conf, db, mailer)
	}

	// notFound replaces gorm's stack-traced error with a one-line message.
	notFound := func(id string, err error) error {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("no diagram with id %q", id)
		}
		return err
	}

	cmd.Command("ls", "list stored diagrams").Action(func(*kingpin.ParseContext) error {
		m, err := open()
		if err != nil {
			return err
		}
		diagrams, err := m.List(context.Background())
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tCREATED\tLAST VIEWED\tEXPIRES")
		for i := range diagrams {
			d := &diagrams[i]
			expires := "never"
			if at, ok := m.ExpiresAt(d); ok {
				expires = at.Format(time.DateTime)
			}
			switch {
			case d.Pinned:
				expires += " (pinned)"
			case d.VerifiedUntil != nil:
				expires += " (verified)"
			case d.KeepUntil != nil:
				expires += " (extended)"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.Id,
				d.CreatedAt.Format(time.DateTime), d.LastViewedAt.Format(time.DateTime), expires)
		}
		return w.Flush()
	})

	pin := cmd.Command("pin", "keep a diagram forever")
	pinID := pin.Arg("id", "diagram id").Required().String()
	pin.Action(func(*kingpin.ParseContext) error {
		m, err := open()
		if err != nil {
			return err
		}
		return notFound(*pinID, m.SetPinned(context.Background(), *pinID, true))
	})

	unpin := cmd.Command("unpin", "let a pinned diagram expire again")
	unpinID := unpin.Arg("id", "diagram id").Required().String()
	unpin.Action(func(*kingpin.ParseContext) error {
		m, err := open()
		if err != nil {
			return err
		}
		return notFound(*unpinID, m.SetPinned(context.Background(), *unpinID, false))
	})

	extend := cmd.Command("extend", "keep a diagram for at least the given period from now, even if nobody views it")
	extendID := extend.Arg("id", "diagram id").Required().String()
	extendFor := extend.Arg("period", `period such as "365d" or "720h"`).Required().String()
	extend.Action(func(*kingpin.ParseContext) error {
		d, err := util.ParseDuration(*extendFor)
		if err != nil {
			return err
		}
		m, err := open()
		if err != nil {
			return err
		}
		until, err := m.Extend(context.Background(), *extendID, d.Std())
		if err != nil {
			return notFound(*extendID, err)
		}
		fmt.Printf("%s is kept until at least %s\n", *extendID, until.Format(time.DateTime))
		return nil
	})

	rm := cmd.Command("rm", "delete a diagram")
	rmID := rm.Arg("id", "diagram id").Required().String()
	rm.Action(func(*kingpin.ParseContext) error {
		m, err := open()
		if err != nil {
			return err
		}
		return notFound(*rmID, m.Delete(context.Background(), *rmID))
	})
}
