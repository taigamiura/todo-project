# GitHub Actions Workflows

GitHub Actions の workflow ファイルは仕様上 `.github/workflows` 直下に置く必要があるため、このリポジトリではファイル名の接頭辞でカテゴリをそろえています。

## CI

- `ci-web-app.yml`: Next.js アプリの lint, unit test, build
- `ci-bff.yml`: BFF の test と build
- `ci-user-service.yml`: user-service の test と build
- `ci-todo-service.yml`: todo-service の test と build
- `ci-stack.yml`: compose 設定と nginx 設定の検証

## QA

- `qa-e2e.yml`: Docker Compose でスタックを起動して Playwright E2E を実行

## Security

- `security-codeql.yml`: JavaScript/TypeScript と Go の CodeQL 解析
- `security-container-report.yml`: Trivy の SARIF を生成して code scanning に upload
- `security-container-enforce.yml`: Trivy で HIGH/CRITICAL を fail 条件として強制

## Automation

- `automation-dependabot-auto-merge.yml`: Dependabot PR を自動 approve し auto-merge を有効化
