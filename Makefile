ifneq ("$(wildcard .env)","")
	include .env
endif

MOCKERY_VERSION := v3.5.5
MOCKERY := bin/mockery
MOCKERY_STAMP := bin/mockery-$(MOCKERY_VERSION)
MINIO_CONTAINER ?= minio
POSTGRES_CONTAINER ?= fyp-postgres
CATALOG_DATA_DUMP ?= data/catalog-data.sql
CATALOG_DUMP_TABLES := meal_categories prebuilt_meals prebuilt_meal_images
MIGRATION_FILES := $(sort $(wildcard db/migrations/*.up.sql))

.PHONY: dev-env-start dev dev-client dev-start generate dev-migrate migrate-up catalog-dump catalog-seed catalog-install catalog-install-ml catalog-lint catalog-test

dev-env-start:
	@if podman container exists $(POSTGRES_CONTAINER); then \
		podman start $(POSTGRES_CONTAINER); \
	else \
		podman run -d \
			--name $(POSTGRES_CONTAINER) \
			-e POSTGRES_USER=${DATABASE_USER} \
			-e POSTGRES_PASSWORD=${DATABASE_PASSWORD} \
			-e POSTGRES_DB=${DATABASE_NAME} \
			-p ${DATABASE_PORT}:5432 \
			postgres; \
	fi
	@if podman container exists $(MINIO_CONTAINER); then \
		podman start $(MINIO_CONTAINER); \
	else \
		podman run -d \
			--name $(MINIO_CONTAINER) \
			-e MINIO_ROOT_USER=${MINIO_ACCESS_KEY} \
			-e MINIO_ROOT_PASSWORD=${MINIO_SECRET_KEY} \
			-p 9000:9000 \
			-p 9001:9001 \
			-v minio_data:/data \
			minio/minio server /data --console-address ":9001"; \
	fi
	@until curl -fsS http://${MINIO_ENDPOINT}/minio/health/ready >/dev/null 2>&1; do \
		sleep 1; \
	done
	@podman run --rm --network host --entrypoint /bin/sh minio/mc -c \
		'mc alias set local http://${MINIO_ENDPOINT} ${MINIO_ACCESS_KEY} ${MINIO_SECRET_KEY} >/dev/null && mc mb --ignore-existing local/${MINIO_BUCKET}'

dev:
	go run main.go

dev-client:
	pnpm -C client dev

dev-start:
	podman run -d \
  	--name fyp-postgres \
  	-e POSTGRES_USER=${DATABASE_USER} \
  	-e POSTGRES_PASSWORD=${DATABASE_PASSWORD} \
  	-e POSTGRES_DB=${DATABASE_NAME} \
  	-p ${DATABASE_PORT}:5432 \
  	postgres

generate: $(MOCKERY_STAMP)
	$(MOCKERY)
	go generate ./...

dev-migrate: dev-env-start migrate-up catalog-seed

# Apply every db/migrations/*.up.sql
migrate-up:
	@for migration in $(MIGRATION_FILES); do \
		echo "Applying $$migration"; \
		podman exec -i -e PGPASSWORD="${DATABASE_PASSWORD}" $(POSTGRES_CONTAINER) psql \
			-v ON_ERROR_STOP=1 \
			-U "${DATABASE_USER}" \
			-d "${DATABASE_NAME}" \
			< "$$migration"; \
	done

# Snapshot the catalog tables from current local Postgres database into a SQL file.
# Write to data/catalog-data.sql 
# export catalog tables from DB into data/catalog-data.sql
catalog-dump:
	@mkdir -p $(dir $(CATALOG_DATA_DUMP))
	podman exec -e PGPASSWORD="${DATABASE_PASSWORD}" $(POSTGRES_CONTAINER) pg_dump \
		-U "${DATABASE_USER}" \
		-d "${DATABASE_NAME}" \
		--data-only \
		--column-inserts \
		--no-owner \
		--no-privileges \
		$(foreach table,$(CATALOG_DUMP_TABLES),--table=$(table)) \
			> "$(CATALOG_DATA_DUMP)"
	@echo "Wrote catalog data dump to $(CATALOG_DATA_DUMP)"

# Loads data/catalog-data.sql into DB
catalog-seed:
	@test -f "$(CATALOG_DATA_DUMP)" || (echo "Missing $(CATALOG_DATA_DUMP). Run make catalog-dump first." && exit 1)
	podman exec -e PGPASSWORD="${DATABASE_PASSWORD}" $(POSTGRES_CONTAINER) psql \
		-v ON_ERROR_STOP=1 \
		-U "${DATABASE_USER}" \
		-d "${DATABASE_NAME}" \
		-c "DELETE FROM prebuilt_meal_images; DELETE FROM prebuilt_meals; DELETE FROM meal_categories;"
	podman exec -i -e PGPASSWORD="${DATABASE_PASSWORD}" $(POSTGRES_CONTAINER) psql \
		-v ON_ERROR_STOP=1 \
		-U "${DATABASE_USER}" \
		-d "${DATABASE_NAME}" \
		< "$(CATALOG_DATA_DUMP)"
	@echo "Seeded catalog data from $(CATALOG_DATA_DUMP)"

catalog-install:
	python3 -m venv .venv
	.venv/bin/pip install -e '.[dev,image]'

catalog-install-ml:
	.venv/bin/pip install -e '.[ml]'

catalog-lint:
	.venv/bin/ruff check scripts/meal_catalog
	.venv/bin/ruff format --check scripts/meal_catalog

catalog-test:
	.venv/bin/pytest scripts/meal_catalog/tests -q

$(MOCKERY_STAMP):
	GOBIN=$(CURDIR)/bin go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)
	touch $(MOCKERY_STAMP)
