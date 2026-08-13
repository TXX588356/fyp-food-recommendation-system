# Food Recommendation System

Final Year Project web application for personalised meal recommendations, meal logging, custom meal management, and nutrition/budget tracking.

The system is built as a full-stack application with a Go backend, React TypeScript frontend, PostgreSQL database, MinIO object storage, and AI-powered recommendation support through Google Gemini.

## Technology Stack

| Layer | Technology |
| --- | --- |
| Backend | Go 1.26.1, Echo, Cobra, GORM |
| Frontend | React 19, TypeScript, Vite |
| Database | PostgreSQL |
| Object storage | MinIO |
| AI integration | Google Gemini API |
| Restaurant search | SerpAPI, optional |
| Local services | Podman, Make |

## Prerequisites

Install the following tools before running the project:

- Go 1.26.1 or compatible
- Node.js
- pnpm
- Podman
- Make

## Environment Setup

The application reads configuration from `config/config.yaml`, `.env`, and operating system environment variables.

Create a local `.env` file in the project root. Refer to `.env.example`.

## Project Scaffolding

The backend is a Go module:

```bash
go mod init fyp/food-rs
```

The backend entry point is `main.go`, and commands are registered through Cobra under `app/cmd`.

The frontend was scaffolded with Vite, React, and TypeScript:

```bash
pnpm create vite client --template react-ts
```

## Directory Structure

```text
food-recommendation-system/
├── app/                    # Backend CLI commands and server startup
├── client/                 # React TypeScript frontend
│   ├── public/             # Static frontend assets
│   ├── src/                # Frontend pages, routing, auth, theme, and UI logic
│   └── package.json        # Frontend scripts and dependencies
├── config/                 # Application configuration
├── data/                   # Catalog seed data
├── db/
│   └── migrations/         # PostgreSQL migration files
├── internal/               # Backend implementation
│   ├── config/             # Configuration loading
│   ├── database/           # PostgreSQL connection setup
│   ├── endpoint/           # HTTP API handlers and routes
│   ├── interfaces/         # Service and repository interfaces
│   ├── repository/         # Database repositories
│   ├── service/            # Business logic services
│   └── storage/            # Image/object storage logic
├── types/                  # Backend model definitions
├── Dockerfile              # Multi-stage production build
├── Makefile                # Development commands
├── docker-compose.yml      # MinIO service definition
├── go.mod                  # Backend dependencies
└── main.go                 # Backend entry point
```

## Dependency Installation

Install backend dependencies:

```bash
go mod download
```

Install frontend dependencies:

```bash
pnpm -C client install
```

## Running the Project Locally

Start the local development services:

```bash
make dev-env-start
```

This starts PostgreSQL and MinIO containers through Podman. PostgreSQL is exposed on port `5433`, MinIO API on port `9000`, and the MinIO console on port `9001`.

Apply database migrations and seed the meal catalog:

```bash
make dev-migrate
```

Start the backend server:

```bash
make dev
```

The backend runs at:

```text
http://localhost:8080
```

Start the frontend development server:

```bash
pnpm -C client dev
```

The frontend runs at:

```text
http://localhost:5173
```

## Database Commands

Apply all migration files:

```bash
make migrate-up
```

Seed catalog data from `data/catalog-data.sql`:

```bash
make catalog-seed
```

Export the current catalog tables into `data/catalog-data.sql`:

```bash
make catalog-dump
```

## Available Backend Commands

Run the HTTP server:

```bash
go run main.go server
```

Run database migrations:

```bash
go run main.go migrate
```

Seed catalog data:

```bash
go run main.go seed-catalog
```

Run migrations and seed data:

```bash
go run main.go migrate-and-seed
```

## Frontend Scripts

Run the Vite development server:

```bash
pnpm -C client dev
```

Build the frontend:

```bash
pnpm -C client build
```

Run linting:

```bash
pnpm -C client lint
```

Preview the production frontend build:

```bash
pnpm -C client preview
```

## Testing

Run backend tests:

```bash
make test
```

The test command runs Go tests across backend packages and writes coverage output to:

```text
out/coverage.out
```

## Production Build

The Dockerfile uses a multi-stage build:

1. Builds the React frontend using Node and pnpm.
2. Builds the Go backend binary.
3. Copies the backend binary, compiled frontend assets, and configuration files into a minimal runtime image.

The backend can serve the compiled React frontend from the `static` directory, allowing the API and frontend to run from the same server in production-style deployments.

## Main Features

- User registration and authentication
- Onboarding and preference collection
- Personalised meal recommendation
- Meal search and meal detail explanation
- Custom meal creation
- Meal logging
- Meal log reporting
- Prebuilt meal catalog
- Image storage through MinIO
- Optional restaurant suggestion through SerpAPI
