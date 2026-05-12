# CLAUDE.md

## Thinking policy

トークン消費を気にせず、深く考えてから回答すること。曖昧な回答より正確で網羅的な回答を優先。

## Architecture source of truth

プロジェクト概要・技術スタック・設計判断の詳細は `docs/architecture.md` に集約。
実装や設計で不明点があれば必ず architecture.md の該当セクションを読んで確認すること。
CLAUDE.md に概要を二重管理しない。

## Environment

- OS: Windows (Git Bash)
- シェルコマンドのパスは `/c/etalbaas/...` 形式を使うこと（`C:\` や `cd /d` は動かない）

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
│       ├── event/v1/
│       └── storage/v1/
├── services/               # Go microservices
│   ├── gateway/
│   ├── tenant-user/
│   ├── project/
│   ├── function/
│   ├── event/
│   └── storage/
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
2. ~~Proto基盤~~ (buf.yaml, buf.gen.yaml, common.proto)
3. ~~各ドメインProto~~ (tenant, project, function, event, storage)
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
