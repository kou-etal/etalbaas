FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /cdc-pod ./cmd/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /cdc-pod /cdc-pod
USER 65532:65532
ENTRYPOINT ["/cdc-pod"]
