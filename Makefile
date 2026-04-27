.PHONY: hooks-install test vet check

# Activate the tracked git hooks (.githooks/) for this clone.
# Run once after cloning. Idempotent.
hooks-install:
	git config --local core.hooksPath .githooks
	@echo "Pre-commit hook activated. To bypass: git commit --no-verify"

test:
	go test ./...

vet:
	go vet ./...

check: vet test
