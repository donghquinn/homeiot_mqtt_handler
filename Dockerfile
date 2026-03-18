# ============ BASE ===========
FROM golang:1.25.4-alpine3.22 AS base

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# ========= BUILDER ==========
FROM base AS builder

# Install security updates
RUN apk update && apk upgrade && apk add --no-cache ca-certificates

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN go build -ldflags="-w -s -extldflags '-static'" -a -installsuffix cgo -o backend cmd/main.go

# ========= RUNNER ==========
FROM alpine:3.22 AS release

ARG USER_ID=1000
ARG GROUP_ID=1000

RUN apk update && apk upgrade && \
    apk add --no-cache ca-certificates tzdata && \
    rm -rf /var/cache/apk/*

RUN addgroup -g ${GROUP_ID} -S node && \
    adduser -u ${USER_ID} -S node -G node -h /home/node -s /sbin/nologin

WORKDIR /home/node

COPY --from=builder --chown=${USER_ID}:${GROUP_ID} /app/backend ./backend

RUN mkdir -p logs && \
    chown -R ${USER_ID}:${GROUP_ID} /home/node logs

USER node

CMD ["./backend"]