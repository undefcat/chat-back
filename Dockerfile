# build stage

FROM golang:latest AS builder

MAINTAINER "undefcat <undefcat@gmail.com>"

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build cmd/main.go

# run stage

FROM scratch

MAINTAINER "undefcat <undefcat@gmail.com>"

COPY --from=builder /app/main /app/

EXPOSE 8000

ENTRYPOINT ["/app/main"]

CMD ["-port=8000"]
