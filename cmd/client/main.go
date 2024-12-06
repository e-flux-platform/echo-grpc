package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	echov1 "github.com/e-flux-platform/echo-grpc/gen/go/road/echo/v1"
)

func main() {
	var serverAddr string

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "server-addr",
				EnvVars:     []string{"SERVER_ADDR"},
				Destination: &serverAddr,
				Required:    true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			ctx, cancel := signal.NotifyContext(cCtx.Context, syscall.SIGTERM, syscall.SIGINT)
			defer cancel()

			return run(ctx, serverAddr, cCtx.Args().First())
		},
	}

	if err := app.RunContext(context.Background(), os.Args); err != nil {
		slog.Error("exiting", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, serverAddr, message string) error {
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := echov1.NewEchoServiceClient(conn)

	res, err := client.Echo(ctx, &echov1.EchoRequest{Message: message})
	if err != nil {
		return err
	}

	fmt.Printf("received from server: %s\n", res.Message)

	return nil
}
