# MyEnglish Test Frontend

A hybrid test frontend for MyEnglish Backend v4 that covers 100% of the GraphQL API.

## Quick Start

```bash
npm install
npm run dev
```

The dev server runs at http://localhost:3000 with proxy to backend at http://localhost:8080.

## Authentication

The backend auth endpoints (login/refresh/logout) are not exposed via HTTP yet.
Use the **manual JWT paste** on the login page:

1. Start the backend with required env vars
2. Generate a JWT token (via tests, direct DB, or auth service)
3. Paste the token on the login page

## Pages

| Page | Route | Description |
|------|-------|-------------|
| Login | `/login` | JWT paste + Google OAuth (placeholder) |
| Catalog | `/catalog` | Search reference catalog, preview entries, add to dictionary |
| Dictionary | `/dictionary` | Entry list with filters/sort/pagination, create, import/export, batch delete, trash |
| Entry Detail | `/entry/:id` | Full content editing: senses, translations, examples, images, notes, card |
| Study | `/study` | Dashboard, review flow with SRS grades, sessions, card history/stats |
| Topics | `/topics` | Topic CRUD, entry linking/unlinking |
| Inbox | `/inbox` | Quick notes with pagination, clear all |
| Profile | `/profile` | User info, SRS settings |
| API Explorer | `/explorer` | Raw access to all 53 GraphQL operations |

## API Explorer

The API Explorer page at `/explorer` provides direct access to every GraphQL query and mutation,
organized by domain: RefCatalog, Dictionary, Content, Study, Organization, User.

Each operation has a form with typed fields and shows the raw GraphQL request/response.

## Backend Configuration

The Vite dev server proxies these paths to `http://localhost:8080`:
- `/query` -- GraphQL endpoint
- `/health`, `/ready`, `/live` -- Health checks

To change the backend URL, edit `vite.config.ts`.

Ensure the backend CORS config allows `http://localhost:3000` (default is `*`).

## Tech Stack

- React 18 + TypeScript
- Vite
- Tailwind CSS v4
- React Router v6
- Fetch-based GraphQL client (no Apollo)
