.PHONY: check check-backend check-admin check-wxapp check-postgres check-dependencies generate-api check-api-generated

check: check-backend check-admin check-wxapp

check-backend:
	cd backend && node --test scripts/validate_migrations.test.mjs scripts/api_contract.test.mjs scripts/api_route_inventory.test.mjs scripts/api_codegen.test.mjs scripts/product_doc_governance.test.mjs
	$(MAKE) check-api-generated
	cd backend && go test ./...
	cd backend && go vet ./...

generate-api:
	cd backend && node scripts/api_codegen.mjs --write

check-api-generated:
	cd backend && node scripts/api_codegen.mjs --check

check-postgres:
	@test -n "$(WPLINK_TEST_POSTGRES_DSN)" || (echo "WPLINK_TEST_POSTGRES_DSN 必须指向可丢弃测试库"; exit 1)
	cd backend && go test ./app/internal/task -run '^TestCoordinatorIntegration' -count=1 -v
	cd backend && go test ./app/internal/logic/auth -run '^TestSQLSMSSendLimiterIntegration' -count=1 -v

check-dependencies:
	npm audit --prefix admin-web --audit-level=high
	npm audit --prefix wxapp --audit-level=high
	cd backend && go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

check-admin:
	npm --prefix admin-web run check

check-wxapp:
	npm --prefix wxapp run check
