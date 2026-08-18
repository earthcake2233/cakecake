# cakecake backend image (includes ffmpeg for the async transcode pipeline)
# Build context is the repo root; referenced by docker-compose.yml.
FROM golang:1.25-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY docs ./docs

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cakecake ./cmd/cakecake

FROM alpine:3.22

# Upgrade all OS packages to patched versions before installing (container
# CVE hygiene: the base tag's snapshot may still ship vulnerable packages).
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata ffmpeg

WORKDIR /app

COPY --from=build /out/cakecake .
COPY configs ./configs
COPY migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["./cakecake"]
