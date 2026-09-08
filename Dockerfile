FROM golang:1.26.2-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY ent ./ent
COPY internal ./internal
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/paper-bot ./cmd/paper-bot

FROM alpine:3.22

RUN apk add --no-cache ca-certificates su-exec \
    && addgroup -S app \
    && adduser -S -G app -h /app app
WORKDIR /app
COPY --from=build /out/paper-bot /app/paper-bot
COPY --chmod=755 scripts/docker-entrypoint.sh /app/docker-entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["/app/docker-entrypoint.sh"]
