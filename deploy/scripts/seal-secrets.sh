#!/usr/bin/env bash
set -euo pipefail

# Seal platform secrets from a .env file using kubeseal.
#
# Usage:
#   1. Copy secrets.env.example to secrets.env
#   2. Fill in real values
#   3. Run: bash deploy/scripts/seal-secrets.sh
#
# Output: deploy/sealed-secrets/*.yaml (committed to Git)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUTPUT_DIR="$REPO_ROOT/deploy/sealed-secrets"
ENV_FILE="$REPO_ROOT/deploy/scripts/secrets.env"
NAMESPACE="platform-system"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: $ENV_FILE not found. Copy secrets.env.example and fill in values."
  exit 1
fi

command -v kubeseal >/dev/null 2>&1 || { echo "ERROR: kubeseal not installed"; exit 1; }

mkdir -p "$OUTPUT_DIR"

# --- platform-secrets ---
echo "Sealing platform-secrets..."
kubectl create secret generic platform-secrets \
  --namespace "$NAMESPACE" \
  --from-env-file=<(grep -E '^(JWT_SECRET|S3_ACCESS_KEY|S3_SECRET_KEY|GITHUB_OAUTH_SECRET|GOOGLE_OAUTH_SECRET)=' "$ENV_FILE") \
  --dry-run=client -o yaml \
  | kubeseal --format yaml \
  > "$OUTPUT_DIR/platform-secrets.yaml"

# --- gotrue-jwt-keys (multi-line JWK Set) ---
JWK_FILE=$(grep '^GOTRUE_JWK_SET_FILE=' "$ENV_FILE" | cut -d= -f2- || true)
if [[ -n "$JWK_FILE" && -f "$JWK_FILE" ]]; then
  echo "Sealing gotrue-jwt-keys..."
  kubectl create secret generic gotrue-jwt-keys \
    --namespace "$NAMESPACE" \
    --from-file=jwk-set="$JWK_FILE" \
    --dry-run=client -o yaml \
    | kubeseal --format yaml \
    > "$OUTPUT_DIR/gotrue-jwt-keys.yaml"
fi

# --- cloudflare-api-token ---
CF_TOKEN=$(grep '^CLOUDFLARE_API_TOKEN=' "$ENV_FILE" | cut -d= -f2-)
if [[ -n "$CF_TOKEN" ]]; then
  echo "Sealing cloudflare-api-token..."
  kubectl create secret generic cloudflare-api-token \
    --namespace cert-manager \
    --from-literal=api-token="$CF_TOKEN" \
    --dry-run=client -o yaml \
    | kubeseal --format yaml \
    > "$OUTPUT_DIR/cloudflare-api-token.yaml"
fi

# --- discord-webhook ---
DISCORD_URL=$(grep '^DISCORD_WEBHOOK_URL=' "$ENV_FILE" | cut -d= -f2- || true)
if [[ -n "$DISCORD_URL" ]]; then
  echo "Sealing discord-webhook..."
  kubectl create secret generic discord-webhook \
    --namespace monitoring \
    --from-literal=url="$DISCORD_URL" \
    --dry-run=client -o yaml \
    | kubeseal --format yaml \
    > "$OUTPUT_DIR/discord-webhook.yaml"
fi

# --- ghcr-pull-secret (container registry auth) ---
GHCR_USER=$(grep '^GHCR_USERNAME=' "$ENV_FILE" | cut -d= -f2- || true)
GHCR_TOKEN=$(grep '^GHCR_TOKEN=' "$ENV_FILE" | cut -d= -f2- || true)
if [[ -n "$GHCR_USER" && -n "$GHCR_TOKEN" ]]; then
  echo "Sealing ghcr-pull-secret..."
  kubectl create secret docker-registry ghcr-pull-secret \
    --namespace "$NAMESPACE" \
    --docker-server=ghcr.io \
    --docker-username="$GHCR_USER" \
    --docker-password="$GHCR_TOKEN" \
    --dry-run=client -o yaml \
    | kubeseal --format yaml \
    > "$OUTPUT_DIR/ghcr-pull-secret.yaml"
fi

echo ""
echo "Done. Sealed secrets written to: deploy/sealed-secrets/"
echo "Next: git add deploy/sealed-secrets/ && git push"
