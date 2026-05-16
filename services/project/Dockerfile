FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY proto/go.mod proto/go.sum ./proto/
COPY services/project/go.mod services/project/go.sum ./services/project/
RUN cd services/project && go mod download
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY services/project/ ./services/project/
RUN cd services/project && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app ./cmd/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app /app
USER 65532:65532
ENTRYPOINT ["/app"]
