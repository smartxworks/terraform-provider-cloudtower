version = $(file < VERSION)
GOOS =
GOARCH = 
exeExtension =
ifeq ($(OS),Windows_NT)
	GOOS = windows
	ifeq ($(PROCESSOR_ARCHITEW6432),AMD64)
		GOARCH = amd64
	else ifeq ($(PROCESSOR_ARCHITECTURE),IA64)
		GOARCH = amd64
	else ifeq ($(PROCESSOR_ARCHITECTURE),x86)
		GOARCH = 386
	endif
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		GOOS = linux
	endif
	ifeq ($(UNAME_S),Darwin)
		GOOS = darwin
	endif
	UNAME_P := $(shell uname -p)
	ifeq ($(UNAME_P),) 
		UNAME_P = $(shell uname -m)
	else ifeq ($(UNAME_P),unknown)
		UNAME_P = $(shell uname -m)
	endif
	ifeq ($(UNAME_P),x86_64)
		GOARCH = amd64
	endif
	ifneq ($(filter %86,$(UNAME_P)),)
		GOARCH = 386
	endif
	ifneq ($(filter arm%,$(UNAME_P)),)
		ifeq ($(GOARCH),386)
			GOARCH = arm
		else
			GOARCH = arm64
		endif
	endif
endif

ifndef GOOS
	$(error GOOS is not set)
endif
ifndef GOARCH
	$(error GOARCH is not set)
endif

BuildTarget = 
LocalRegistry = 
MkdirCmd = 
MvCmd = 
CleanRegistryCmd = 
ifeq ($(GOOS),windows)
	BuildTarget = .\bin\terraform-provider-cloudtower.exe
	LocalRegistry = $(APPDATA)\terraform.d\plugins\registry.terraform.io\smartx\cloudtower\$(version)\$(GOOS)_$(GOARCH)
	MkdirCmd = mkdir $(LocalRegistry)
	MvCmd = move $(BuildTarget) $(LocalRegistry)
	CleanRegistryCmd = if exist "$(LocalRegistry)" rmdir /S /Q $(LocalRegistry)
else
	BuildTarget = ./bin/terraform-provider-cloudtower
	LocalRegistry = ~/.terraform.d/plugins/registry.terraform.io/smartx/cloudtower/$(version)/$(GOOS)_$(GOARCH)/
	MkdirCmd = mkdir -p $(LocalRegistry)
	MvCmd = mv $(BuildTarget) $(LocalRegistry)
	CleanRegistryCmd = if [ -d "$(LocalRegistry)" ]; then rm -rf $(LocalRegistry); fi
endif

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

vendor: ## Run go mod vendor against code.
	go mod vendor

tidy:  ## Run tidy against code.
	go mod tidy

clean: ## Remove all build artifacts.
	$(CleanRegistryCmd)
	$(MkdirCmd)

build: fmt clean ## Build binary.
	go build -o $(BuildTarget) main.go

install: build ## Run a controller from your host.
	$(MvCmd)
