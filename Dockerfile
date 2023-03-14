FROM golang:1.20-buster AS build

WORKDIR /app

COPY Makefile ./

COPY go.mod ./
COPY go.sum ./

COPY cmd ./cmd
COPY codes ./codes
COPY controllers ./controllers
COPY models ./models
COPY repositories ./repositories
COPY simulator ./simulator
COPY socket ./socket
COPY webserver ./webserver

COPY config.json ./

RUN make install-dep && make build


FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /app/bin/lwnsimulator lwnsimulator 

COPY docker /

EXPOSE 8000
EXPOSE 8001

USER nonroot:nonroot

ENTRYPOINT [ "./lwnsimulator" ]
