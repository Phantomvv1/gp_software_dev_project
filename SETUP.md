# GP Software Dev Project – Setup Manual

A step-by-step guide for running the project both **manually** (local Go + external DB)
and via **Docker Compose**.

---

## Prerequisites

| Tool | Minimum version | Download |
|------|-----------------|----------|
| Go   | 1.23            | https://go.dev/dl/ |
| Git  | any             | https://git-scm.com |
| PostgreSQL | 14+      | https://www.postgresql.org/download/ *(manual run only)* |
| Docker + Docker Compose | 24 / v2 | https://docs.docker.com/get-docker/ *(Docker run only)* |

---

## 1 — Clone the repository

```bash
git clone https://github.com/Phantomvv1/gp_software_dev_project.git
cd gp_software_dev_project
```

---

## 2 — Environment variables

Both run methods require the same three environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `JWT_KEY` | **Yes** | Secret string used to sign/verify JWT tokens. Use a long random value in production. |
| `API_KEY` | **Yes** | Pre-shared key clients must send in the `X-API-KEY` HTTP header. |
| `DATABASE_URL` | **Yes** | Full PostgreSQL DSN, e.g. `postgres://user:pass@host:5432/dbname?sslmode=disable` |

> **Tip:** Create a `.env` file in the project root (it is already in `.gitignore`):
>
> ```env
> JWT_KEY=supersecretjwtkey
> API_KEY=mypresharedapikey
> DATABASE_URL=postgres://gpuser:gppassword@localhost:5432/gpdb?sslmode=disable
> POSTGRES_USER=gpuser
> POSTGRES_PASSWORD=gppassword
> POSTGRES_DB=gpdb
> ```

---

## 3 — Run with Docker Compose (recommended)

Docker Compose starts **both** the PostgreSQL database and the Go API in isolated
containers and wires them together automatically.

### 3.1 Build and start

```bash
docker compose up --build
```

On subsequent starts (no code changes) you can omit `--build`:

```bash
docker compose up
```

### 3.2 Run in the background (detached)

```bash
docker compose up -d --build
```

Check logs:

```bash
docker compose logs -f api   # API logs
docker compose logs -f db    # Postgres logs
```

### 3.3 Stop everything

```bash
docker compose down          # stops containers, keeps DB volume
docker compose down -v       # stops containers AND removes DB volume (deletes data)
```

### 3.4 Apply database migrations

The project uses **Goose** for migrations. After the containers are running:

```bash
# Install goose (once)
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations against the containerised DB
goose -dir ./migrations postgres \
  "postgres://gpuser:gppassword@localhost:5432/gpdb?sslmode=disable" up
```

---

## 4 — Run manually (local Go toolchain)

### 4.1 Install Go

1. Download the installer for your OS from https://go.dev/dl/
2. Follow the official install instructions.
3. Verify:

```bash
go version   # should print go1.23 or newer
```

### 4.2 Start PostgreSQL

Make sure a PostgreSQL server is running and you have created a database:

```sql
CREATE USER gpuser WITH PASSWORD 'gppassword';
CREATE DATABASE gpdb OWNER gpuser;
```

### 4.3 Apply database migrations

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest

goose -dir ./migrations postgres \
  "postgres://gpuser:gppassword@localhost:5432/gpdb?sslmode=disable" up
```

### 4.4 Set environment variables and run

**Linux / macOS:**

```bash
export JWT_KEY="supersecretjwtkey"
export API_KEY="mypresharedapikey"
export DATABASE_URL="postgres://gpuser:gppassword@localhost:5432/gpdb?sslmode=disable"

go run ./cmd/gp_software_dev_project/main.go
```

**Windows (PowerShell):**

```powershell
$env:JWT_KEY="supersecretjwtkey"
$env:API_KEY="mypresharedapikey"
$env:DATABASE_URL="postgres://gpuser:gppassword@localhost:5432/gpdb?sslmode=disable"

go run .\cmd\gp_software_dev_project\main.go
```

The API server starts on **port 42069**.

---

## 5 — Verify the server is running

```bash
curl -s -o /dev/null -w "%{http_code}" \
  -H "X-API-KEY: mypresharedapikey" \
  http://localhost:42069
# Expected: 200
```

---

## 6 — Quick API reference

Some requests require the `X-API-KEY` header.
Protected endpoints additionally require `Authorization: Bearer <token>`.

### Register a doctor
```bash
curl -X POST http://localhost:42069/doctors \
  -H "X-API-KEY: mypresharedapikey" \
  -H "Content-Type: application/json" \
  -d '{"name":"Dr. Smith","email":"smith@clinic.com","password":"secret","address":"1 Main St"}'
```

### Login
```bash
curl -X POST http://localhost:42069/login \
  -H "X-API-KEY: mypresharedapikey" \
  -H "Content-Type: application/json" \
  -d '{"email":"smith@clinic.com","password":"secret"}'
# Returns: {"result":{"token":"<jwt>"}}
```

### Use a protected endpoint
```bash
curl http://localhost:42069/me \
  -H "X-API-KEY: mypresharedapikey" \
  -H "Authorization: Bearer <token>"
```

---

## 7 — Directory structure (reference)

```
gp_software_dev_project/
├── cmd/gp_software_dev_project/main.go   ← entry point
├── routes/routes.go                      ← route registration
├── internal/
│   ├── auth/          ← JWT, SHA-512, Login handler
│   ├── middleware/    ← APIKeyAuth, AuthMiddleware, RequireRole
│   ├── doctors/       ← Doctor CRUD handlers
│   ├── patients/      ← Patient CRUD handlers
│   ├── visits/        ← Visit handlers
│   └── working_hours/ ← Working hours handlers
├── migrations/        ← Goose SQL migrations
├── Dockerfile
├── docker-compose.yml
└── .env               ← (create this yourself; not committed to git)
```

---

## 8 — Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `401 Unauthorized` on every request | Missing or wrong `X-API-KEY` | Check the `API_KEY` env var matches the header value |
| `403 Forbidden` on protected routes | Expired or malformed JWT | Re-login to get a fresh token |
| `connection refused` on port 5432 | DB not running | Start PostgreSQL or run `docker compose up db` |
| Binary panics on start | Missing env var | Ensure `JWT_KEY`, `API_KEY`, and `DATABASE_URL` are all set |
| Goose migration fails | DB doesn't exist yet | Create the database first (section 4.2) |
