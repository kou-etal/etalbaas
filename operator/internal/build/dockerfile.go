package build

import (
	"fmt"
	"strings"
)

// GenerateDockerfile generates a Dockerfile for the given runtime preset and requirements.
func GenerateDockerfile(preset string, requirements []string, customDockerfile string) (string, error) {
	switch preset {
	case "python-3.11":
		return pythonDockerfile("3.11", requirements, false), nil
	case "python-3.11-ml":
		return pythonDockerfile("3.11", requirements, true), nil
	case "python-3.12":
		return pythonDockerfile("3.12", requirements, false), nil
	case "node-20":
		return nodeDockerfile("20", requirements), nil
	case "node-22":
		return nodeDockerfile("22", requirements), nil
	case "go-1.22":
		return goDockerfile("1.22"), nil
	case "custom":
		if customDockerfile == "" {
			return "", fmt.Errorf("custom preset requires a Dockerfile")
		}
		return customDockerfile, nil
	default:
		return "", fmt.Errorf("unsupported runtime preset: %s", preset)
	}
}

func pythonDockerfile(version string, requirements []string, ml bool) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("FROM python:%s-slim AS builder\n", version))
	b.WriteString("WORKDIR /app\n")
	b.WriteString("RUN pip install --no-cache-dir --upgrade pip\n")

	// ML preset: pre-install heavy packages
	if ml {
		b.WriteString("RUN pip install --no-cache-dir numpy pandas scikit-learn\n")
	}

	// Install user requirements
	b.WriteString("COPY requirements.txt* ./\n")
	b.WriteString("RUN if [ -f requirements.txt ]; then pip install --no-cache-dir -r requirements.txt; fi\n")

	// Install inline requirements
	if len(requirements) > 0 {
		b.WriteString(fmt.Sprintf("RUN pip install --no-cache-dir %s\n", strings.Join(requirements, " ")))
	}

	b.WriteString("COPY . .\n\n")

	b.WriteString(fmt.Sprintf("FROM python:%s-slim\n", version))
	b.WriteString("WORKDIR /app\n")
	b.WriteString("RUN adduser --disabled-password --gecos '' appuser\n")
	b.WriteString("COPY --from=builder /usr/local/lib/python*/site-packages /usr/local/lib/python*/site-packages\n")
	b.WriteString("COPY --from=builder /usr/local/bin /usr/local/bin\n")
	b.WriteString("COPY --from=builder /app /app\n")
	b.WriteString("USER appuser\n")
	b.WriteString("ENTRYPOINT [\"python\", \"main.py\"]\n")

	return b.String()
}

func nodeDockerfile(version string, requirements []string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("FROM node:%s-slim AS builder\n", version))
	b.WriteString("WORKDIR /app\n")
	b.WriteString("COPY package*.json ./\n")
	b.WriteString("RUN npm ci --only=production 2>/dev/null || true\n")

	// Install inline requirements
	if len(requirements) > 0 {
		b.WriteString(fmt.Sprintf("RUN npm install %s\n", strings.Join(requirements, " ")))
	}

	b.WriteString("COPY . .\n\n")

	b.WriteString(fmt.Sprintf("FROM node:%s-slim\n", version))
	b.WriteString("WORKDIR /app\n")
	b.WriteString("RUN adduser --disabled-password --gecos '' appuser\n")
	b.WriteString("COPY --from=builder /app /app\n")
	b.WriteString("USER appuser\n")
	b.WriteString("ENTRYPOINT [\"node\", \"index.js\"]\n")

	return b.String()
}

func goDockerfile(version string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("FROM golang:%s AS builder\n", version))
	b.WriteString("WORKDIR /app\n")
	b.WriteString("COPY go.mod go.sum* ./\n")
	b.WriteString("RUN go mod download\n")
	b.WriteString("COPY . .\n")
	b.WriteString("RUN CGO_ENABLED=0 go build -o /app/handler .\n\n")

	b.WriteString("FROM gcr.io/distroless/static:nonroot\n")
	b.WriteString("COPY --from=builder /app/handler /handler\n")
	b.WriteString("USER 65532:65532\n")
	b.WriteString("ENTRYPOINT [\"/handler\"]\n")

	return b.String()
}
