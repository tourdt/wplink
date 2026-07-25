.PHONY: check check-backend check-admin check-wxapp

check: check-backend check-admin check-wxapp

check-backend:
	cd backend && node --test scripts/validate_migrations.test.mjs scripts/api_contract.test.mjs
	cd backend && go test ./...
	cd backend && go vet ./...

check-admin:
	npm --prefix admin-web run check

check-wxapp:
	npm --prefix wxapp run check
