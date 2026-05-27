FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY operator/go.mod operator/go.sum ./operator/
RUN cd operator && go mod download
COPY pkg/ ./pkg/
COPY operator/ ./operator/
RUN cd operator && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /operator ./cmd/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /operator /operator
USER 65532:65532
ENTRYPOINT ["/operator"]
