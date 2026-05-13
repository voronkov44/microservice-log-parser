FROM golang:1.26 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY proto ./proto
COPY app ./app

ENV CGO_ENABLED=0

RUN go build -o /app ./app

FROM alpine:3.20

COPY --from=build /app /app

ENTRYPOINT ["/app", "-config", "/config.yaml"]