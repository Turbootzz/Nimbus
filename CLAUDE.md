## 🏗️ Project Overview

**Nimbus** is a self-hosted, multi-user homelab dashboard.  
Tech stack:

- **Frontend**: Next.js + React + Tailwind CSS  
- **Backend**: Go with Fiber — RESTful API handling authentication, role-based access, and service health checks  
- **Database**: PostgreSQL — stores users, roles, services/links, user preferences, and service status logs  
- **Deployment**: Docker / docker-compose for local and production deployments  

The MVP includes:  
- User registration/login (JWT-based auth)  
- Role-based access control (admin / user)  
- CRUD for “service links” (name, icon, URL)  
- Health-check polling for services (online/offline, response times)  
- Per-user themeing: background images, light/dark mode  
- Admin-level config via JSON or web UI  

Later features (beyond MVP): live service control, iframe previews, OAuth2 login, plugin widgets.

---

## 📁 Directory Structure (expected)
```bash
nimbus/ #root
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── db/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   ├── models/
│   │   ├── repository/
│   │   ├── services/
│   │   └── utils/
│   ├── go.mod
│   └── Makefile
├── frontend/
│   ├── public/
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── hooks/
│   │   ├── lib/
│   │   └── types/
│   ├── package.json
│   └── tsconfig.json
├── docker/
│   ├── backend.Dockerfile
│   ├── frontend.Dockerfile
│   └── nginx/
├── docker-compose.yml
├── .env.example
└── README.md
```

If Claude ever suggests altering this structure, ask to confirm alignment with existing code before applying changes.

---

## 🧭 Coding & Style Guidelines

- **Go backend**  
  - Use idiomatic Go patterns (structs, methods, composition over inheritance).  
  - Use dependency injection via interfaces (pass dependencies explicitly).  
  - Separate concerns: *handlers* (HTTP) vs *services* (business logic) vs *repository* (DB).  
  - Errors must be handled (don’t ignore `err`).  
  - Use `context.Context` in handlers and pass it to deeper layers.  
  - JSON tags: use `json:"field_name"` for serialization.  
  - Use `golang-migrate` or equivalent for migrations.  
  - Keep logic modular so adding new features (e.g. OAuth, widget APIs) is straightforward.

- **Frontend (Next.js / React)**  
  - Use functional components with hooks.  
  - TypeScript types/interfaces in `src/types/`.  
  - Use `fetch()` or `axios` via a shared `lib/api.ts` to call backend APIs.  
  - Keep components small and focused (e.g. `ServiceCard`, `ThemeSelector`).  
  - Use CSS variables and Tailwind for theming (dark/light mode, accent colors).  
  - Pages in `pages/` follow Next.js routing conventions.  
  - For API calls, always include error-handling, loading states, and token refresh logic.  

- **Naming / Conventions**  
  - Use **camelCase** for JavaScript/TypeScript variables and functions.  
  - Use **PascalCase** for React components and Go exported struct types.  
  - For Go private members, use lowercase names.  
  - Use meaningful names: `userService`, `healthService`, `serviceHandler`, etc.  
  - Commit messages: “feat: …”, “fix: …”, etc. (keep conventional style)  

---

## 🧩 Developer Workflow & Commands

- **Backend**  
  - `make dev` or `go run ./cmd/server` → run local server  
  - `make migrate` → run DB migrations  
  - `make test` → run backend tests  
  - `go fmt` / `gofmt` → formatting  
  - `go vet` → lint / checks  

- **Frontend**  
  - `npm run dev` → start Next.js dev server  
  - `npm run build` → production build  
  - `npm run start` → serve built frontend  
  - `npm run lint` → lint checks  
  - `npm run test` → frontend tests  

- **Docker / Deployment**  
  - `docker-compose up -d --build` → start full stack  
  - `.env` file controls env vars (API_URL, DB_URL, JWT_SECRET, etc.)  
  - `.env.example` should list all required variables without secrets  

---

## ✅ How Claude Should Use This Context

- When asked to generate or modify code, first check **section “Project Overview”**, **directory structure**, and **style guidelines** to align output.  
- When adding new endpoints or features, ask for confirmation (“Does this align with current module layout?”).  
- Use this file to recall naming conventions or directory paths.  
- If code changes become large or risky, propose a plan first (don’t jump into writing code).  
- Respect modular boundaries (don’t mix UI logic into backend, etc.).  

---

## 🛠 Known Edge Cases & Warnings

- If branching for new features (e.g. widgets), maintain same patterns (handlers → services → repository).  
- When modifying `docker-compose.yml`, respect service names (`backend`, `frontend`, `db`) unless explicitly asked to rename.  
- If adding new dependencies (Go modules or npm packages), update module files (`go.mod`, `package.json`) and ensure minimal version numbers.  