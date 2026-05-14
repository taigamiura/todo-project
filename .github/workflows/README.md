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
- `security-container-enforce.yml`: Trivy の SARIF upload と HIGH/CRITICAL の fail 判定を別 job で実行

## Automation

- `automation-dependabot-auto-merge.yml`: Dependabot PR を自動 approve し auto-merge を有効化
