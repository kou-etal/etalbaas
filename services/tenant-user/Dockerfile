FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY proto/go.mod proto/go.sum ./proto/
COPY services/tenant-user/go.mod services/tenant-user/go.sum ./services/tenant-user/
RUN cd services/tenant-user && go mod download
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY services/tenant-user/ ./services/tenant-user/
RUN cd services/tenant-user && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app ./cmd/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app /app
USER 65532:65532
ENTRYPOINT ["/app"]
