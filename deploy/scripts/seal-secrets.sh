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

# Strip Windows carriage returns (CRLF → LF) to avoid \r in secret values.
ENV_CLEAN="$(mktemp)"
tr -d '\r' < "$ENV_FILE" > "$ENV_CLEAN"
trap 'rm -f "$ENV_CLEAN"' EXIT

# --- platform-secrets ---
# Map secrets.env key names to the key names expected by Helm templates.
S3_AK=$(grep '^S3_ACCESS_KEY_ID=' "$ENV_CLEAN" | cut -d= -f2- || true)
S3_SK=$(grep '^S3_SECRET_ACCESS_KEY=' "$ENV_CLEAN" | cut -d= -f2- || true)
JWT=$(grep '^JWT_SECRET=' "$ENV_CLEAN" | cut -d= -f2- || true)
GH_OAUTH=$(grep '^GITHUB_OAUTH_SECRET=' "$ENV_CLEAN" | cut -d= -f2- || true)
GO_OAUTH=$(grep '^GOOGLE_OAUTH_SECRET=' "$ENV_CLEAN" | cut -d= -f2- || true)

echo "Sealing platform-secrets..."
PLATFORM_ARGS=(
  --namespace "$NAMESPACE"
  --from-literal=JWT_SECRET="$JWT"
)
[[ -z "$S3_AK" ]] || PLATFORM_ARGS+=(--from-literal=S3_ACCESS_KEY="$S3_AK")
[[ -z "$S3_SK" ]] || PLATFORM_ARGS+=(--from-literal=S3_SECRET_KEY="$S3_SK")
[[ -z "$GH_OAUTH" ]] || PLATFORM_ARGS+=(--from-literal=GITHUB_OAUTH_SECRET="$GH_OAUTH")
[[ -z "$GO_OAUTH" ]] || PLATFORM_ARGS+=(--from-literal=GOOGLE_OAUTH_SECRET="$GO_OAUTH")

kubectl create secret generic platform-secrets \
  "${PLATFORM_ARGS[@]}" \
  --dry-run=client -o yaml \
  | kubeseal --format yaml \
  > "$OUTPUT_DIR/platform-secrets.yaml"

# --- gotrue-jwt-keys (multi-line JWK Set) ---
JWK_FILE=$(grep '^GOTRUE_JWK_SET_FILE=' "$ENV_CLEAN" | cut -d= -f2- || true)
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
CF_TOKEN=$(grep '^CLOUDFLARE_API_TOKEN=' "$ENV_CLEAN" | cut -d= -f2-)
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
DISCORD_URL=$(grep '^DISCORD_WEBHOOK_URL=' "$ENV_CLEAN" | cut -d= -f2- || true)
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
GHCR_USER=$(grep '^GHCR_USERNAME=' "$ENV_CLEAN" | cut -d= -f2- || true)
GHCR_TOKEN=$(grep '^GHCR_TOKEN=' "$ENV_CLEAN" | cut -d= -f2- || true)
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

# --- r2-backup-creds (CNPG barman S3 backup) ---
R2_AK=$(grep '^S3_BACKUP_ACCESS_KEY_ID=' "$ENV_CLEAN" | cut -d= -f2- || true)
R2_SK=$(grep '^S3_BACKUP_SECRET_ACCESS_KEY=' "$ENV_CLEAN" | cut -d= -f2- || true)
if [[ -n "$R2_AK" && -n "$R2_SK" ]]; then
  echo "Sealing r2-backup-creds..."
  kubectl create secret generic r2-backup-creds \
    --namespace "$NAMESPACE" \
    --from-literal=ACCESS_KEY_ID="$R2_AK" \
    --from-literal=SECRET_ACCESS_KEY="$R2_SK" \
    --dry-run=client -o yaml \
    | kubeseal --format yaml \
    > "$OUTPUT_DIR/r2-backup-creds.yaml"
fi

echo ""
echo "Done. Sealed secrets written to: deploy/sealed-secrets/"
echo "Next: git add deploy/sealed-secrets/ && git push"
