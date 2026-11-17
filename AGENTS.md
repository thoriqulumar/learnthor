# AGENTS.md

Guidance for AI coding agents working in this repository.

## Repository overview

Learnthor is a Japanese vocabulary / kanji learning app with two parts:

- `backend/` — Go (Gin + GORM + PostgreSQL) serving `/api/v1` REST routes.
  Layered as: `main.go` → `router/` (composition root) → `api/handlers/` (HTTP)
  → `api/middleware/` (auth, CORS, gating) → `services/` (business logic) →
  `models/` (GORM structs with UUID PKs via `models.Base`). `db/` connects &
  migrates. `seed` is a standalone seeder binary entrypoint.
- `frontend/` — React 19 SPA (Vite + React Router 7, Tailwind CSS 3, Axios).
  `src/main.jsx` boots `<App/>`, `src/lib/api.js` is the pre-configured Axios
  instance (JWT from localStorage, 401 → redirect to `/login`),
  `src/lib/errors.js` maps backend error codes to bilingual friendly
  messages (`friendlyError(err, gloss)`), `src/context/AuthContext.jsx`
  manages auth, `src/pages/*` are route views, `src/components/*` are
  shared widgets. API base is `VITE_API_BASE_URL`.

The backend also integrates Midtrans for one-time premium purchases: free
users access N5 + basics; N4/N3 + Daily Vocab sections require premium
(`Section.IsPremium`, `User.SubscriptionTier`). The payment flow lives in
`services/payment_service.go` + `handlers/payment_handler.go`; the Midtrans
Snap popup is opened client-side from `src/pages/UpgradePage.jsx`.

There are no repository-root config files beyond this one; each app manages its
own tooling. Cursor/Copilot rule files were not present at creation time.

## Build, lint, test commands

### Backend (Go 1.23, module `github.com/learnthor/backend`)

Run from `backend/`:

```bash
go build ./...                     # compile (no binary emitted)
go run main.go seed                # run server / seeder
go vet ./...                       # lint
gofmt -l .                         # formatting check (fix with gofmt -w .)
go test ./...                      # run all tests
go test ./services                 # run one package
go test ./services -run TestX -v   # single test
golangci-lint run                  # if installed
```

Required env (see `backend/.env.example`): `DATABASE_URL`, `JWT_SECRET`
(panics if unset), optional `PORT` (default 8080). `go fmt ./...` is the
canonical formatter — **always gofmt before committing**; Go files use tabs.

### Frontend (Node, ESLint flat config, Vite)

Run from `frontend/`:

```bash
npm install
npm run dev            # vite dev server
npm run build          # production build → dist/
npm run lint           # eslint .  (flat config in eslint.config.js)
npm run preview        # serve built bundle
```

ESLint uses `@eslint/js` recommended + `eslint-plugin-react-hooks` +
`eslint-plugin-react-refresh`. JS/JSX only (no TypeScript). Lint *is* the
typecheck equivalent — there is no `tsc`. To lint a single file:
`npx eslint src/pages/QuizPage.jsx`.

Tests run with Vitest + React Testing Library (jsdom):

```bash
npm test             # run all tests once (vitest run)
npm run test:watch   # watch mode
npx vitest src/lib/api.test.js   # single file
```

Test files live next to the code they cover (`*.test.js` / `*.test.jsx`).
Config is the `test` block in `vite.config.js`; setup (jest-dom matchers) is
`src/test/setup.js`. Pre-existing lint errors in `QuizPage.jsx`,
`ModulePage.jsx`, and `AuthContext.jsx` (react-refresh export shape) are
known — don't "fix" them unless asked.

### Backend tests

Service tests use an in-memory SQLite database via
`github.com/glebarez/sqlite` (pure Go, no CGO). Each test calls
`newTestDB(t)` from `services/testhelper_test.go` for an isolated migrated
DB, then uses the fixture helpers (`seedCurriculum`, `createTestUser`,
`createSession`) to build a course → section → module → vocab chain. Tests
exercise real GORM queries (joins, upserts, transactions) rather than mocks.
Note: `newTestDB` must NOT reuse `db.Migrate` — it runs Postgres-only
`CREATE EXTENSION pgcrypto`.

## Go code style (backend)

