FROM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o operator ./cmd/

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/operator .
USER 65532:65532
ENTRYPOINT ["/operator"]
