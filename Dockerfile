# build stage

FROM golang:latest AS builder

LABEL maintainer="undefcat <undefcat@gmail.com>"

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o main cmd/main.go

# run stage

FROM scratch

LABEL maintainer="undefcat <undefcat@gmail.com>"

WORKDIR /app

COPY --from=builder /app/main main

EXPOSE 8000

ENTRYPOINT ["./main"]

CMD ["-port=8000"]
