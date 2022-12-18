FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY cronc.go ./

RUN go build -o cronc .

CMD ["./cronc"]
