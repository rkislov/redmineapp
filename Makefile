.PHONY: run test build package
run:
	go run .
test:
	go test -race ./internal/redmine ./internal/settings
	go test -tags ci ./internal/ui
build:
	mkdir -p dist
	go build -trimpath -o dist/redmineapp .
package:
	./packaging/package.sh
