# Todo Platform

Next.js フロントエンドを入口にしつつ、内部を nginx, BFF, user-service, todo-service, PostgreSQL, Redis に分離した構成です。フロント側は localStorage を廃止し、API Route と HTTP-only cookie セッションで認証と Todo CRUD を扱います。

## Architecture

- nginx: HTTPS 終端、静的アセットキャッシュ、API レート制限
- todo-app: Next.js App Router, middleware, API Route, zod, react-hook-form
- bff: JWT 発行・検証、user/todo service の集約、Redis キャッシュ
- user-service: ユーザー登録・認証、bcrypt ハッシュ化、PostgreSQL 永続化
- todo-service: Todo CRUD、ユーザー単位分離、PostgreSQL 永続化
- redis: BFF の Todo 一覧/詳細キャッシュ

## Security Notes

- セッションは HTTP-only cookie に保持し、Next middleware とサーバー側判定で保護しています。
- nginx はローカル自己署名証明書で TLS を終端し、HSTS と基本的なセキュリティヘッダを付与します。
- パスワードは user-service 内で bcrypt ハッシュ化します。
- private ネットワークは internal 指定で外部公開しません。

## Run

1. ルートの compose ファイルを使って起動します。

```bash
docker compose -f compose.yml up --build
```

2. ブラウザで https://localhost:8443 を開きます。

3. 自己署名証明書の警告はローカル開発用として許可してください。

## Environment

- 開発用のサンプル値は [.env.example](.env.example) に置いています。
- 初回セットアップ時に `.env.example` を `.env` にコピーして値を埋めてください。
- compose は root の `.env` を自動で読み込みます。
- `APP_SESSION_SECRET`, `USER_DB_PASSWORD`, `TODO_DB_PASSWORD`, `GRAFANA_ADMIN_PASSWORD` は必須です。

## Validation

- frontend: `npm run lint`, `npm run build`
- frontend tests: `cd todo-app && npm run test:all`
- frontend e2e: `cd todo-app && npm run playwright:install && npm run test:e2e`
- user-service: `go build ./...`
- todo-service: `go build ./...`
- bff: `go build ./...`
- container security: GitHub Actions の `Container Security` workflow で Trivy scan を実行

- Playwright は既定で `https://localhost:8443` を叩き、自己署名証明書は許容します。別 URL に向ける場合は `PLAYWRIGHT_BASE_URL` を上書きしてください。

## Observability

- Edge entrypoint: http://localhost:8080 または https://localhost:8443
- Jaeger UI: http://localhost:16686
- Prometheus UI: http://localhost:9090
- Grafana UI: http://localhost:3001
- Grafana の初期ログインは `admin` / `.env` の `GRAFANA_ADMIN_PASSWORD` です。
- Prometheus には [observability/prometheus/alerts.yml](observability/prometheus/alerts.yml) の latency / error rate alert rule を読み込みます。
- Grafana は起動時に `Todo Platform Overview` ダッシュボードを自動 provision します。

## OpenAPI And Mock Testing

- OpenAPI 定義は [openapi/bff.yaml](openapi/bff.yaml), [openapi/user-service.yaml](openapi/user-service.yaml), [openapi/todo-service.yaml](openapi/todo-service.yaml) にあります。
- DB や Redis を起動せずに contract test したい場合は Prism mock を使います。

```bash
docker compose -f compose.mock.yml up bff-mock
```

- 外部向け API を mock したいときは `http://localhost:8080` を参照してください。
- internal service 単位で mock したいときは `user-service-mock` と `todo-service-mock` も起動できます。
- Next 側のテストや手動確認では `BFF_BASE_URL=http://localhost:8080` を向ければ DB なしで API 契約を確認できます。

### Frontend Contract Sync

- todo-app では `npm run generate:api-types` で [todo-app/src/lib/api/generated/bff.ts](todo-app/src/lib/api/generated/bff.ts) を再生成します。
- API route の BFF 契約テストは `cd todo-app && npm run test:api:mock` で実行できます。

## Container Security

- GitHub Actions では [security-container-enforce.yml](.github/workflows/security-container-enforce.yml) が `nginx`, `bff`, `user-service`, `todo-service`, `todo-app` のイメージを対象に Trivy の report job と enforce job を実行します。
- report job は SARIF を GitHub code scanning に upload し、enforce job は `HIGH` と `CRITICAL` の脆弱性を検出したら CI を fail させます。
- CI では `HIGH` と `CRITICAL` の脆弱性を fail 条件にしています。
- SARIF レポートは GitHub の code scanning に upload されるため、PR と Security タブの両方で確認できます。
- 一時的に除外が必要な場合だけ [.trivyignore](.trivyignore) を使ってください。

### Trivy Flow

1. workflow が対象イメージを build します。
2. [security-container-enforce.yml](.github/workflows/security-container-enforce.yml) の report job が `HIGH,CRITICAL` を scan して SARIF を出力し、GitHub に upload します。
3. 同じ workflow の enforce job が同じ条件で table scan します。
4. `HIGH,CRITICAL` が残っていれば required check を fail させます。

### Ignore Policy

1. まず base image や dependency の更新で解消できるか確認します。
2. fix がない、または実行経路に載らない場合だけ一時 ignore を検討します。
3. ignore する場合は CVE と理由、見直し期限、追跡 issue を残します。
4. `.trivyignore` は恒久逃げ道ではなく、一時運用の台帳として扱います。