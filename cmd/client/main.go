package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	echov1 "github.com/e-flux-platform/echo-grpc/gen/go/road/echo/v1"
)

func main() {
	var (
		serverAddr string
		useTLS     bool
	)

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "server-addr",
				EnvVars:     []string{"SERVER_ADDR"},
				Destination: &serverAddr,
				Required:    true,
			},
			&cli.BoolFlag{
				Name:        "tls",
				EnvVars:     []string{"TLS"},
				Destination: &useTLS,
			},
		},
		Action: func(cCtx *cli.Context) error {
			ctx, cancel := signal.NotifyContext(cCtx.Context, syscall.SIGTERM, syscall.SIGINT)
			defer cancel()

			return run(ctx, serverAddr, useTLS, cCtx.Args().First())
		},
	}

	if err := app.RunContext(context.Background(), os.Args); err != nil {
		slog.Error("exiting", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, serverAddr string, useTLS bool, message string) error {
	var transportCredentials credentials.TransportCredentials
	if useTLS {
		rootCAs, _ := x509.SystemCertPool()
		tlsConfig := &tls.Config{
			RootCAs: rootCAs,
		}
		transportCredentials = credentials.NewTLS(tlsConfig)
	} else {
		transportCredentials = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(transportCredentials))
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
