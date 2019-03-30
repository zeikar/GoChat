FROM golang:alpine

WORKDIR /go/src/
COPY src .

WORKDIR /go/src/chat

EXPOSE 8088

CMD go run chat