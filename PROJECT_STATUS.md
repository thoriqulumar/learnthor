> September 28, 2026 update: JSON seeding now uses a validated canonical catalog with immutable IDs, transactional in-place updates, offline validation, and explicit reset confirmation. See [seed guide](backend/cmd/seed/README.md). Earlier sections below are historical; routine updates must use the normal seed command, not reset. Web is the current product focus; mobile is out of scope.

# Learnthor — Project Status & Plan

> Exported snapshot of the current session: what's built, what's planned,
> key decisions, bug fixes, and how to verify everything.

---

## 1. Project overview

Learnthor is a Japanese vocabulary / kanji learning web app (Go backend +
React 19 frontend) with JLPT-structured courses, progressive module gating,
multiple-choice quizzes, spaced-repetition (FSRS-lite), per-skill mastery
tracking, Midtrans premium purchases, and an Expo mobile app (planned).

```
learnthor/
├── backend/          Go 1.23 (Gin + GORM + PostgreSQL)
├── frontend/         React 19 (Vite + React Router 7 + Tailwind 3 + Axios)
├── mobile/           [planned] Expo React Native (Android first)
├── DEPLOY.md         Docker Compose + bare-metal deploy guide
├── docker-compose.yml
├── .env.docker.example
└── AGENTS.md         coding/style guidelines for AI agents
```

---

## 2. Build, lint, test commands

### Backend (from `backend/`)

```bash
gofmt -w .                # format (tabs)
go vet ./...              # lint
go build ./...            # compile
go test ./...             # all tests
go test ./services -run TestX -v   # single test
go run main.go            # start server
go run ./cmd/seed         # seed database
go run ./cmd/seed --reset --confirm-reset # wipe curriculum + re-seed (preserves users)
```

Required env: `DATABASE_URL`, `JWT_SECRET`, optional `PORT` (default 8080).
Payments require: `MIDTRANS_SERVER_KEY`, `MIDTRANS_CLIENT_KEY`,
`MIDTRANS_ENV=sandbox`, `PREMIUM_PRICE_IDR=99000`.

### Frontend (from `frontend/`)

```bash
npm install
npm run dev              # vite dev server
npm run build            # production build → dist/
npm run lint            # eslint . (flat config)
npm test                # vitest run (all tests once)
npm run test:watch      # vitest watch mode
```

### Backend tests

Service tests use an in-memory SQLite database via
`github.com/glebarez/sqlite` (pure Go, no CGO). Each test calls
`newTestDB(t)` from `services/testhelper_test.go` for an isolated migrated
DB, then uses the fixture helpers (`seedCurriculum`, `createTestUser`,
`createSession`) to build a course → section → module → vocab chain.
`newTestDB` must NOT reuse `db.Migrate` (it runs Postgres-only
`CREATE EXTENSION pgcrypto`).

### Frontend tests

Vitest + React Testing Library (jsdom). Test files live next to the code
they cover (`*.test.js` / `*.test.jsx`). Config is the `test` block in
`vite.config.js`; setup is `src/test/setup.js` (jest-dom matchers).

---

## 3. Phase status

| Phase | Description | Status |
|-------|-------------|--------|
| 0     | SRS engine (FSRS-lite) + bilingual gloss | **Complete** |
| 0     | Docker Compose deployment | **Complete** |
| 1     | Unit tests (backend + frontend) | **Complete** |
| 2     | Midtrans payments + friendly errors | **Complete** |
| 3     | UI overhaul (playful/colorful) + framer-motion | **Planned** |
| 4     | Expo mobile app (Android first) | **Planned** |

---

## 4. Phase 0 — SRS engine + bilingual gloss (complete)

### What was built

- **FSRS-lite scheduler** (`backend/services/fsrs.go`): hand-rolled
  FSRS-4.5 algorithm — Stability, Difficulty, Retrievability, state
  transitions (New→Learning→Review→Relearning), lapse handling, short
  relearning steps. 9 tests in `fsrs_test.go`.
- **SessionService refactor** (`backend/services/session_service.go`):
  `SubmitAnswer` now calls `ScheduleReview` (Stability/Difficulty actually
  update — were dead columns before). Accepts optional `Rating` 1–4;
  falls back to `IsCorrect` bool for backwards compat.
  - New: `StartSession` (creates/resumes session, 3-item batch of
    brand-new items per PRD §3.2), `ResumeSession`, `pickNewBatch`.
- **ReviewService** (`backend/services/review_service.go`): cross-module
  daily SRS review queue. `GetDueCards` (unlimited, most-overdue first),
  `SubmitReview` (grade 1–4, FSRS update), `GetStats` (counts by state).
