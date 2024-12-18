package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	echov1 "github.com/e-flux-platform/echo-grpc/gen/go/road/echo/v1"
)

func main() {
	var listenAddr string

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "listen-addr",
				EnvVars:     []string{"LISTEN_ADDR"},
				Destination: &listenAddr,
				Required:    true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			ctx, cancel := signal.NotifyContext(cCtx.Context, syscall.SIGTERM, syscall.SIGINT)
			defer cancel()

			return run(ctx, listenAddr)
		},
	}

	if err := app.RunContext(context.Background(), os.Args); err != nil {
		slog.Error("exiting", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, listenAddr string) error {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	srv := grpc.NewServer()
	echov1.RegisterEchoServiceServer(srv, &server{})
	reflection.Register(srv)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return srv.Serve(lis)
	})
	eg.Go(func() error {
		<-ctx.Done()
		srv.GracefulStop()
		return nil
	})
	return eg.Wait()
}

type server struct{}

func (s *server) Echo(ctx context.Context, request *echov1.EchoRequest) (*echov1.EchoResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.DataLoss, "failed to get metadata")
	}

	items := make([]*echov1.MetadataItem, 0, len(md))
	for key, values := range md {
		items = append(items, &echov1.MetadataItem{
			Key:    key,
			Values: values,
		})
	}

	return &echov1.EchoResponse{
		Message: request.Message,
		Metadata: &echov1.Metadata{
			Items: items,
		},
	}, nil
}
