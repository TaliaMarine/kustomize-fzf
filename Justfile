default: build

build:
	go build -o bin/kustomize-fzf

install:
	go install kustomize-fzf

clean:
	rm -rf bin dist
	@echo "Cleaned build artifacts"

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

