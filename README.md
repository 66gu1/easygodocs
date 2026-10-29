# EasyGoDocs

[![CI](https://github.com/66gu1/easygodocs/actions/workflows/ci.yaml/badge.svg)](https://github.com/66gu1/easygodocs/actions/workflows/ci.yaml)

⚠️ **Status:** demo project. It is intended to demonstrate Go backend development and architecture design.
It is **not production-ready**, see [Known limitations](#-known-limitations).

EasyGoDocs is a wiki backend: a tree of departments and articles with versioning, personal drafts,
hierarchical role-based access control and JWT authentication with server-side sessions.

---

## ✨ Features
- Go, PostgreSQL, chi, GORM, goose migrations, Swagger
- Entity tree with depth limit and cycle prevention that stays consistent under concurrent requests
- Versioning of every change and personal drafts with conflict detection on publish
- Hierarchical RBAC: permissions are granted on a subtree
- JWT access tokens + server-side sessions: logout, password change and user deletion take effect immediately
- Unit tests with mocks, integration tests on a real PostgreSQL (testcontainers), race tests (coverage: **~82%**)
- CI: lint, tests with `-race`, govulncheck

---

## 🚀 Quick Start

Requirements: Docker and Docker Compose.

```bash
docker compose up --build
```

This starts PostgreSQL, applies the migrations, creates the admin user (`admin@example.com` / `changeme`)
and starts the API on `http://localhost:8080/api/v1`.
Swagger UI: http://localhost:8080/api/v1/swagger/index.html

### Walkthrough

The flow below uses `curl` and `jq`. Authenticated requests need the `Authorization: Bearer <access_token>` header.

```bash
API=http://localhost:8080/api/v1
JSON='Content-Type: application/json'

# log in as admin
ADMIN=$(curl -s -X POST $API/login -H "$JSON" \
  -d '{"email":"admin@example.com","password":"changeme"}' | jq -r .access_token)

# create a root department (only admins can create root entities)
DEP=$(curl -s -X POST $API/entities -H "$JSON" -H "Authorization: Bearer $ADMIN" \
  -d '{"type":"department","name":"Engineering","content":"Team docs"}' | jq -r .id)

# register a user and give them write access to the department and its subtree
curl -s -X POST $API/register -H "$JSON" \
  -d '{"email":"bob@example.com","name":"Bob","password":"password123"}'
BOB_LOGIN=$(curl -s -X POST $API/login -H "$JSON" -d '{"email":"bob@example.com","password":"password123"}')
BOB=$(echo "$BOB_LOGIN" | jq -r .access_token)
BOB_ID=$(echo "$BOB" | jq -rR 'split(".")[1] | @base64d | fromjson | .sub')   # user ID from the token
curl -s -X POST $API/roles -H "$JSON" -H "Authorization: Bearer $ADMIN" \
  -d "{\"user_id\":\"$BOB_ID\",\"role\":\"write\",\"entity_id\":\"$DEP\"}"

# Bob writes an article: it is created as version 1
DOC=$(curl -s -X POST $API/entities -H "$JSON" -H "Authorization: Bearer $BOB" \
  -d "{\"type\":\"article\",\"name\":\"Onboarding\",\"content\":\"v1\",\"parent_id\":\"$DEP\"}" | jq -r .id)

# Bob prepares changes in a draft: readers still see version 1
curl -s -X PUT $API/entities/$DOC/draft -H "$JSON" -H "Authorization: Bearer $BOB" \
  -d '{"name":"Onboarding","content":"v2: updated"}'
curl -s $API/entities/$DOC -H "Authorization: Bearer $ADMIN" | jq .content   # "v1"

# publishing the draft creates version 2
curl -s -X POST $API/entities/$DOC/draft/publish -H "Authorization: Bearer $BOB"
curl -s $API/entities/$DOC/versions -H "Authorization: Bearer $ADMIN" | jq '.[] | {current_version, content}'

# the tree shows only what the user may read
curl -s $API/entities -H "Authorization: Bearer $BOB" | jq

# refresh the tokens: the refresh token is rotated and can be used only once
SESSION=$(echo "$BOB_LOGIN" | jq -r .refresh_token.session_id)
TOKEN=$(echo "$BOB_LOGIN" | jq -r .refresh_token.token)
BOB=$(curl -s -X POST $API/refresh -H "$JSON" \
  -d "{\"session_id\":\"$SESSION\",\"token\":\"$TOKEN\"}" | jq -r .access_token)

# log out: the access token stops working immediately, not when it expires
curl -s -o /dev/null -w '%{http_code}\n' -X DELETE "$API/sessions/$SESSION?user_id=$BOB_ID" \
  -H "Authorization: Bearer $BOB"                                               # 204
curl -s -o /dev/null -w '%{http_code}\n' $API/entities -H "Authorization: Bearer $BOB"   # 401
```

---

## 📦 Domain

### Entities
- **department** – an organizational node. Can contain departments and articles and has its own content.
- **article** – a content node. Can contain other articles, but not departments, and must have a parent.

The tree depth is limited (`entity.max_hierarchy_depth`, 15 by default).
Moving an entity under one of its descendants is rejected.

### Versioning and drafts
- Every published change creates a new version; all versions are kept and can be read via the API.
- Changes can be prepared in a **draft** without affecting readers. Every user with write permission
  has their own draft of an entity (`/entities/{id}/draft`); readers keep seeing the published version.
- Publishing a draft creates a new version. If someone else has published the entity since the draft was started,
  publishing fails with `409 Conflict` instead of silently overwriting their changes.
- A new entity can be created unpublished (`"is_draft": true`). It is visible only to its creator and admins
  and can't have children until it is published.

### Roles
- **admin** – global, unrestricted access.
- **write** – create, update and delete entities in the subtree it was granted on. Includes `read`.
- **read** – view entities in the subtree it was granted on and the path to it (its ancestors).

`read` and `write` are granted on an entity and apply to its whole subtree. Root entities can be created only by admins.

---

## 🧭 Design decisions

### Layers
`transport/http` → `usecase` → domain core → `repo/gorm`, one package per domain (`auth`, `user`, `entity`).
- **transport** decodes requests and maps errors to HTTP statuses;
- **usecase** does authorization and orchestration;
- **core** holds domain rules and validation;
- **repo** is SQL only.

Interfaces are declared by the consumer, so every layer is tested with generated mocks (minimock),
and the repositories are tested against a real PostgreSQL.

### Tree consistency under concurrency
Validating a move ("the new parent is not my descendant", "the depth stays within the limit") and applying it
are two steps, so two concurrent moves can each pass validation and together create a cycle.
- Structural changes (create with a parent, move, delete) run in a transaction and are serialized
  with a PostgreSQL advisory lock (`pg_advisory_xact_lock`). Content-only edits don't take it.
- Version numbers are assigned under a row lock (`SELECT ... FOR UPDATE`), so concurrent edits get sequential
  versions instead of failing on a duplicate key.
- Permissions are checked against the parent the caller saw; if the entity was moved in between,
  the update fails with `409 Conflict`.

Each of these races has an integration test that runs the requests concurrently; the tests fail without the locks.

### Permission check
A permission is granted on an entity and applies to its subtree. Instead of loading all permitted subtrees,
which for a user with access to a root department is the whole wiki, the check loads only the ancestors
of the checked entity (at most `max_hierarchy_depth` rows) and looks for a permitted one among them.
An integration test checks that the result is the same as with loading the subtrees.

### Drafts
A draft is a separate row per (entity, user) that stores the version it was started from, so publishing
can detect a concurrent publish without locking the entity while someone is editing.
Unpublished entities can't have children, otherwise published entities would be hidden together with their parent.

### Authentication
- Short-lived JWT access tokens (15 minutes, HS256 only, `exp`, `iss` and `iat` are checked) plus a server-side session
  with a rotating refresh token.
- The session is checked on every request. This costs one query per request, but logout, session revocation,
  password change and user deletion take effect immediately instead of after the token expires.
- Refresh tokens are random 256-bit values, so they are stored as SHA-256 hashes (a slow hash like bcrypt adds nothing
  for high-entropy secrets) and compared in constant time. Rotation is a compare-and-swap on the old hash,
  so a refresh token can be used only once.
- Login for an unknown email still runs bcrypt against a dummy hash, so the response time doesn't reveal
  which emails are registered. Public endpoints are rate limited per IP.

### Errors and logging
Errors are typed (`apperr`): a class that maps to an HTTP status, a stable code and field violations for the client,
and internal details that are never sent to the client. Lower layers only wrap errors with context;
every failed request is logged exactly once, in `httpx.ReturnError`, with the full error chain, request ID and user ID.

---

## ⚠️ Known limitations
- **Rate limiting trusts `X-Forwarded-For`.** `middleware.RealIP` takes the client IP from the request headers,
  so the per-IP limit can be bypassed unless the service runs behind a proxy that overwrites them.
- **No refresh token reuse detection.** A reused refresh token is rejected, but the session is not revoked,
  so a stolen token that was used first keeps working.
- **No graceful shutdown**, health endpoints or `ReadHeaderTimeout`.
- **No pagination:** the tree and version lists are returned in full.
- An outdated draft can't be merged: it has to be discarded and started again from the current version.
- Soft-deleted entities and expired sessions are never purged.

---

## 🛠️ Development

```bash
make tools       # install gofumpt, goimports, golangci-lint, govulncheck
make lint        # golangci-lint
make test        # all tests with -race; integration tests start PostgreSQL in Docker
make cover       # tests with coverage
make generate    # mocks and Swagger docs
make run         # run the server locally (needs DATABASE_DSN and JWT_SECRET, see .env.example)
```

Integration tests are behind the `testutil` build tag (`go test -tags testutil ./...`) and need Docker.

### Environment
- `DATABASE_DSN` – PostgreSQL connection string
- `JWT_SECRET` – key for signing JWT tokens, at least 32 bytes (the server refuses to start otherwise)
- `ADMIN_EMAIL`, `ADMIN_PASSWORD` – credentials of the initial admin created by `cmd/seedadmin`

⚠️ The values in `docker-compose.yml` are for the demo only.

Other settings (token TTLs, rate limit, DB pool, validation limits) are in `config/config.yaml`.

### Project layout
- `cmd/server` – API service
- `cmd/seedadmin` – creates the initial admin user
- `internal/app/{auth,user,entity}` – domains: `core`, `usecase`, `transport/http`, `repo/gorm`
- `internal/infrastructure` – errors, HTTP helpers, logging, DB transactions, crypto
- `migrations/` – goose migrations
- `config/` – configuration

---

## 📄 License
MIT – see [LICENSE](./LICENSE).
