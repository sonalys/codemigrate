.SILENT:

PKGS_RECURSIVE = $$(go work edit -json | jq -r '.Use[].DiskPath + "/..."')
PKGS = $$(go work edit -json | jq -r '.Use[].DiskPath')
LINTER = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.0.2

.PHONY: lint
lint:
	$(LINTER) run $(PKGS_RECURSIVE) --timeout 5m

.PHONY: test
test:
	go test $(PKGS_RECURSIVE)

.PHONY: test-coverage
test-coverage:
	go test -cover $(PKGS_RECURSIVE)

tidy:
	for pkg in $(PKGS); do \
		(cd "$$pkg" && go mod tidy); \
	done