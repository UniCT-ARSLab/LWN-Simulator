FROM golang:1.20-alpine3.17 as build

RUN apk update
RUN apk add make

ADD . /build/src
WORKDIR /build/src
#RUN make install-dep
RUN go install github.com/rakyll/statik@latest
RUN make build

# deployment image
FROM alpine:3.17.2
WORKDIR /app

COPY --from=build /build/src/bin/config.json /app/config.json
COPY --from=build /build/src/bin/lwnsimulator /app/lwnsimulator

EXPOSE 8000

ENTRYPOINT ["/app/lwnsimulator"]
