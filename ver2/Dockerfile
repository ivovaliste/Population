FROM golang:1.19 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -v -o /app/app

FROM alpine

WORKDIR /app
COPY --from=build /app/app /app/app
EXPOSE 8080
ENTRYPOINT [ "./app" ]