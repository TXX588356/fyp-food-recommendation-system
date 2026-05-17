podman-postgres:
	podman start fyp-postgres

dev:
	go run .

dev-client:
	pnpm -C client dev

