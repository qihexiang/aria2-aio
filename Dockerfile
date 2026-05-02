# Stage 1: Build frontend
FROM node:22-alpine AS frontend

WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.24-alpine AS backend

WORKDIR /app

ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY ui/embed.go ui/
COPY --from=frontend /app/ui/dist/ ui/dist/
COPY internal/ internal/
COPY cmd/ cmd/

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /aria2-aio ./cmd/aria2-aio/

# Stage 3: Runtime
FROM alpine:3.21

RUN apk add --no-cache aria2

COPY --from=backend /aria2-aio /usr/local/bin/aria2-aio

RUN mkdir -p /data

WORKDIR /data

EXPOSE 8080

ENTRYPOINT ["aria2-aio"]
CMD ["--data-dir", "/data"]