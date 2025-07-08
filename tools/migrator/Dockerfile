FROM golang:1.23-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .
RUN go build -o ./build/migrator ./cmd/migrator

FROM alpine:3
WORKDIR /app
COPY --from=build /app/build/migrator /app/migrator
COPY --from=build /app/migrations ./migrations

ARG MIGRATIONS_PATH=./migrations
ENV MIGRATIONS_PATH=${MIGRATIONS_PATH}



ENTRYPOINT ["/app/migrator"]