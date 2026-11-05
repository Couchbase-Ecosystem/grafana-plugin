# Agent notes for grafana-plugin

Repo-specific guidance for automated/agent maintenance of the Couchbase Grafana datasource plugin.

## Layout
- `couchbase-datasource/` is the plugin: Go backend (`pkg/`, `Magefile.go`, `go.mod`) plus a Grafana React frontend (`src/`, `package.json`, `yarn.lock`).
- Root `docker-compose.yaml` / `run.sh` start Grafana with the plugin mounted; `release.sh`, `plugins/` and `.github/workflows/release.yml` are the release/repository-hosting path.

## Toolchain
- Package managers: Go modules and Yarn 1 (`packageManager: yarn@1.22.21`). Do not switch to npm/pnpm/Yarn Berry.
- Go version comes from `couchbase-datasource/go.mod` (`go` + `toolchain` directives); CI reads it via `setup-go` `go-version-file`. Bump the `toolchain` line to pick up Go standard-library security fixes.
- CI Node version is set in `.github/workflows/ci.yml`.

## Validation (same as PR CI)
Run from `couchbase-datasource/`:

```shell
yarn install --frozen-lockfile
yarn typecheck
yarn test:ci
yarn build
mage -v coverage
mage -v buildAll
```

`yarn test:ci` currently has no frontend tests (`--passWithNoTests`); backend tests live in `pkg/plugin`.

## Manual validation
For dependency or backend changes, run the built plugin in Grafana against a real Couchbase cluster:
- Mount `couchbase-datasource/dist` into `/var/lib/grafana/plugins/couchbase-datasource` and set `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=couchbase-datasource`.
- Provision or add a Couchbase datasource (`jsonData.host`, `jsonData.username`, `secureJsonData.password`), then run "Test" (expects `Ping OK`).
- In Explore, run the README queries using `str_time_range(<RFC3339 field>)` and `time_range(<millisecond field>)`.
- Known issue: `time_range` on a field stored as a JSON number panics in the backend (`interface {} is float64, not string`, issues #15/#21, open PR #20). Timestamps stored as strings work.

## Release safety
- `release.yml` runs on `v*.*.*` tags or `workflow_dispatch` and pushes to ECR using AWS secrets. Do not create tags, dispatch releases, publish signed plugin artifacts, or push images without explicit maintainer approval.
- Do not commit real credentials in `datasources/`.
