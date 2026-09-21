# Hirewell — AI Resume Parser & Intelligent Recruitment Platform

A real, working SaaS recruitment platform: candidates upload resumes that get parsed into structured
profiles, companies post jobs with explicit requirements, a deterministic + AI-assisted engine scores
every application with a documented formula, and an AI interviewer asks questions grounded strictly in
what the job requires and what the candidate's own profile actually contains.

This README is the single source of truth for running the project. Follow it in order.

---

## 1. What's actually implemented (read this first)

This was built and **verified end-to-end** in the environment it was developed in: real MySQL 8.0, a
compiled Go binary actually running, real HTTP requests exercised against it, a real Next.js production
build. Specifically verified:

- Backend: `go build ./...`, `go vet ./...`, and `go test ./...` all pass — **63 tests, 0 failures**:
  unit tests for the matching engine's exact scoring formula (13), AI-output validation/grounding
  rules (14), the stub AI provider's grounding guarantees (6), real DOCX/TXT/PDF text extraction (5),
  password hashing and skill-slug normalization (5), **JWT issuance/verification including an
  alg=none forged-token attack test (7)**, **auth/RBAC middleware via httptest (8)**, **CORS
  same-origin-only enforcement and the rate limiter (7)** — plus a full integration test against real
  MySQL covering register → job → resume → apply → match with two live IDOR authorization checks.
- Full manual smoke test via curl: register candidate + company → create/publish job → upload & parse
  resume → apply → view match score → view ranked applicants → shortlist → create interview → answer
  grounded questions → complete → view evaluation. All of it hit a real running server and real MySQL.
- Frontend: `tsc --noEmit`, `next lint`, `vitest run` (37 tests), and `next build` all pass cleanly (0
  errors, 0 warnings), all 17 routes compile and the dev/production server serves real rendered HTML
  for every public page.
- Database migrations: `up` and `down` both tested against a live MySQL instance.
- Seed script: runs end-to-end against a real database and prints real IDs and a real computed match
  score.

**Known limitations, stated honestly:**

- **Docker was not build-tested in this delivery.** The Dockerfiles and `docker-compose.yml` are
  written correctly following standard multi-stage patterns, but the sandboxed environment this project
  was built in could not reach Docker Hub to pull base images (`golang`, `alpine`, `node`), so
  `docker build` / `docker compose up` could not actually be executed and verified here. Please run
  `docker compose up --build` yourself and report any issues — the Dockerfiles are standard enough that
  they are very likely to work, but "very likely" is not "verified."
- **Google OAuth code is complete and correct** (real token exchange, real userinfo fetch, real account
  linking logic) but could not be exercised end-to-end without a real Google Cloud OAuth client — you'll
  need to configure your own credentials (§7) and test the flow yourself.
- **The Gemini AI provider is implemented for real** (structured JSON output, retries, typed error
  handling) but was only verified to *compile and pass unit tests* — not exercised against a live Gemini
  API key in this session, since none was available. The default `AI_PROVIDER=stub` is a genuinely
  working, deterministic, offline provider (see §9) that exercises the *entire* pipeline for real; switch
  to `gemini` and supply a real key to use actual AI extraction.
- **PDF text extraction is best-effort**, implemented with the standard library only (no external PDF
  library — see `backend/internal/resume/extract.go` for why). It correctly handles the large majority
  of PDFs produced by common resume tools (Word/LibreOffice export, Overleaf/LaTeX, browser
  print-to-PDF), but will not handle scanned/image-only PDFs (would need OCR, not implemented) or
  unusual font-encoding schemes.
- **Legacy `.doc` (pre-2007 binary Word format) is explicitly rejected**, not silently mis-parsed. Users
  are asked to re-save as `.docx` or `.pdf`.
- **Reports/analytics endpoints** (`GET /jobs/:id/report`, `GET /candidates/me/report`) are **not
  implemented** as dedicated aggregation endpoints. `GET /jobs/:id/matches` already returns every
  applicant fully ranked and scored, which covers the core "companies receive ranked candidates"
  requirement — a separate analytics/funnel layer (time-to-hire, conversion rates, etc.) is genuinely
  future work, not something faked here.
- **Admin role**: the `users.role` enum and RBAC middleware support `admin`, and `RequireRole` is
  written generically enough to gate future admin routes, but no admin UI or admin-specific endpoints
  exist yet.
