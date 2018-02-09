FROM golang:1.9-stretch

RUN apt-get update
RUN apt-get install net-tools

WORKDIR /go/src/github.com/pyalex/ucu-raft

RUN go get google.golang.org/grpc
RUN go get google.golang.org/grpc/reflection
RUN go get github.com/gin-gonic/gin
RUN go get github.com/satori/go.uuid

COPY src .

# RUN go get -v ./...
# RUN go install -v ./...

RUN go build main.go

EXPOSE 3000

CMD ["./main"]