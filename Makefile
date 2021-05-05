install-dep:
	@go get -d -u github.com/rakyll/statik
	@go get -u -d ./...

build:
	@echo -e "\e[96mBuilding the \e[95mLWN Simulator\e[39m"
	@echo -e "\e[96mBuilding the \e[94mUser Interface\e[39m"
	@cd webserver && statik -src=public -f 1>/dev/null
	@mkdir -p bin
	@mkdir -p bin/config
	@export GHW_DISABLE_WARNINGS=1
	@cp -f config.json bin/config.json
	@echo -e "\e[96mBuilding \e[93mthe source\e[39m"
	@go build -o bin/lwnsimulator cmd/main.go
	@echo -e "\e[92mBuild Complete\e[39m"

run:
	@go run cmd/main.go

run-release:
	@bin/lwnsimulator