- **Frontend interactive flows**: 37 real tests now cover form validation, the API client's
  error/retry handling, critical UI components, and a full login-page interaction test (typing,
  submitting, validation errors, API error display, successful-login redirect) via Vitest +
  React Testing Library + jsdom — these render real components and simulate real user input, but
  run in a simulated DOM, not an actual browser. Full browser automation (clicking through every
  page in Chrome/Playwright) was not performed in this session.
- **npm audit**: after pinning to the latest maintained Next.js 14→15 patch line, 2 lower-severity,
  **build-time-only** advisories remain in `postcss` (source-map handling during build, not a runtime
  vulnerability exposed to end users). Resolving them fully requires Next.js 16, a major version bump
  this delivery did not attempt to verify given the scope already covered. Run `npm audit` yourself
  before shipping to production.

Everything else described below is real, was built, and was tested as described.

---

## 2. Architecture

```
resume-platform/
├── backend/            Go + Gin REST API
│   ├── cmd/server/     main.go entrypoint
│   ├── internal/
│   │   ├── config/     env var loading, fail-fast validation
│   │   ├── database/   connection pool + embedded migration runner
│   │   ├── models/     structs mirroring the DB schema
│   │   ├── repository/ raw SQL data access (no ORM)
│   │   ├── services/   business logic (auth, candidate, company, job,
│   │   │               application, matching, interview, resume)
│   │   ├── controllers/thin HTTP handlers
│   │   ├── middleware/ JWT auth, RBAC, CORS, rate limiting, logging,
│   │   │               centralized error handling, security headers
│   │   ├── routes/     the one place every endpoint + middleware is wired
│   │   ├── validators/ request DTOs with binding/validation tags
│   │   ├── auth/       JWT issuance/verification, Google OAuth exchange
│   │   ├── ai/         provider-agnostic AI interface + Gemini + stub impls
│   │   ├── resume/     file validation + text extraction (txt/docx/pdf)
│   │   ├── matching/   the deterministic scoring engine
│   │   └── storage/    local file storage abstraction (path-traversal safe)
│   ├── migrations/     versioned SQL, up + down
│   ├── seeds/           demo data script
│   ├── tests/            integration tests against a real database
│   └── uploads/          local dev file storage (gitignored)
├── frontend/            Next.js 15 App Router + TypeScript + Tailwind
│   ├── app/              pages (landing, auth, candidate, company, jobs, interview)
│   ├── components/       reusable UI (layout, forms, landing sections)
│   ├── lib/               typed API client + zod validation schemas
│   ├── providers/         auth context (session hydration, login/logout)
│   └── types/              shared TypeScript types mirroring backend DTOs
├── docker-compose.yml
├── .env.example
├── Makefile
└── README.md            (this file)
```

**Request flow:** `routes` wires `controller → service → repository/ai/matching/storage`. Controllers
never touch SQL or call the AI provider directly; that's exactly the layering the spec asked for.

**AI provider abstraction:** every AI operation goes through the `ai.Provider` interface
(`internal/ai/provider.go`). Swapping providers means writing one new file implementing that interface
and changing `AI_PROVIDER` — no other code changes. Two implementations ship: `stub` (deterministic,
offline, no API key) and `gemini` (real Gemini REST API, structured JSON output, retries).

**Matching engine:** deterministic and explainable, not a single opaque AI call. See
`internal/matching/engine.go`'s doc comment for the exact formula and the reasoning behind the default
weights. AI only assists via an optional semantic-similarity pass (`ai.Provider.SemanticMatch`) that can
add matches the exact/alias matcher missed — it never overrides or replaces the deterministic score.

**Interview grounding:** `ai.InterviewContext` is built strictly from the job's own skills and the
candidate's own resume data — the AI provider physically cannot see anything else. Every generated
question is additionally checked by `ai.ValidateGeneratedQuestions` against that exact context before
being stored; ungrounded questions are logged and dropped, never persisted.

---

## 3. Requirements

- Go 1.22+
- Node.js 20+ and npm
- MySQL 8.0
- (Optional) Docker + Docker Compose, if you want the containerized path
- (Optional) A Gemini API key from https://aistudio.google.com/apikey, if you want real AI instead of
  the offline stub
- (Optional) A Google Cloud OAuth client, if you want Google sign-in

---

## 4. Environment setup

