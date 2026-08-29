# Contributing to IdpForge

Thanks for considering it. This project is young (v1.0.0), so there's
plenty of real, useful work available, and no backlog of stale
half-finished PRs to compete with.

New to open source in general? [First Contributions](https://github.com/firstcontributions/first-contributions)
walks through the whole git/GitHub workflow end to end. It's fine to
open a PR that isn't perfect. Reviews exist for a reason.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
Participation means agreeing to abide by it.

## Finding something to work on

- [`good first issue`](https://github.com/raven-clown/idpforge/issues?q=is%3Aopen+label%3A%22good+first+issue%22)
  labels a small, well-scoped, clearly-described task. If the
  description is missing something you need to get started, ask in the
  issue rather than guessing. That's what it's there for.
- [`help wanted`](https://github.com/raven-clown/idpforge/issues?q=is%3Aopen+label%3A%22help+wanted%22)
  is bigger or needs more judgment, but still concrete.
- Found a real bug or gap that isn't filed yet? Open an issue first for
  anything nontrivial, so the approach gets agreed on before you sink
  time into an implementation. Typos, docs fixes, and genuinely small
  changes can just go straight to a PR.

## Branching

This project uses [GitHub Flow](https://docs.github.com/en/get-started/using-github/github-flow),
not GitFlow. There's one long-lived branch, `main`, always deployable
and protected (CI has to pass before anything merges into it). No
`develop` branch, no long-lived release branches, that's ceremony this
project doesn't need at its current size.

To contribute:

1. Fork the repo (or branch directly if you have write access).
2. Branch off `main`, named for what it does:
   `feat/oidc-client-crud`, `fix/webauthn-nil-panic`,
   `docs/clarify-redis-setup`, `chore/bump-deps`. The prefix isn't
   enforced by tooling, it's just so PR titles and branch lists stay
   scannable.
3. Keep the branch short-lived. Rebase on `main` before opening the PR
   if it's drifted.
4. Open a PR against `main`. Once CI is green and it's merged, the
   branch is auto-deleted (repo setting, not something you need to
   clean up yourself).

PRs merge via squash, so the commits on your branch can be messy while
you work, only the final PR title/description end up in `main`'s
history. Write that description like it's the commit message, because
it is.

## Local setup

Requires Go 1.26+ and Node 20+.

```bash
git clone https://github.com/raven-clown/idpforge
cd idpforge
cp .env.example .env   # set IDPFORGE_DB_DSN, or use the sqlite example below
```

Fastest way to run it locally is SQLite, no external database needed:

```bash
export IDPFORGE_DB_DRIVER=sqlite
export IDPFORGE_DB_DSN="file:./idpforge.db"
export IDPFORGE_REDIS_ENABLED=false
export IDPFORGE_DEFAULT_PASSWORD="Welcome123!"
go run ./cmd/server
```

The admin console is a separate Next.js app under `web/`, built as a
static export and embedded into the Go binary via `go:embed`. For
backend-only work you don't need to touch it, a placeholder UI is
already committed to `internal/webui/dist/`. If you're changing the
frontend:

```bash
cd web
npm install
npm run dev        # http://localhost:3000, proxies API calls to :8080
```

To embed a fresh build into the Go binary:

```bash
cd web && npm run build
rm -rf ../internal/webui/dist && cp -r out ../internal/webui/dist
```

## Before opening a PR

```bash
gofmt -l .          # must produce no output
go vet ./...
go test ./...

cd web
npm run lint
npm run build        # type-checks and produces the static export
```

CI runs all of the above. Not passing locally first just wastes a round
trip.

## Code conventions

Picked up by reading the existing code, not written down anywhere else,
so this is the actual source of truth:

- One Go package per concern under `internal/` (`users`, `rbac`,
  `audit`, `apiclient`, `iot`, `auth/oidc`, ...), each with its own
  `Repository` struct wrapping `*db.DB`.
- SQL is hand-written and dialect-aware, not an ORM. Use
  `db.Placeholder(n)` for parameter markers (`$1` / `?` / `@p1`
  depending on driver) so the same query works across Postgres, MySQL,
  MSSQL, and SQLite.
- A schema change needs a migration file in all four
  `internal/db/migrations/{postgres,mysql,mssql,sqlite}/NNNN_name.sql`
  directories, numbered one past the current highest. They're
  deliberately not auto-generated from a single source, since the four
  dialects genuinely differ (identity columns, quoting, index syntax).
- Every mutating HTTP handler logs an `audit.Entry`. Follow the pattern
  already in `internal/httpserver/*_handlers.go` rather than inventing a
  new one.
- Comments explain *why*, not *what*. If a comment just restates what
  the code already says, it doesn't belong. Reserve them for a
  non-obvious constraint, a workaround, or something that would
  genuinely surprise a reader.
- No AI-authored commit messages or co-author trailers. Write the
  commit message yourself, in your own words, describing what changed
  and why.
- `internal/httpserver` has real integration tests
  (`internal/httpserver/testserver_test.go` builds a full in-memory test
  server). Add to that file's pattern for new handler tests rather than
  mocking pieces individually.

## Security

Found a real vulnerability? Please don't open a public issue for it.
Email ekdanai.kk@gmail.com directly with details and a reproduction if
you have one.
