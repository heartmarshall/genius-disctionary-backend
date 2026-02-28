# Roles and Access Control

## Roles

| Role | Description | How assigned |
|---|---|---|
| User | Learner who builds dictionaries and studies vocabulary | Automatic on registration |
| Admin | Platform administrator with access to enrichment management and user management | Assigned by another admin via `PUT /admin/users/{id}/role` |

There are only two roles. No concept of "super admin" — all admins have equal privileges.

## Permission Matrix

| Action | User | Admin |
|---|---|---|
| **Dictionary** | | |
| Create/edit/delete own entries | Yes | Yes (own) |
| Search reference catalog | Yes | Yes |
| Import/export entries | Yes | Yes |
| **Study** | | |
| Start/finish study sessions | Yes | Yes |
| Review cards, undo reviews | Yes | Yes |
| View dashboard and statistics | Yes | Yes |
| **Organization** | | |
| Create/manage own topics | Yes | Yes |
| Create/manage own inbox items | Yes | Yes |
| **User Management** | | |
| Update own profile and settings | Yes | Yes |
| View all users | No | Yes |
| Change any user's role | No | Yes |
| **Enrichment (AI Queue)** | | |
| View enrichment queue stats | No | Yes |
| View enrichment queue items | No | Yes |
| Retry failed enrichments | No | Yes |
| Reset stuck "processing" items | No | Yes |
| Manually enqueue words for enrichment | No | Yes |

## Access Control Implementation

### Authentication
- All API requests (except health check) require a valid JWT access token in the `Authorization: Bearer` header
- Anonymous requests receive no user context — most operations will return 401
- JWT contains: user ID (subject), role, issuer, expiry

### Authorization Checks
- **User-level**: Every data operation scopes queries by `user_id` from the JWT context. Users can only see and modify their own data. There is no cross-user data access.
- **Admin-level**: Admin endpoints check `ctxutil.IsAdminCtx(ctx)` — returns 403 Forbidden if role is not `admin`

### Self-Protection
- **An admin cannot demote themselves** — the system prevents an admin from changing their own role. This avoids accidentally removing the last admin.

**Code**:
- Role definitions: `internal/domain/enums.go:135-155`
- Admin check middleware: `internal/transport/rest/admin.go`
- Admin service operations: `internal/service/user/admin.go`
- Auth middleware: `internal/transport/middleware/auth.go`
- Context utilities: `pkg/ctxutil/ctxutil.go`

## Data Isolation

All user data is strictly isolated:
- Dictionary entries, senses, translations, examples → scoped by user_id
- Cards and review logs → scoped by user_id
- Topics, inbox items → scoped by user_id
- Study sessions → scoped by user_id

Only the **reference catalog** is shared across all users (read-only for users).

## Admin Endpoints (REST)

| Endpoint | Purpose |
|---|---|
| `GET /admin/enrichment/stats` | View enrichment queue statistics (pending, processing, done, failed counts) |
| `GET /admin/enrichment/queue` | List enrichment queue items |
| `POST /admin/enrichment/retry` | Retry failed enrichment items |
| `POST /admin/enrichment/reset-processing` | Reset stuck "processing" items back to "pending" |
| `POST /admin/enrichment/enqueue` | Manually enqueue a reference entry for AI enrichment |
| `GET /admin/users` | List all users |
| `PUT /admin/users/{id}/role` | Change a user's role (user ↔ admin) |
