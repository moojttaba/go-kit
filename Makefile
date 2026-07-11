.PHONY: test vet lint fmt tidy hooks
hooks:
	@printf '#!/usr/bin/env bash\nexec "$$(git rev-parse --show-toplevel)/.githooks/pre-push" "$$@"\n' > .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "git pre-push gate installed → .git/hooks/pre-push chains to .githooks/pre-push (vet·test·lint·gitleaks·govulncheck on push to main)"
test:
	go test ./...
vet:
	go vet ./...
lint:
	golangci-lint run --timeout=5m
fmt:
	gofumpt -w .
tidy:
	go mod tidy
