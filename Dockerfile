FROM golang:1.26-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@latest
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["sh", "-c", "swag init -g cmd/api/main.go -o docs && air -c .air.toml"]
