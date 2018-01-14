FROM golang:1.9-stretch

WORKDIR /go/src/github.com/pyalex/ucu-raft

RUN go get google.golang.org/grpc
RUN go get google.golang.org/grpc/reflection
RUN go get github.com/gin-gonic/gin

COPY src .

# RUN go get -v ./...
# RUN go install -v ./...

RUN go build main.go

EXPOSE 3000

CMD ["./main"]