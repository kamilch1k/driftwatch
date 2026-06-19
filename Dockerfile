FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -trimpath -ldflags="-s -w" -o /out/driftwatch-api ./cmd/api

FROM alpine:3.22
RUN adduser -D -H driftwatch
USER driftwatch
COPY --from=build /out/driftwatch-api /usr/local/bin/driftwatch-api
EXPOSE 8080
ENTRYPOINT ["driftwatch-api"]
