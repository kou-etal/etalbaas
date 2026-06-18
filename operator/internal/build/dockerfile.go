package build

import (
	"fmt"
	"regexp"
	"strings"
)

// validPyPkgRe matches PyPI package names with optional version specifiers (PEP 508 simplified).
var validPyPkgRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?(\[([a-zA-Z0-9._-]+,?)+\])?(([=!<>~]=?|===?)[a-zA-Z0-9.*]+)?$`)

// validNpmPkgRe matches npm package names with optional version/tag specifiers.
var validNpmPkgRe = regexp.MustCompile(`^(@[a-zA-Z0-9._-]+/)?[a-zA-Z0-9._-]+(@[a-zA-Z0-9._^~>=<|\-]+)?$`)

func validateRequirements(reqs []string, re *regexp.Regexp) error {
	for _, r := range reqs {
		if !re.MatchString(r) {
			return fmt.Errorf("invalid package name: %s", r)
		}
	}
	return nil
}

// GenerateDockerfile generates a Dockerfile for the given runtime preset and requirements.
func GenerateDockerfile(preset string, requirements []string, customDockerfile string) (string, error) {
	switch preset {
	case "python-3.11":
		if err := validateRequirements(requirements, validPyPkgRe); err != nil {
			return "", err
		}
		return pythonDockerfile("3.11", requirements, false), nil
	case "python-3.11-ml":
		if err := validateRequirements(requirements, validPyPkgRe); err != nil {
			return "", err
		}
		return pythonDockerfile("3.11", requirements, true), nil
	case "python-3.12":
		if err := validateRequirements(requirements, validPyPkgRe); err != nil {
			return "", err
		}
		return pythonDockerfile("3.12", requirements, false), nil
	case "node-20":
		if err := validateRequirements(requirements, validNpmPkgRe); err != nil {
			return "", err
		}
		return nodeDockerfile("20", requirements), nil
	case "node-22":
		if err := validateRequirements(requirements, validNpmPkgRe); err != nil {
			return "", err
		}
		return nodeDockerfile("22", requirements), nil
	case "go-1.22":
		return goDockerfile("1.22"), nil
	case "custom":
		if customDockerfile == "" {
			return "", fmt.Errorf("custom preset requires a Dockerfile")
		}
		if err := validateCustomDockerfile(customDockerfile); err != nil {
			return "", err
		}
		return customDockerfile, nil
	default:
		return "", fmt.Errorf("unsupported runtime preset: %s", preset)
	}
}

const maxCustomDockerfileSize = 10 * 1024 // 10 KB

// validateCustomDockerfile performs basic safety checks on user-provided Dockerfiles.
// Note: Pod SecurityContext enforces runAsNonRoot + runAsUser=65532, so even if
// the Dockerfile does not set USER, the container runs as non-root.
func validateCustomDockerfile(content string) error {
	if len(content) > maxCustomDockerfileSize {
		return fmt.Errorf("custom Dockerfile exceeds maximum size of %d bytes", maxCustomDockerfileSize)
	}
	// Must contain at least one FROM instruction.
	hasFrom := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), "FROM ") {
			hasFrom = true
			break
		}
	}
	if !hasFrom {
		return fmt.Errorf("custom Dockerfile must contain a FROM instruction")
	}
	return nil
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

// NodeRuntimeWrapper is a minimal HTTP server that imports the user's handler
// and serves it on port 8080. Supports ESM default exports.
// Added to the source ConfigMap as index.js when not provided by the user.
const NodeRuntimeWrapper = `import { createServer } from 'node:http';
import { readdir } from 'node:fs/promises';

async function findHandler() {
  const files = await readdir('.');
  const candidates = ['handler.mjs','handler.js','main.mjs','main.js','index.mjs'];
  for (const c of candidates) {
    if (files.includes(c) && c !== 'index.js') {
      const mod = await import('./' + c);
      return mod.default || mod.handler || mod;
    }
  }
  throw new Error('No handler file found. Expected handler.js or main.js');
}

const handler = await findHandler();
const PORT = process.env.PORT || 8080;

createServer(async (req, res) => {
  try {
    let body = '';
    for await (const chunk of req) body += chunk;
    const request = new Request('http://localhost' + req.url, {
      method: req.method,
      headers: req.headers,
      body: ['GET','HEAD'].includes(req.method) ? undefined : body,
    });
    const response = await handler(request);
    const text = await response.text();
    const headers = {};
    response.headers.forEach((v, k) => { headers[k] = v; });
    res.writeHead(response.status || 200, headers);
    res.end(text);
  } catch (err) {
    res.writeHead(500, {'Content-Type':'application/json'});
    res.end(JSON.stringify({error: err.message}));
  }
}).listen(PORT, () => console.log('Function listening on port ' + PORT));
`

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
