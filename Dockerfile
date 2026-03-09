FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /barber_bot ./cmd/barber_bot

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Europe/Moscow
WORKDIR /app
COPY --from=build /barber_bot .
VOLUME ["/app/data", "/app/backups"]
ENV DATABASE_DSN=file:/app/data/barber_bot.db?_journal_mode=WAL
ENV BACKUP_DIR=/app/backups
EXPOSE 8080
ENTRYPOINT ["./barber_bot"]
