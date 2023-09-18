bin/gpkg:
	mkdir bin/
	go build -o ./bin/gpkg ./cmd/gpkg/main.go

.PHONY: build
build: bin/gpkg

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -rf bin/
