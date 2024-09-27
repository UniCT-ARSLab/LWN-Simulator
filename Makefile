ifeq ($(OS),Windows_NT)
    copy_config = powershell Copy-Item config.json bin -Force
    make_bin = powershell New-Item -ItemType Directory -Force  -Path bin  | powershell Out-Null
    output_file = bin\lwnsimulator.exe
else
    copy_config = cp -f ./config.json ./bin/config.json
    make_bin = mkdir -p bin
    output_file = bin/lwnsimulator
endif

install-dep:
	@echo Installing Deps
	@go install github.com/rakyll/statik@latest
	@go mod download -x

build:
	@echo Starting the build the LWN Simulator
	@echo Baking the User Interface
	@cd webserver && statik -f -src=public
	@$(make_bin)
	@$(copy_config)
	@echo Building the source
	@go build -o $(output_file) cmd/main.go
	@echo Build Complete

build-platform:
	@echo Starting the build the LWN Simulator $(SUFFIX)
	@echo Baking the User Interface
	@cd webserver && statik -src=public
	@$(make_bin)
	@$(copy_config)
	@echo Building the source
	@go build -o bin//lwnsimulator$(SUFFIX) cmd/main.go
	@echo "Build Complete"

linux-build-x64:
	@make build-platform GOOS=linux GOARCH=amd64 SUFFIX="_x64"

linux-build-x86:
	@make build-platform GOOS=linux GOARCH=386 SUFFIX="_x86"

linux-build-all:
	@make build-x64
	@make build-x86
run:
	@echo Baking the User Interface
	@cd webserver && statik -f -src=public
	@echo Running
	@go run cmd/main.go
run-release:
	@$(output_file)