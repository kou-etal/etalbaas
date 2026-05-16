FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /nats-sidecar ./cmd/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /nats-sidecar /nats-sidecar
USER 65532:65532
ENTRYPOINT ["/nats-sidecar"]
