package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
	"github.com/janderland/fdbq/engine/facade"
	"github.com/janderland/fdbq/internal/app/flag"
	"github.com/janderland/fdbq/internal/app/headless"
	"github.com/janderland/fdbq/parser/format"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func Run() {
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		if _, err := fmt.Fprintf(os.Stderr, "%v\n", err); err != nil {
			panic(err)
		}
		os.Exit(1)
	}
}

func run(args []string, stdout *os.File, stderr *os.File) error {
	flags, queries, err := flag.Parse(args, stderr)
	if err != nil {
		return errors.Wrap(err, "failed to parse args")
	}

	// If Parse returns nil flags, it's assumed the help
	// flag was given and the help message was printed.
	if flags == nil {
		return nil
	}

	log := zerolog.Nop()
	if flags.Log {
		writer := zerolog.ConsoleWriter{Out: stderr}
		writer.FormatLevel = func(_ interface{}) string { return "" }
		log = zerolog.New(writer).With().Timestamp().Logger()
	}

	log.Log().Str("cluster file", flags.Cluster).Msg("connecting to DB")
	if err := fdb.APIVersion(620); err != nil {
		return errors.Wrap(err, "failed to set FDB API version")
	}
	db, err := fdb.OpenDatabase(flags.Cluster)
	if err != nil {
		return errors.Wrap(err, "failed to connect to DB")
	}

	app := headless.App{
		Format: format.Format{
			Builder: &strings.Builder{},
			Cfg: format.Cfg{
				PrintBytes: flags.Bytes,
			},
		},
		Flags: *flags,
		Log:   log,
		Out:   stdout,
	}
	return app.Run(context.Background(), facade.NewTransactor(db, directory.Root()), queries)
}