- **Endpoints** (`backend/router/router.go`):
  - `POST /sessions` — start/resume a module study session
  - `GET /sessions/:id` — resume active session
  - `POST /sessions/submit` — submit a single answer
  - `GET /reviews/due` — all cards due for review
  - `POST /reviews/submit` — grade a due card (rating 1–4)
  - `GET /reviews/stats` — counts for dashboard widget
- **Frontend ReviewPage** (`frontend/src/pages/ReviewPage.jsx`): four
  FSRS rating buttons (Again/Hard/Good/Easy), due-card queue, completion
  summary. Route `/review`.
- **Bilingual gloss toggle**: `User.PreferredGloss` (`en`|`id`|`both`,
  default `both`). `GET/PUT /me/preferences`. EN/ID/EN+ID toggle in
  HomePage header, reflected app-wide via `refreshUser` in AuthContext.
  LearnPage + ReviewPage cards respect the preference.

### Test counts

- Backend: 65 tests total (9 FSRS + 12 session + 6 review + 11 quiz + 8
  gating + 7 auth + 9 payment + 3 premium-gating)
- Frontend: 22 tests (5 api + 5 AuthContext + 5 HomePage + 6 errors +
  1 HomePage premium)

---

## 5. Phase 0 — Docker Compose (complete)

Files: `backend/Dockerfile`, `backend/.dockerignore`, `frontend/Dockerfile`,
`frontend/.dockerignore`, `frontend/nginx.conf`, `docker-compose.yml`,
`.env.docker.example`.

- Multi-stage builds (alpine → minimal runtime images)
- `docker compose up -d --build` starts db + backend + frontend
- `docker compose run --rm seed --reset --confirm-reset` re-seeds
- Only port 80 exposed; Postgres + backend internal to compose network

---

## 6. Phase 1 — Unit tests (complete)

### Coverage

**Backend service-layer tests** (in-memory SQLite):
- `session_service_test.go` — SubmitAnswer streak/mastery/complete paths,
  wrong-session rejection, completed-session rejection, rating overrides
  IsCorrect. StartSession batch creation, resume, exclude-already-reviewed,
  module-not-found. ResumeSession items + not-found.
- `review_service_test.go` — GetDueCards ordering + user isolation,
  SubmitReview validation + create + update, GetStats by state.
