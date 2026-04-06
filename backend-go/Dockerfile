# ---- build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o backend .

# ---- runtime stage ----
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/backend .

EXPOSE 8000

ENTRYPOINT ["./backend"]
