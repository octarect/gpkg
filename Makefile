.PHONY: build
build:
	@mkdir bin/
	@go build -o ./bin/gpkg ./cmd/gpkg/main.go

.PHONY: clean
clean:
	rm -rf bin/
