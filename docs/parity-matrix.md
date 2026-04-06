# Parity Matrix

| Capability | `express-notes` repo | Online series / notes found online | `go-notes` target |
| --- | --- | --- | --- |
| Auth | Local username/password with cookie-backed JWT-like token rotation | Later series adds auth-related topics | OIDC + PKCE, opaque Valkey-backed session cookie |
| Auth session validation | `POST /auth/me` | Present in repo and series | `GET /api/v1/auth/me` |
| Logout | Cookie/session removal | Present in repo and series | `POST /api/v1/auth/logout` |
| Note CRUD | Create, read, update, delete | Present in repo and series | Create, read, patch, delete |
| Shared notes | Public lookup by short URL | Present in repo | `GET /api/v1/notes/shared/{slug}` |
| Search | Client-side Vuex filtering | Series later covers server filtering/search | Server-side `search` query parameter |
| Pagination | Not present in cloned repo | Covered in series | Server-side `page` and `page_size` |
| Sorting | Not present in cloned repo | Covered in series | Server-side `sort` and `order` |
| Filtering | Archived in repo list query only | Covered in series | `status`, `shared`, `tag`, date filters |
| Validation | Minimal | Covered in series | Central JSON validation + field errors |
| Database | MongoDB + Mongoose | MongoDB in the series | PostgreSQL + `sqlc` + `pgx/v5` |
| Cache | None | None | Valkey for OIDC state, sessions, and note caches |
| Frontend | Nuxt app | Nuxt/video-driven walkthrough | API-only project with extensive docs |

## Intentional differences

- `PATCH` replaces the old “send the whole note with PUT” style.
- Timestamps use typed UTC `time.Time` values instead of numeric JS timestamps.
- Optional DB fields are represented as Go pointers instead of loose objects.
- Search, pagination, sorting, and filters live in the API instead of the browser store.
