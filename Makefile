.DEFAULT_GOAL := help

.PHONY: update-nix-hash
update-nix-hash:
	@go run bin/update-nix-hash.go

.PHONY: clean
clean:
	rm -rf result result-*

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  update-nix-hash  - Automatically update the vendorHash in flake.nix"
	@echo ""
	@echo "Updating vendorHash:"
	@echo "  After changing Go dependencies (go get, go mod tidy), run:"
	@echo "    make update-nix-hash"
	@echo "  or directly:"
	@echo "    go run scripts/update-nix-hash.go"
