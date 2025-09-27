package gomponentsconverter

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Serve   ServeCmd   `cmd:"" help:"Run converter as a server"`
	Convert ConvertCmd `cmd:"" default:"" help:"Convert stdin to stdout"`
}

type ServeCmd struct {
	Port int `flag:"" name:"port" default:"3333" help:"Port to serve HTTP"`
}

func (s *ServeCmd) Run(ctx context.Context, cli *CLI) error {
	port := cli.Serve.Port
	if port < 1 {
		if val, ok := os.LookupEnv("HTTP_PORT"); ok {
			port, _ = strconv.Atoi(val)
		}
	}
	if port < 1 {
		port = 3333
	}
	return Serve(ctx, port)
}

type ConvertCmd struct{}

func (s *ConvertCmd) Run(ctx context.Context) error {
	result, err := Convert(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Conversion error: %s\n", err)
		return err
	}

	fmt.Fprintln(os.Stdout, result)
	return nil
}

func Run(ctx context.Context, w io.Writer) error {
	cli := &CLI{}
	kongCtx := kong.Parse(cli)

	logger := slog.New(slog.NewTextHandler(
		w,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	))
	slog.SetDefault(logger)

	kongCtx.Bind(&cli)
	kongCtx.BindTo(ctx, (*context.Context)(nil))
	return kongCtx.Run(ctx)
}