- **Package layout**: one concern per sub-package (`handlers`, `middleware`,
  `services`, `models`, `db`, `router`). Avoid import cycles.
- **Imports** in blocks, stdlib first, then third-party, then local
  (`github.com/learnthor/backend/...`); gofmt keeps this order. No blank lines
  between import lines within a group.
- **Naming**: `CamelCase` for exported, `camelCase` for unexported. Exported
  service types `XxxService`, constructor `NewXxxService(db *gorm.DB)`. Handler
  types `XxxHandler` with `NewXxxHandler`. Struct fields use Go names; JSON tags
  use `snake_case`.
- **Comments**: every exported identifier gets a doc comment starting with the
  identifier name. Use the `// ── Section ──` divider style seen in services for
  grouping. Keep `@Summary`/`@Router` swag comments on handlers in sync.
- **Errors**: define domain sentinels in `services` (`ErrNotFound`,
  `ErrBadRequest`, `ErrForbidden` in `services/session_service.go`). Services
  return these (or wrap with `fmt.Errorf("...: %w", err)`); handlers map them
  to HTTP status and a consistent `errorResponse{Code, Message}` JSON envelope.
  Use `errors.Is` for comparisons when wrapping. Never leak internal error
  strings to clients — return generic `"INTERNAL_ERROR"` for 500s.
- **GORM models**: embed `models.Base` for UUID PK + timestamps; use
  `gorm:""`/`json:""` tags. Don't expose `PasswordHash` in JSON.
- **HTTP**: use `c.JSON(status, body)`, `c.AbortWithStatusJSON` in middleware.
  Bind request bodies with `ShouldBindJSON` + `binding:"..."` tags. Put the
  auth user's UUID into context via `c.Set("user_id", ...)` and read it in
  handlers (`c.Get("user_id")`), injecting it into request structs tagged `json:"-"`.
- Transactions: `s.db.Transaction(func(tx *gorm.DB) error {...})` for multi-step
  writes; use `clause.Locking{Strength:"UPDATE"}` for read-modify-write rows.

## React/JS code style (frontend)

- **No TypeScript**: plain `.js`/`.jsx`. Keep components functional with hooks.
- **Imports**: named from React, third-party (`react-router-dom`, `axios`), then
  relative. The shared Axios client is `../lib/api` — **never** `import axios`
  directly; use `api` so JWT interceptor & 401 redirect apply.
- **Styling**: Tailwind utility classes inline (see `tailwind.config.js`). No
  CSS modules; avoid co-located `.css` except `src/index.css`/`App.css`.
- **Naming**: components `PascalCase` (file matches: `QuizPage.jsx`); hooks/
  helpers `camelCase`. Page components `export default`. Constants grouped at
  top of file (e.g. SRS constants in `QuizPage.jsx`).
- **State**: `useState`/`useEffect`. Use `useRef` for timers and mutable
  non-render state. Add `// eslint-disable-line react-hooks/exhaustive-deps`
  sparingly when intentionally omitting deps (see existing files).
- **Routing**: declare routes in `src/App.jsx`; protected routes nested under
  `<ProtectedLayout/>`, guest-only routes under `<GuestRoute/>`. Auth via
  `useAuth()` from `AuthContext`.
- **API calls**: prefer `.then/.catch/.finally` chains (existing style) or
  async/await consistently within a file. Handle 401/403 distinctly (403 →
  "module locked" messages). Persist resumeable quiz state in `localStorage`
  keyed per user+module+type with a 7-day TTL.
- **Error handling UI**: render spinner during loading, error banner with a
  "go back" affordance; never crash on missing fields.

## General rules for agents

- Run `go vet ./...` / `gofmt -w .` for backend and `npm run lint` for frontend
  before reporting a task complete.
- Match existing patterns: new service → `NewXxxService`, new handler →
  `NewXxxHandler`, register routes in `router/router.go`. Add model to
  `db.Migrate`.
- Don't introduce new top-level dependencies without reason; if you must, add
  to `go.mod` via `go get` or `npm install <pkg>` and update lockfiles.
- Keep secret values out of code: read from env. Never commit `.env`.
- Don't add comments unless explaining non-obvious logic; otherwise let code &
  doc comments on exported identifiers carry the intent.
- This repo currently has no git repository initialized (`Is directory a git
  repo: no`) — do not assume `git` commands work without checking first.