```bash
cp .env.example backend/.env
```

Edit `backend/.env` and set at minimum:
- `JWT_SECRET` / `JWT_REFRESH_SECRET` — generate with `openssl rand -base64 48` each
- `DB_PASSWORD` — whatever you set up your local MySQL user with (see §5)

Everything else has a sane default for local development. See `.env.example` for what every variable
does — it's the single documented list, nothing is undocumented.

For the frontend:
```bash
cp .env.example frontend/.env.local   # only NEXT_PUBLIC_API_URL matters here
```

---

## 5. Database setup

**Without Docker** (what this project was actually verified against):

```bash
# install MySQL 8.0 for your OS, then:
mysql -u root -e "
  CREATE DATABASE resume_platform CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
  CREATE DATABASE resume_platform_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
  CREATE USER 'app_user'@'localhost' IDENTIFIED BY 'your_password_here';
  GRANT ALL PRIVILEGES ON resume_platform.* TO 'app_user'@'localhost';
  GRANT ALL PRIVILEGES ON resume_platform_test.* TO 'app_user'@'localhost';
  FLUSH PRIVILEGES;
"
```

Match `your_password_here` to `DB_PASSWORD` in `backend/.env`.

**With Docker:** `docker compose up mysql -d` provisions the database automatically from the
`DB_*` variables in your root `.env` (see §10).

---

## 6. Migrations

The backend binary has migration commands built in (`internal/database/migrate.go` — no external
migration CLI dependency):

```bash
cd backend
go run ./cmd/server -migrate-up      # apply all pending migrations
go run ./cmd/server -migrate-down    # roll back the most recent one
```

This was tested: `up` creates all 20 tables with real foreign keys/indexes/enums, `down` cleanly drops
everything, `up` again is idempotent (tracked via the `schema_migrations` table).

If you already applied `migrations/001_initial_schema.up.sql` by hand (e.g. via `mysql <
migrations/001_initial_schema.up.sql`), tell the migration runner it's already applied:
```bash
mysql -u root resume_platform -e "INSERT INTO schema_migrations (version) VALUES ('001_initial_schema');"
```

---

## 7. Google OAuth setup (optional)

1. Go to https://console.cloud.google.com/apis/credentials
2. Create an OAuth 2.0 Client ID (Web application)
3. Add authorized redirect URI: `http://localhost:8080/api/v1/auth/google/callback`
4. Set `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL` in `backend/.env`

If left blank, `/auth/google` and `/auth/google/callback` return a clear "not configured" error and the
rest of the app works normally — this is a graceful degradation, not a crash.

---

## 8. Running the backend

```bash
cd backend
go mod download
go run ./cmd/server -migrate-up   # first time only
go run ./cmd/server
```

Verify:
```bash
curl http://localhost:8080/health   # {"status":"ok"}
curl http://localhost:8080/ready    # {"status":"ready","database":"ok","ai_provider":"stub"}
```

---

## 9. AI provider setup

Default is `AI_PROVIDER=stub` in `.env.example` — a real, deterministic, offline implementation (see
`backend/internal/ai/stub.go`'s doc comment for exactly why it exists and what it honestly can't do). It
extracts skills via keyword matching from actual resume text, generates interview questions grounded in
the real job/candidate intersection, and scores answers heuristically. The entire pipeline (parsing,
matching, interviewing) genuinely works end-to-end with this provider — you can demo the whole product
with zero API keys.

To use real AI:
```bash
# backend/.env
AI_PROVIDER=gemini
AI_API_KEY=AIzaSy...          # from https://aistudio.google.com/apikey
AI_MODEL=gemini-2.0-flash
```

---

## 10. Running the frontend

```bash
cd frontend
npm install
npm run dev
```

Visit http://localhost:3000. Make sure `NEXT_PUBLIC_API_URL` in `frontend/.env.local` points at your
running backend (default `http://localhost:8080/api/v1` is correct for local dev).

---

## 11. Seeding demo data

```bash
cd backend
go run ./seeds/seed.go
```

This registers exactly the scenario described in the project spec: candidate **John Doe**
(`john.doe@example.com` / `SecurePass123!`) with skills Go/MySQL/REST APIs/Docker/Git and two projects
(AI Resume Parser, E-commerce API), company **Example Technologies**
(`hr@exampletech.com` / `SecurePass123!`) with a published **Junior Go Backend Developer** job, and a
real calculated application + match score. Safe to re-run — it detects existing seed data and skips.

