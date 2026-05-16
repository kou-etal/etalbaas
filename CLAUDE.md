# CLAUDE.md

## Thinking policy

トークン消費を気にせず、深く考えてから回答すること。曖昧な回答より正確で網羅的な回答を優先。

## Architecture source of truth

プロジェクト概要・技術スタック・設計判断の詳細は `docs/architecture.md` に集約。
実装や設計で不明点があれば必ず architecture.md の該当セクションを読んで確認すること。

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
│       ├── secret/v1/
│       └── storage/v1/
├── services/               # Go microservices
│   ├── gateway/
│   ├── tenant-user/
│   ├── project/
│   ├── function/
│   ├── event/
│   └── storage/
├── operator/               # kubebuilder Operator (Project/Function CRDs)
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

## Deferred refactoring

- 横断的な変更が必要でその Step では対応しないリファクタリング項目は `REFACTOR_BACKLOG.md` に追記する
- このファイルは `.gitignore` 済み（ローカル専用のバックログ）
- 記載フォーマット: 見出し(問題の要約)、発見元、対象ファイル、問題の説明、対策案
- codex review で「却下（横断的変更）」と判断した指摘は必ずここに記録する

## Codex review

- ファイル編集が全て完了してから、`codex review` を **1ファイルずつ** 実行する
- 一括レビュー（`codex review --uncommitted`）は浅くなるため使わない
- **実行時の注意**: `codex exec "..." 2>&1` のみで実行すること。パイプ（`| grep`, `| tail`, `| head`）やリダイレクト（`> file`, `< /dev/null`）を付けると stdin の問題でハングする。並列実行も不可。必ず **1つずつ順次、パイプなし** で実行する
- プロンプトテンプレート:
  ```
  codex exec "Review <file> as a senior Go engineer.
  Flag any anti-patterns, bugs, security risks, concurrency issues,
  performance concerns, and API design problems.
  Context: shared package for a multi-tenant BaaS platform
  (connect-go RPC, k8s native, pgx, OpenTelemetry).
  Output format: severity (high/medium/low), file:line, description."
  ```
- 観点を狭く列挙しない（広い網をかけてアンチパターンを拾う）
- Codex の指摘は鵜呑みにしない。architecture.md の設計判断と照合し、根拠を持って反映/却下を判断する
- 判断結果を表形式（指摘 / 判断 / 理由）でユーザーに提示する

## Implementation plan

1. ~~メタDB スキーマ設計~~ → `docs/database.md` + `deploy/migrations/meta/`
2. ~~Proto基盤~~ (buf.yaml, buf.gen.yaml, common.proto)
3. ~~各ドメインProto~~ (tenant, project, function, event, storage)
4. ~~Service共通基盤~~ (go.work, pkg/)
5. ~~Gateway~~
6. ~~Tenant User MS~~
7. ~~Project MS~~
8. ~~Function MS~~
9. ~~Event MS~~
10. ~~Storage MS~~ — Phase 1 スキップ (Step 13 で実装)
11. ~~Platform Operator~~ (Project/Function CRDs, Kaniko build, multi-tenant isolation)
12. ~~NATS JetStream 統合 + KEDA NATS Scaler~~
13. ~~Storage Provider + Storage MS (R2/MinIO)~~
14. ~~Function Auto-Build 完成度向上 + 実行経路動作確認~~
15. ~~GPU Provider 抽象化レイヤ~~ (Provider interface, RunPod Serverless, Self-managed GPU config, Dispatcher binary)
16. Envoy Gateway + Cloudflare DNS + Wildcard TLS
17. Dashboard (Next.js)
18. Integration Tests + Beta
19. Production Deploy

