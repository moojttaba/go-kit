.PHONY: test vet lint fmt tidy hooks
hooks:
	git config core.hooksPath .githooks
	@echo "git hooks enabled → .githooks/pre-push (vet·test·lint·gitleaks·govulncheck on push to main)"
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
