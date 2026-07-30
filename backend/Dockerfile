# Imagem de DESENVOLVIMENTO: hot reload com air e a doc interativa do Swagger.
# As duas coisas ficam só aqui — a imagem de produção é a Dockerfile.prod, que
# não instala swag e compila sem a build tag que liga a doc.
FROM golang:1.26-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@latest
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["sh", "-c", "swag init -g cmd/api/main.go -o docs && air -c .air.toml"]
