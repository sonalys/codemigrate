.SILENT:

PKGS_RECURSIVE = $$(go work edit -json | jq -r '.Use[].DiskPath + "/..."')
PKGS = $$(go work edit -json | jq -r '.Use[].DiskPath')

LINTER = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.0.2
TESTER = go run gotest.tools/gotestsum@v1.12.1

.PHONY: lint
lint:
	$(LINTER) run $(PKGS_RECURSIVE) --timeout 5m

.PHONY: test
test:
	$(TESTER) -- $(PKGS_RECURSIVE)

.PHONY: test-coverage
test-coverage:
	$(TESTER) -- -cover $(PKGS_RECURSIVE)

tidy:
	for pkg in $(PKGS); do \
		(cd "$$pkg" && go mod tidy); \
	done