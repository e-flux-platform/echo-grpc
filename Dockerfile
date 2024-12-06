FROM golang:1.23-bookworm AS build
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /echo-grpc ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=build echo-grpc /echo-grpc
ENTRYPOINT [ "/echo-grpc" ]
