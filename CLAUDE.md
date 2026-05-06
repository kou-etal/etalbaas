# CLAUDE.md

## Thinking policy

トークン消費を気にせず、深く考えてから回答すること。曖昧な回答より正確で網羅的な回答を優先。

## Project overview

etalbaas - GPU対応・Self-hostable なマルチテナント BaaS（k8s ネイティブ）
詳細設計: `docs/architecture.md` (v7.4)

## Tech stack

- Backend: Go + connect-go (Connect RPC)
- API schema: Protocol Buffers (buf)
- Orchestration: Kubernetes (CRD + Operator, kubebuilder)
- Auth: GoTrue (Supabase fork)
- DB: PostgreSQL + pgvector, Redis
- Storage: AWS S3
- Frontend: Next.js (Vercel)
- IaC: Helm + Terraform

## Project structure

```
etalbaas/
├── proto/                  # Protocol Buffers definitions
│   ├── buf.yaml
│   ├── buf.gen.yaml
│   └── etalbaas/
│       ├── common/v1/
│       ├── tenant/v1/
│       ├── project/v1/
│       ├── function/v1/
│       └── invoke/v1/
├── services/               # Go microservices
│   ├── gateway/
│   ├── tenant-user/
│   ├── project/
│   ├── function/
│   └── invoke/
├── operator/               # kubebuilder Operator (planned)
├── dashboard/              # Next.js frontend (planned)
├── deploy/
│   ├── helm/               # Self-host配布用 Chart
│   ├── migrations/meta/    # golang-migrate (メタDB)
│   └── argocd/             # GitOps
└── docs/
    ├── architecture.md     # 設計書 v7.4
    └── database.md         # メタDB設計
```

## Commands

- Proto generate: `cd proto && buf generate`
- Proto lint: `cd proto && buf lint`
- Go test (per service): `cd services/<name> && go test ./...`
- Go build (per service): `cd services/<name> && go build ./...`

## Architecture decisions

- Go module path: `github.com/kou-etal/etalbaas`
- Proto package prefix: `etalbaas.<domain>.v1`
- buf.yaml module: `buf.build/etalbaas/api`
- Each service has its own go.mod (not workspace)
- Proto imports: `etalbaas/` prefix (not `proto/etalbaas/`)
- CRD API group: `etalbaas.io/v1alpha1`
- JWT verification: Gateway only (downstream MSs trust headers)
- DB: sqlc (SQL → Go codegen), golang-migrate
- Phase 1: Free plan only, no billing, cron未実装

## Workflow

- Language: respond in Japanese, code/variables in English
- Commits: Conventional Commits format, no co-authored-by lines
- Branch: `<prefix>/<issue-number>-<description>`
- Always run tests and lint before considering a task done
- Do not auto-commit or auto-push without explicit instruction

## Implementation plan

1. ~~メタDB スキーマ設計~~ → `docs/database.md` + `deploy/migrations/meta/`
2. Proto基盤 (buf.yaml, common.proto)
3. 各ドメインProto (tenant → project → function → event → storage)
4. Service共通基盤 (go.work, pkg/)
5. Gateway
6. Tenant User MS
7. Project MS
8. Function MS
9. Event MS
10. Storage MS
11. Platform Operator
12. Providers (GPU/Storage/EventSource)
13. Deploy (Helm/ArgoCD/kubespray)
14. Dashboard（最後）
