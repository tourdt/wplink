.PHONY: check check-backend check-admin check-wxapp check-postgres

check: check-backend check-admin check-wxapp

check-backend:
	cd backend && node --test scripts/validate_migrations.test.mjs scripts/api_contract.test.mjs
	cd backend && go test ./...
	cd backend && go vet ./...

check-postgres:
	@test -n "$(WPLINK_TEST_POSTGRES_DSN)" || (echo "WPLINK_TEST_POSTGRES_DSN 必须指向可丢弃测试库"; exit 1)
	cd backend && go test ./app/internal/task -run '^TestCoordinatorIntegration' -count=1 -v
	cd backend && go test ./app/internal/logic/auth -run '^TestSQLSMSSendLimiterIntegration' -count=1 -v

check-admin:
	npm --prefix admin-web run check

check-wxapp:
	npm --prefix wxapp run check
