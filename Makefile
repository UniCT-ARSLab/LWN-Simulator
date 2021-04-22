install-dep:
	@go get -u github.com/rakyll/statik
	@mkdir -p "webserver/public"
	@if [[ ! -e webserver/public/index.html ]]; then\
    	echo "LWNSimulator - Need to do \"make build\" or similar build (for other platforms) before using GUI!" > webserver/www/index.html;\
	fi
	@cd webserver && statik -src=public -f 1>/dev/null
	@go get -u -v all

build:
	@cp -f config.json bin/config.json
	@go build -o bin/lwnsimulator cmd/main.go

run:
	@go run cmd/main.go

run-release:
	@bin/lwnsimulator