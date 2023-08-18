.PHONY: build
build:
	@mkdir bin/
	@go build -o ./bin/gpkg ./main.go

.PHONY: clean
clean:
	rm -rf bin/
