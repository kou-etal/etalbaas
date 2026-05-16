FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY proto/go.mod proto/go.sum ./proto/
COPY storage-providers/go.mod storage-providers/go.sum ./storage-providers/
COPY services/storage/go.mod services/storage/go.sum ./services/storage/
RUN cd services/storage && go mod download
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY storage-providers/ ./storage-providers/
COPY services/storage/ ./services/storage/
RUN cd services/storage && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app ./cmd/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app /app
USER 65532:65532
ENTRYPOINT ["/app"]