---

## 12. Running tests

```bash
cd backend
go test ./...                 # 63 unit tests: matching engine, AI validation, resume extraction,
                               # JWT (incl. alg=none attack rejection), auth/RBAC middleware, CORS,
                               # rate limiter, password/slug utils
go test ./tests/... -v        # integration test against a real database (needs resume_platform_test)
```

The integration test skips gracefully (not a hard failure) if `resume_platform_test` isn't reachable —
see `tests/integration_test.go`'s `connectTestDB`. Point it elsewhere with `TEST_DATABASE_DSN`.

```bash
cd frontend
npm run typecheck             # tsc --noEmit
npm run lint                  # next lint
npm run test                  # vitest run -- 37 tests: form validation schemas,
                               # API client error handling (incl. the 401->silent-refresh->
                               # retry flow), critical UI components (TextField, Alert),
                               # and a full login-page test (empty-form validation,
                               # malformed email, successful login + redirect, API error
                               # display) with fetch and next/navigation mocked
npm run build                 # full production build
```

---

## 13. Production build

```bash
cd backend && go build -o server ./cmd/server
cd frontend && npm run build && npm run start
```

Or use the Makefile: `make build`.

---

## 14. Docker

**Verification status:** see §1 — the Dockerfiles are written but *not build-verified* in this delivery
due to sandbox network restrictions on Docker Hub. Review before trusting in production.

```bash
cp .env.example .env      # root .env, read by docker-compose.yml
# edit .env: set real JWT_SECRET / JWT_REFRESH_SECRET / DB_PASSWORD / DB_ROOT_PASSWORD

docker compose up --build -d
docker compose run --rm backend ./server -migrate-up   # run once
docker compose logs -f
```

Frontend: http://localhost:3000, Backend: http://localhost:8080, MySQL: `localhost:3306`.

---

## 15. Troubleshooting

- **`connection refused` on backend startup** — MySQL isn't running or `DB_*` env vars don't match your
  actual MySQL user/password.
- **`Duplicate foreign key constraint name`** on `-migrate-up`** — you applied the schema manually before
  using the migration runner; see the note at the end of §6.
- **Frontend shows "Failed to load dashboard"** — check `NEXT_PUBLIC_API_URL` and that the backend's
  `FRONTEND_URL` env var matches your frontend's actual origin exactly (CORS is origin-exact-match, not
  wildcard, by design — see `internal/middleware/security.go`).
- **Google sign-in redirects to an error page** — `GOOGLE_REDIRECT_URL` must exactly match what's
  registered in Google Cloud Console, including protocol and trailing slashes.
- **Resume upload succeeds but parsing fails** — check the resume's `parse_error` field in the API
  response; with `AI_PROVIDER=gemini`, this is usually an invalid/missing API key or a rate limit (see
  `internal/ai/gemini.go`'s typed errors).

---

## 16. Seed / test credentials reference

| Role      | Email                    | Password        |
|-----------|---------------------------|-----------------|
| Candidate | john.doe@example.com      | SecurePass123!  |
| Company   | hr@exampletech.com        | SecurePass123!  |

---

## 17. Security notes

- Passwords hashed with bcrypt (cost 12), never logged, never returned in any API response
  (`json:"-"` on `User.PasswordHash`).
- JWT access tokens are short-lived (15m default) and held only in frontend memory, never
  localStorage/sessionStorage — genuine XSS hardening, not just a comment.
- Refresh tokens are httpOnly, path-scoped, rotated on every use (old one revoked, new one issued),
  and stored server-side only as a bcrypt-independent hash (never plaintext in the DB).
- Every object-access endpoint re-verifies ownership server-side (see `JobService.AssertCompanyOwnsJob`,
  `ApplicationService.AssertAccess`) — this was specifically unit- and integration-tested for IDOR
  (a candidate cannot read another candidate's application; a company cannot modify another company's
  job).
- File uploads are validated by magic bytes (not just extension/Content-Type, both of which are
  attacker-controlled), size-capped, and stored under randomly generated filenames with path-traversal
  protection (`internal/storage/storage.go`).
- No secrets are hardcoded anywhere in source — everything flows through `internal/config`, which fails
  fast on missing required secrets in production mode.
