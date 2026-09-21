# API Reference

The authoritative, always-up-to-date list of every endpoint, its middleware, and its handler is
`backend/internal/routes/routes.go` — routes, auth requirements, and role restrictions are all declared
together in that one file by design, specifically so this kind of doc can't silently drift out of sync
with reality. Read that file for the exact current API surface.

## Response envelope

Every endpoint returns one of these two shapes (`backend/internal/utils/response.go`):

```json
// success
{ "success": true, "data": { ... }, "message": "optional human-readable message" }

// error
{ "success": false, "error": { "code": "VALIDATION_ERROR", "message": "...", "details": { ... } } }
```

Error codes (`backend/internal/utils/errors.go`): `VALIDATION_ERROR`, `UNAUTHORIZED`, `FORBIDDEN`,
`NOT_FOUND`, `CONFLICT`, `INTERNAL_ERROR`, `DATABASE_ERROR`, `AI_PROVIDER_ERROR`, `FILE_ERROR`,
`BAD_REQUEST`, `RATE_LIMITED`.

## Authentication

- `POST /api/v1/auth/register/candidate`, `POST /api/v1/auth/register/company` — register
- `POST /api/v1/auth/login` — email/password login
- `GET /api/v1/auth/google`, `GET /api/v1/auth/google/callback` — Google OAuth flow
- `POST /api/v1/auth/refresh` — rotate the refresh-token cookie for a new access token
- `POST /api/v1/auth/logout` — revoke the current refresh token
- `GET /api/v1/auth/me` — current authenticated user (requires `Authorization: Bearer <token>`)

Access tokens: pass as `Authorization: Bearer <token>`. Refresh tokens: automatically sent as an
httpOnly cookie scoped to `/api/v1/auth`; never exposed to JavaScript.

## Everything else

See `docs/../backend/internal/routes/routes.go` for candidates, companies, jobs, applications, matching,
and interview endpoints — each grouped by resource with its exact middleware chain visible inline.