- `quiz_service_test.go` — FinishQuiz required-skills (no-example module
  doesn't require fill_blank), unplayed-skill-zero, high-score-keeps-best,
  not-found, empty-answers. GenerateQuiz questions-have-correct-answer,
  skips-fill-blank-without-example. isQuizCompatible matrix. blankSentence.
  SectionQuiz gating (forbidden without mastery, privileged bypass).
- `gating_service_test.go` — first-module-allowed, denied-without-progress,
  denied-below-threshold, allowed-at-threshold, privileged bypass,
  not-found, IsPrivilegedUser matrix, premium-section gating
  (free denied, premium allowed, admin bypass).
- `auth_service_test.go` — register creates student with token,
  dup-email/username rejection, login success + JWT round-trip,
  bad credentials, ParseToken rejects garbage, preferences update +
  validation + nonexistent user.

**Frontend tests** (Vitest + RTL):
- `api.test.js` — request interceptor (JWT attach, no-token),
  response interceptor (401 clears + redirects, non-401 passes through)
- `AuthContext.test.jsx` — anonymous start, stored-session restore,
  login stores token+user, logout clears, refreshUser patches + persists
- `HomePage.test.jsx` — renders sections+modules, premium overlay for
  free users, premium user sees unlocked, review widget hidden/shown,
  admin bypass
- `errors.test.js` — code mapping, status fallback, network errors,
  unknown errors, bilingual parts

### Bugs found through testing (fixed)

1. **`review_service.go` FirstOrCreate inline conds**: passing a struct
   as inline conds to `FirstOrCreate` merges its non-zero fields into the
   search WHERE clause. The search included `state='new'` +
   `difficulty=5` + `next_review_at=now`, so any EXISTING review (state
   is `review`/`learning`/etc.) would never match → tried to INSERT a
   duplicate → 500 error on every re-review in production. Fixed by using
   `.Attrs()` (creation-only values, matching session_service's pattern).

2. **`models.Base` Postgres-only default**: `default:gen_random_uuid()`
   is Postgres-specific; SQLite (used in tests) can't parse it. The
   `BeforeCreate` hook already guarantees UUIDs are assigned in Go, so
   the DB-level default is redundant. Removed it from the tag → models
   are now DB-portable, tests work on SQLite, zero runtime behavior
   change in Postgres.

---

## 7. Phase 2 — Midtrans payments + friendly errors (complete)

### Data model changes

- `User.SubscriptionTier` (`free`/`premium`, default `free`)
- `User.PremiumUntil *time.Time` (null = lifetime; reserved for future
  subscription plans)
- `Payment` model: `order_id` (unique), `user_id`, `plan`, `gross_amount`
  (IDR), `status`, `snap_token`, `snap_url`, `paid_at`. Migrated via
  `db.Migrate`.
- `Section.IsPremium bool` — seeder marks N4/N3/Daily Vocab sections.

### Payment flow

1. `POST /api/v1/payments/create` (auth) → `PaymentService.CreateTransaction`:
   rejects already-premium users; reuses existing pending payment (prevents
   order-id spam); calls `MidtransGateway.CreateSnapTransaction`; persists
   `Payment` (pending); returns `{order_id, snap_token, snap_url,
   gross_amount, client_key, env, reused}`.
2. Frontend `UpgradePage.jsx` loads `snap.js` (sandbox/production host
   from `env`), opens `snap.pay(token, callbacks)`.
3. `POST /api/v1/payments/notification` (public, signature-verified) —
   Midtrans webhook. `PaymentService.HandleNotification`: verifies
   `signature_key` = SHA512(order_id + status_code + gross_amount +
   server_key); re-checks transaction server-to-server via
   `CheckTransaction` (never trusts webhook body alone); maps
   Midtrans status to local `PaymentStatus`; on settlement →
   `UPDATE users SET subscription_tier='premium'`. Idempotent.
4. `GET /api/v1/payments/:order_id` (auth) — frontend polls after Snap
   popup closes until `status===settlement` → `refreshUser` → success.

### Premium gating

- `GateResult.RequiresPremium bool` — distinct from progression lock.
- `CheckModuleAccess`: if `section.IsPremium && user.SubscriptionTier !=
  premium` → deny with `RequiresPremium=true` (runs before predecessor
  check — premium is the outermost gate). Admin/editor bypass.
- `RequireModuleAccess` middleware: `RequiresPremium` → 403 with code
  `UPGRADE_REQUIRED` (vs `MODULE_LOCKED` for progression).
- `GetSections` handler: exposes `is_premium` on each section; locks
  premium sections for free users; premium users + privileged see them
  open.

### Seeder premium flags

- `upsertSection` syncs `is_premium` on re-runs (like `sort_order`).
- `seedPagedVocab` specs: n5 free, n4/n3 premium.
- `seedDayFiles`: "Daily Vocab (Day 29–59)" → `IsPremium: true`.

### Payment tests (9 passing)

`payment_service_test.go` with a fake `MidtransGateway`:
- CreateTransaction creates pending payment, reuses pending, rejects
  already-premium.
- HandleNotification: settlement grants premium (checks ground-truth
  round-trip), rejects bad signature (no CheckTransaction call),
  idempotent on settled, unknown order → not-found.
- mapMidtransStatus matrix (capture+accept, capture+challenge,
  settlement, expire, cancel, deny, pending).
- GetPaymentStatus own-only (other user → not-found).

### Frontend — friendly errors

`src/lib/errors.js`:
- `friendlyError(err, lang)` — single message in user's preferred
  language (`id` → Indonesian; `en`/`both` → English).
- `friendlyErrorParts(err)` — returns `{en, id}` for components that
  want both.
- Maps ~18 backend codes (INVALID_CREDENTIALS, UPGRADE_REQUIRED,
  MODULE_LOCKED, ALREADY_PREMIUM, …), HTTP status fallbacks (400/401/
  403/404/409/429/500+), network failures, unknown errors.

### Frontend — Upgrade page + premium UI

- `src/pages/UpgradePage.jsx` — plan card (gradient hero, price Rp
  99.000), benefits list, Snap popup integration, polling after payment,
  success/pending/error states. Already-premium → "You're Premium" screen.
- HomePage: 👑 badge on premium section headers; premium-locked overlay
  (upgrade CTA, replaces 🔒 for premium sections); upgrade banner for
  free users.
- Navbar: "👑 Premium" pill when active; "👑 Upgrade" link when free.
- Route `/upgrade` registered in `App.jsx`.

### Docs updated

- `.env.example`, `.env.docker.example` — Midtrans keys + price.
- `DEPLOY.md` — A.9 Midtrans config section (keys, webhook URL, sandbox
  testing, go-live).
- `AGENTS.md` — payment flow mention + test commands.

---

## 8. Pre-existing issues (not fixed — known)

- ESLint errors in `QuizPage.jsx` (4: no-unused-vars ×3,
  set-state-in-effect ×1), `AuthContext.jsx` (1:
  react-refresh/only-export-components — structural, would need splitting
  `useAuth` into a separate file), `ModulePage.jsx` (1:
  set-state-in-effect). These pre-date this session and are intentionally
  not touched unless asked.
- `go.mod` was upgraded from `go 1.18` to `go 1.23` by `go get
  glebarez/sqlite` (modernc.org/sqlite requires go 1.21+). Dockerfile
  updated to `golang:1.23-alpine`.

---

## 9. Remaining phases (planned, not started)

### Phase 3 — UI overhaul (playful & colorful)

**Design tokens** in `tailwind.config.js`:
- Brand palette: keep indigo as primary; add sakura-pink (#FEM) +
  matcha-green accents.
- Japanese display font: Noto Sans JP via Google Fonts (import in
  `index.html` or `App.css`).
- Consistent radius/shadow scale (soft-2xl cards, layered shadows).

**UI primitives** — `src/components/ui/`:
- `Button` (variants: primary/secondary/ghost/danger, sizes).
- `Card` (soft shadow, rounded-3xl, hover lift).
- `Badge` (small pill, color variants).
- `ProgressRing` (extract from HomePage's MasteryRing).
- `ErrorBanner` (icon + friendly message + retry/go-back affordance) —
  wraps `friendlyError`.

**Animations** — `framer-motion` (npm install):
- LearnPage: card flip (3D rotateY), reveal slide-down.
- QuizPage: option-select feedback (scale + color), correct/incorrect
  bounce.
- ReviewPage: swipe-feel rating buttons (drag gesture).
- HomePage: section cards stagger-in, mastery ring fill animation.
- Page transitions: fade/slide via AnimatePresence in ProtectedLayout.

**Page-by-page polish**:
- HomePage: gradient hero header with subtle pattern, premium-looking
  module cards (soft shadows, hover lift), animated empty states with
  illustrations (emoji → proper SVG later).
- LearnPage: bigger card, flip animation, progress dots with motion.
- QuizPage: animated option states (correct glows, wrong shakes).
- ReviewPage: swipe-feel rating buttons, streak counter animation.
- LoginPage / RegisterPage: centered card with gradient accent, form
  validation inline errors via `ErrorBanner`.

**Dark mode**: `class` strategy toggle stored in localStorage; Navbar
switch. Tailwind `dark:` variants on all components.

**Roll out `friendlyError`**: swap every hardcoded `.catch()` string
across LoginPage, RegisterPage, HomePage, ModulePage, VocabPage,
LearnPage, QuizPage, SectionQuizPage, ReviewPage to `friendlyError(err,
user?.preferred_gloss)`.

### Phase 4 — Expo mobile app (Android first)

New top-level `mobile/` workspace:

```bash
cd mobile
npx create-expo-app learnthor-mobile --template blank-typescript
# or JS: --template blank
```

**Stack**:
- Expo SDK 51+ (managed workflow)
- React Navigation (stack + bottom tabs)
- NativeWind (Tailwind for RN — reuses design tokens)
- expo-secure-store (JWT storage)
- axios (same `api.js` pattern, base URL = VPS IP)
- expo-speech (audio/TTS for pronunciation)

**Screens v1** (port from web):
1. Login / Register
2. Home (sections + modules, lock states, premium badges)
3. Learn (flashcard with flip)
4. Quiz (multiple choice)
5. Review (SRS queue with rating buttons)
6. Upgrade (Midtrans Snap via WebView — Snap is a hosted page)
7. Profile (gloss toggle, subscription status, logout)

**Backend for mobile**: no changes needed — JWT + REST works from mobile
unchanged. CORS is browser-only. Just point the mobile `api.js` at the
VPS IP.

**Extras rolled in**:
- Audio/TTS: browser `speechSynthesis` (ja-JP) on web, `expo-speech` on
  mobile. "Listen" button on every word + sentence.
- Refresh-token endpoint: current JWT is 72h hard-expiry; add a refresh
  endpoint so mobile users aren't logged out every 3 days.

---

## 10. Post-session checklist (to run before shipping)

```bash
# Backend
cd backend
gofmt -w .
go vet ./...
go build ./...
go test ./...
go run ./cmd/seed --reset --confirm-reset   # re-seed with premium flags

# Frontend
cd frontend
npm run lint
npm test
npm run build

# Docker (if testing containerized)
docker compose up -d --build
docker compose run --rm seed --reset --confirm-reset
curl http://127.0.0.1/health
```

---

## 11. Midtrans sandbox testing

1. Get sandbox keys from https://dashboard.midtrans.com → Settings →
   Access Keys. Keys start with `SB-Mid-`.
2. Set env:
   ```
   MIDTRANS_SERVER_KEY=SB-Mid-server-xxx
   MIDTRANS_CLIENT_KEY=SB-Mid-client-xxx
   MIDTRANS_ENV=sandbox
   PREMIUM_PRICE_IDR=99000
   ```
3. Set webhook URL in Midtrans dashboard → Settings → Configuration →
   Payment Notification URL:
   ```
   http://<your-vps-ip>/api/v1/payments/notification
   ```
   (For local dev, use ngrok: `ngrok http 8080` → use the ngrok URL.)
4. Register a test user, log in, visit `/upgrade`, click "Upgrade now".
5. In the Snap sandbox popup, use the test payment methods (QRIS
   simulator, bank transfer simulator, test card 4811111111111114).
6. After payment, the webhook fires → backend verifies → user's
   `subscription_tier` flips to `premium` → HomePage premium sections
   unlock.

---

## 12. File inventory

### New files this session

**Backend**:
- `backend/services/fsrs.go` + `fsrs_test.go`
- `backend/services/review_service.go`
- `backend/services/review_handler.go` (handlers/review_handler.go)
- `backend/services/payment_service.go` + `payment_service_test.go`
- `backend/api/handlers/payment_handler.go`
- `backend/services/testhelper_test.go`
- `backend/services/session_service_test.go`
- `backend/services/review_service_test.go`
- `backend/services/quiz_service_test.go`
- `backend/services/gating_service_test.go`
- `backend/services/auth_service_test.go`
- `backend/Dockerfile` + `.dockerignore`

**Frontend**:
- `frontend/src/pages/ReviewPage.jsx`
- `frontend/src/pages/UpgradePage.jsx`
- `frontend/src/lib/errors.js` + `errors.test.js`
- `frontend/src/lib/api.test.js`
- `frontend/src/context/AuthContext.test.jsx`
- `frontend/src/pages/HomePage.test.jsx`
- `frontend/src/test/setup.js`
- `frontend/Dockerfile` + `.dockerignore` + `nginx.conf`

**Root**:
- `docker-compose.yml`
- `.env.docker.example`
- `DEPLOY.md`

### Modified files this session

**Backend**:
- `backend/models/base.go` — removed `default:gen_random_uuid()` (DB-portable)
- `backend/models/user.go` — `SubscriptionTier`, `PremiumUntil`,
  `PreferredGloss`
- `backend/models/curriculum.go` — `Section.IsPremium`, `VocabItem.IndonesianMeaning`
- `backend/models/payment.go` — new `Payment` model
- `backend/services/session_service.go` — FSRS integration, StartSession,
  ResumeSession, Rating support
- `backend/services/quiz_service.go` — required-skills fix
  (compatibility-based)
- `backend/services/gating_service.go` — `IsPrivilegedUser` (exported),
  premium-section check, `GateResult.RequiresPremium`
- `backend/services/auth_service.go` — preferences endpoints
- `backend/api/handlers/session_handler.go` — StartSession, ResumeSession
- `backend/api/handlers/curriculum_handler.go` — premium gating, gloss
- `backend/api/handlers/auth_handler.go` — preferences handlers
- `backend/api/middleware/gating.go` — `UPGRADE_REQUIRED` code
- `backend/router/router.go` — session + review + payment routes
- `backend/db/db.go` — Payment in AutoMigrate
- `backend/cmd/seed/main.go` — `--reset` flag, paged/day seeders, premium
  flags, romaji transliterator
- `backend/cmd/seed/romaji.go` + `romaji_test.go`
- `backend/go.mod` — glebarez/sqlite, midtrans-go, go 1.23

**Frontend**:
- `frontend/src/App.jsx` — /review + /upgrade routes
- `frontend/src/pages/HomePage.jsx` — premium badges, upgrade overlay,
  review widget, gloss toggle, friendly errors
- `frontend/src/pages/LearnPage.jsx` — bilingual gloss, SRS review link
- `frontend/src/pages/QuizPage.jsx` — submitResults fix
  (skill_type + actual-performance is_correct)
- `frontend/src/pages/SectionQuizPage.jsx` — lint fix
- `frontend/src/context/AuthContext.jsx` — `refreshUser`
- `frontend/src/components/Navbar.jsx` — Premium pill / Upgrade link
- `frontend/package.json` — test scripts, vitest + RTL devDeps
- `frontend/vite.config.js` — vitest test config

**Root**:
- `AGENTS.md` — Go 1.23, test commands, payment flow, backend test
  convention
- `DEPLOY.md` — Docker Path A, bare-metal Path B, Midtrans section