package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	echov1 "github.com/e-flux-platform/echo-grpc/gen/go/road/echo/v1"
)

type config struct {
	serverAddr   string
	useTLS       bool
	clientRootCA string
	clientKey    string
	clientCert   string
}

func main() {
	var cfg config

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "server-addr",
				EnvVars:     []string{"SERVER_ADDR"},
				Destination: &cfg.serverAddr,
				Required:    true,
			},
			&cli.BoolFlag{
				Name:        "tls",
				EnvVars:     []string{"TLS"},
				Destination: &cfg.useTLS,
			},
			&cli.StringFlag{
				Name:        "client-cert",
				EnvVars:     []string{"CLIENT_CERT"},
				Destination: &cfg.clientCert,
			},
			&cli.StringFlag{
				Name:        "client-key",
				EnvVars:     []string{"CLIENT_KEY"},
				Destination: &cfg.clientKey,
			},
			&cli.StringFlag{
				Name:        "client-root-ca",
				EnvVars:     []string{"CLIENT_ROOT_CA"},
				Destination: &cfg.clientRootCA,
			},
		},
		Action: func(cCtx *cli.Context) error {
			ctx, cancel := signal.NotifyContext(cCtx.Context, syscall.SIGTERM, syscall.SIGINT)
			defer cancel()

			return run(ctx, cfg, cCtx.Args().First())
		},
	}

	if err := app.RunContext(context.Background(), os.Args); err != nil {
		slog.Error("exiting", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config, message string) error {
	var transportCredentials credentials.TransportCredentials
	if cfg.useTLS {
		systemCertPool, _ := x509.SystemCertPool()
		tlsConfig := &tls.Config{
			RootCAs: systemCertPool,
		}

		if cfg.clientKey != "" && cfg.clientCert != "" {
			cert, err := tls.LoadX509KeyPair(cfg.clientCert, cfg.clientKey)
			if err != nil {
				return err
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		if cfg.clientRootCA != "" {
			clientCA, err := os.ReadFile(cfg.clientRootCA)
			if err != nil {
				return err
			}
			clientCertPool := x509.NewCertPool()
			clientCertPool.AppendCertsFromPEM(clientCA)
			tlsConfig.ClientCAs = clientCertPool
		}

		transportCredentials = credentials.NewTLS(tlsConfig)
	} else {
		transportCredentials = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(cfg.serverAddr, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := echov1.NewEchoServiceClient(conn)

	res, err := client.Echo(ctx, &echov1.EchoRequest{Message: message})
	if err != nil {
		return err
	}

	fmt.Printf("response from server: %s\n", res.Message)

	fmt.Println("metadata from server:")
	for _, item := range res.Metadata.Items {
		fmt.Printf("%s: %s\n", item.Key, strings.Join(item.Values, ", "))
	}

	return nil
}
