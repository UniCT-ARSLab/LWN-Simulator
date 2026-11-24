module lwn-simulator-backend

require (
	github.com/Giuseppe-Compagnone/lwn-contracts v1.0.0
	github.com/Giuseppe-Compagnone/lwn-engine v1.0.0
	github.com/gin-contrib/cors v1.7.7
	github.com/gin-gonic/gin v1.12.0
	github.com/go-playground/validator/v10 v10.30.3
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/modelcontextprotocol/go-sdk v1.8.0
)

require (
	github.com/bytedance/gopkg v0.1.4 // indirect
	github.com/bytedance/sonic v1.15.2 // indirect
	github.com/bytedance/sonic/loader v0.5.2 // indirect
	github.com/cloudwego/base64x v0.1.7 // indirect
	github.com/dlclark/regexp2/v2 v2.8.1 // indirect
	github.com/dop251/goja v0.0.0-20261018113103-d42f16bb6161 // indirect
	github.com/gabriel-vasile/mimetype v1.4.15 // indirect
	github.com/gin-contrib/sse v1.1.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/leodido/go-urn v1.5.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.61.0 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.2 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.mongodb.org/mongo-driver/v2 v2.8.0 // indirect
	golang.org/x/arch v0.29.0 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/Giuseppe-Compagnone/lwn-contracts => ../../packages/contracts

replace github.com/Giuseppe-Compagnone/lwn-engine => ../../packages/engine

go 1.26.5
