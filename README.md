Chirpy

Overview
- Chirpy is a small Go web service that exposes a REST-style API for posting and managing short messages ("chirps") and users. It also serves static assets for a simple web UI under the /app path.
- The server uses PostgreSQL with sqlc-generated data-access code, JWTs for auth, Argon2id for password hashing, and standard net/http for routing.

Tech stack
- Language: Go (module name: chirpy; go version as specified in go.mod)
- Runtime: standard net/http server
- Database: PostgreSQL
- SQL to Go codegen: sqlc (configured via sqlc.yaml)
- Auth: JWT (github.com/golang-jwt/jwt/v5), Argon2id (github.com/alexedwards/argon2id)
- Env loader: github.com/joho/godotenv

Requirements
- Go toolchain (version per go.mod; currently set to 1.25)
- PostgreSQL instance and a connection URL
- sqlc (only needed if regenerating the internal/database code)

Project structure
- main.go — server entry point; wires configuration, routes, and HTTP server
- internal/
  - api/ — HTTP handlers, middleware, request/response helpers
  - auth/ — password hashing, JWT utilities, API key helpers (+ tests)
  - database/ — sqlc-generated code (Queries, models, and compiled SQL)
- sql/
  - schema/ — database schema DDL (ordered migrations 001_*.sql, 002_*.sql, ...)
  - queries/ — application SQL used by sqlc to generate code
- sqlc.yaml — sqlc configuration
- index.html — static file served under /app
- assets/ — static assets (e.g., logo)

Entry point
- Command: go run ./main.go
- The HTTP server listens on :8080 by default (hardcoded in main.go).
- Static files are served from the repository root under the /app path (e.g., http://localhost:8080/app/index.html).

Environment variables
- DB_URL: PostgreSQL connection string (e.g., postgres://user:pass@localhost:5432/chirpy?sslmode=disable)
- PLATFORM: deployment/platform identifier used by the app logic (string; optional, semantics app-specific)
- BEARER_TOKEN_SECRET: HMAC secret used to sign JWTs
- POLKA_KEY: API key used to authenticate webhook requests to /api/polka/webhooks
- .env support: main.go loads variables from a local .env file if present (via godotenv). Example .env snippet:
  - DB_URL=postgres://user:pass@localhost:5432/chirpy?sslmode=disable
  - PLATFORM=dev
  - BEARER_TOKEN_SECRET=dev-secret-change-me
  - POLKA_KEY=dev-polka-key

Database setup
1. Create a PostgreSQL database and user.
2. Apply the schema files in sql/schema in order:
   - 001_users.sql
   - 002_chirps.sql
   - 003_user_passwords.sql
   - 004_refresh_tokens.sql
   - 005_chirpy_red.sql
   Example using psql:
   - psql "$DB_URL" -f sql/schema/001_users.sql
   - psql "$DB_URL" -f sql/schema/002_chirps.sql
   - psql "$DB_URL" -f sql/schema/003_user_passwords.sql
   - psql "$DB_URL" -f sql/schema/004_refresh_tokens.sql
   - psql "$DB_URL" -f sql/schema/005_chirpy_red.sql
3. Optional: Regenerate sqlc code (only if you modify SQL)
   - Install sqlc: https://docs.sqlc.dev/
   - sqlc generate

Running locally
1. Export environment variables or create a .env file at the repo root.
2. Start the server:
   - go run ./main.go
3. Open the app:
   - Static UI: http://localhost:8080/app/
   - Health check: GET http://localhost:8080/admin/healthz

Common API endpoints (non-exhaustive)
- GET /admin/healthz → 200 OK if server is healthy
- GET /admin/metrics → returns simple fileserver hit metrics
- POST /admin/reset → resets database state (use with care; typically for development/tests)
- POST /api/users → register user
- POST /api/login → login and receive tokens
- PUT /api/users → update current user (auth required)
- POST /api/refresh → exchange refresh token for new access token
- POST /api/revoke → revoke refresh token
- POST /api/chirps → create chirp (auth required)
- GET /api/chirps → list chirps
- GET /api/chirps/{id} → get chirp by ID
- DELETE /api/chirps/{id} → delete chirp by ID (authorization enforced)
- POST /api/polka/webhooks → webhook endpoint secured by POLKA_KEY

Scripts and useful commands
- Run server: go run ./main.go
- Build binary: go build -o bin/chirpy ./main.go
- Run tests: go test ./...
- Lint/format (optional): go fmt ./...
- Regenerate sqlc code: sqlc generate

Testing
- Unit tests are present under internal/auth. Run all tests with:
  - go test ./...
- You can filter to a specific package:
  - go test ./internal/auth -v

Notes and assumptions
- The server port is fixed to :8080 in main.go.
- The application expects PostgreSQL and a valid DB_URL; there is no embedded DB or auto-migration code.
- Static assets are served from the repository root under the /app prefix via http.FileServer.

License
- TODO: Add license information (e.g., MIT, Apache-2.0). No LICENSE file was found in this repository.

Maintenance
- Update go.mod/go.sum as needed with: go get -u and go mod tidy
- If you change SQL in sql/queries or sql/schema, re-run sqlc generate to refresh internal/database code.
