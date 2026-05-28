# syntax=docker/dockerfile:1.7

# --- build stage ---------------------------------------------------------
FROM golang:1.25-alpine AS build

WORKDIR /src

# Install templ at the same version pinned in go.mod
RUN apk add --no-cache git \
 && go install github.com/a-h/templ/cmd/templ@v0.3.1020

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Source
COPY . .

# Generate templ files, then build a static binary
RUN templ generate ./... \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
      -o /out/server ./cmd/server

# --- runtime stage -------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/server /app/server
COPY static /app/static

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
