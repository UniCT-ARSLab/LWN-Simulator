install-dep:
	@go install github.com/rakyll/statik
	@go get -u -d ./...

build:
	@echo -e "\e[96mBuilding the \e[95mLWN Simulator\e[39m"
	@echo -e "\e[96mBuilding the \e[94mUser Interface\e[39m"
	@cd webserver && statik -src=public -f 1>/dev/null
	@mkdir -p bin
	@export GHW_DISABLE_WARNINGS=1
	@cp -f config.json bin/config.json
	@echo -e "\e[96mBuilding \e[93mthe source\e[39m"
	@go build -o bin/lwnsimulator cmd/main.go
	@echo -e "\e[92mBuild Complete\e[39m"

build-platform:
	@echo -e "\e[96mBuilding the \e[95mLWN Simulator (${SUFFIX})\e[39m"
	@echo -e "\e[96mBuilding the \e[94mUser Interface\e[39m"
	@cd webserver && statik -src=public -f 1>/dev/null
	@mkdir -p bin
	@export GHW_DISABLE_WARNINGS=1
	@cp -f config.json bin/config.json
	@echo -e "\e[96mBuilding \e[93mthe source\e[39m"
	@go build -o bin/lwnsimulator${SUFFIX} cmd/main.go
	@echo -e "\e[92mBuild Complete\e[39m"

build-x64:
	@make build-platform GOOS=linux GOARCH=amd64 SUFFIX="_x64"

build-x86:
	@make build-platform GOOS=linux GOARCH=386 SUFFIX="_x86"

build-all:
	@make build-x64
	@make build-x86
run:
	@go run cmd/main.go

run-release:
	@bin/lwnsimulator