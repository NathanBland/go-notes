FROM golang:1.25-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates git

ARG TARGETOS=linux
ARG TARGETARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /out/go-notes ./cmd/api

FROM migrate/migrate:v4.19.0 AS migrate

FROM alpine:3.22
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /out/go-notes /usr/local/bin/go-notes
COPY --from=migrate /usr/local/bin/migrate /usr/local/bin/migrate
COPY migrations /app/migrations

EXPOSE 8080

CMD ["go-notes"]
