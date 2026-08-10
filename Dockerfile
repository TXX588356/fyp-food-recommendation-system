# frontend build
FROM node:22-alpine AS frontend
WORKDIR /app/client

COPY client/package.json client/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

COPY client ./
ARG VITE_API_BASE_URL=""
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN pnpm build

# backend build
FROM golang:1.26.1-alpine AS backend
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o fyp-smart-meal ./main.go

# runtime image
FROM alpine:latest
WORKDIR /app

COPY --from=backend /app/fyp-smart-meal ./fyp-smart-meal
COPY --from=frontend /app/client/dist ./static
COPY config ./config

EXPOSE 8080

CMD ["./fyp-smart-meal", "server"]
