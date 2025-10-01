# EasyGoDocs

⚠️ **Status:** demo project. This repository is intended to demonstrate Go development skills and architecture design. It is **not production-ready**.

EasyGoDocs is a simple wiki system written in Go. It supports hierarchical entities, versioning, and role-based access control (RBAC).

---

## ✨ Features
- Go + PostgreSQL
- JWT authentication with session management
- Hierarchical entities with depth validation and cycle prevention
- Article versioning and draft support
- Integration and unit tests (coverage: **81.6%**)
- CI/CD with GitHub Actions

---

## 🚀 Quick Start

### Requirements
- Docker & Docker Compose

---

### Run

```bash
docker compose up --build
```
---
## Entities
The system defines two types of entities:

- **department** – organizational node.
    - Can contain both departments and articles.
    - May itself belong to another department.
    - Has content and supports versioning, just like articles.

- **article** – content node.
    - Can contain other articles, forming article hierarchies.
    - Cannot contain departments.
    - Also supports content editing and versioning.

This enforces the rule:
- departments organize the structure but also may hold their own content,
- articles are purely content nodes but can build their own subtrees of articles.
---

## 📝 Versioning & Drafts

All entities (both departments and articles) support **content versioning**:

- Each update creates a new version, while the previous versions are preserved.
- Any version can be retrieved via the API.
- The latest version is marked as *current*.

Entities can also be saved as **drafts**:
- Drafts are visible only to their creator and to admins.
- Once published, a draft becomes the new current version.
- An already published entity can be moved back to draft **only if it has no child entities**.

This provides an audit trail of changes and allows safe editing workflows while preserving hierarchy consistency.

---

## 🔑 Authentication

- Authentication is based on **JWT tokens** (access + refresh).
- Access tokens are short-lived and required for most API requests.
- Refresh tokens allow obtaining new access tokens without re-login.
- Sessions are stored in the database and can be listed or revoked.

Endpoints for login, refresh and registration are available in the [API section](#-api).

---

## 🔐 Role System

EasyGoDocs implements hierarchical role-based access control (RBAC).

### Roles
- **admin** – global role, unrestricted access to all entities and actions.
- **write** – can create, update, and delete entities within its scope. Includes all `read` rights.
- **read** – can view entities.

### Hierarchy
`read → write → admin`  
Each role includes the permissions of the lower ones.

### Scope
- `read` / `write` are **scoped to a specific entity**:
    - permissions apply to the entity and all its descendants,
    - allow viewing all ancestors up the hierarchy.
- `admin` is **global** — not scoped to an entity.

### Effective Permissions
- If a user has `admin` → access is always granted.
- Otherwise, the highest role across all assignments is applied (`read` or `write`).
- If no matching assignment exists → access is denied.

---

## 📚 API
Interactive API documentation is available via **Swagger UI** after starting the service:

http://localhost:8080/api/v1/swagger/index.html

### Authentication format
Authenticated requests require the following HTTP header:
```
Authorization: Bearer <access_token>
```

Login example:
```
# Login
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"changeme"}'
```
---

## ⚙️ Environment

The following environment variables are required:

- `DATABASE_DSN` – PostgreSQL connection string
- `ADMIN_EMAIL` – initial admin user email
- `ADMIN_PASSWORD` – initial admin user password
- `JWT_SECRET` – secret key for signing JWT tokens

⚠️ Default values are provided in `docker-compose.yml` for demo purposes only.  
In a real deployment, always override them with secure values.

---

## 👤 Admin Seeding

An initial admin user is automatically created during startup via the `seedadmin` service in Docker Compose.

Credentials are taken from environment variables:

- `ADMIN_EMAIL` (default: `admin@example.com`)
- `ADMIN_PASSWORD` (default: `changeme`)

⚠️ These defaults are for demo only. In real deployments, set strong and unique credentials.

This ensures that the system always has at least one administrator after the first launch.

## 🗺️ Roadmap
- Frontend client for the API
- Background job for expired session cleanup
- Permanent deletion of soft-deleted entities
- Extended test coverage (beyond current baseline)
- Anti-abuse protections: CAPTCHA on registration + rate limiting

---

## 🏗️ Architecture Overview
- `cmd/server` – main API service
- `cmd/seedadmin` – bootstrap for the initial admin user
- `internal/app/...` – business logic (user, auth, entity)
- `internal/infrastructure/...` – infrastructure (logging, security, helpers)
- `config/` – application configuration

---

## 📄 License
This project is licensed under the MIT License – see the [LICENSE](./LICENSE) file for details.