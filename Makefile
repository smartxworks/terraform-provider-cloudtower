version = 0.1.0

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

vendor: ## Run go mod vendor against code.
	go mod vendor

tidy:  ## Run tidy against code.
	go mod tidy

build: fmt  ## Build binary.
	go build -o ./bin/terraform-provider-cloudtower main.go

install: build ## Run a controller from your host.
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/smartx/cloudtower/0.1.0/darwin_arm64
	mv ./bin/terraform-provider-cloudtower ~/.terraform.d/plugins/registry.terraform.io/smartx/cloudtower/0.1.0/darwin_arm64/
