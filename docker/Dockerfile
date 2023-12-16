
FROM golang:1.21.5 as build-stage

WORKDIR /app

COPY go.mod go.sum /api/.env /api ./
RUN go mod download
RUN mkdir /data && chown 65534:65534 /data

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /app/api -ldflags="-extldflags=-static"

FROM scratch

ENV GIN_MODE=release
VOLUME [ "/data" ]
WORKDIR /app

COPY --from=build-stage /data /data
COPY --from=build-stage /app/api /app/api
COPY --from=build-stage /app/config.json /app/config.json
COPY --from=build-stage /app/.env /app/.env

EXPOSE 8080
CMD ["/app/api", "--cfg=/app/config.json"]
