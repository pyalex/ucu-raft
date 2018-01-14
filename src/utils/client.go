package utils

import (
	"log"
	"google.golang.org/grpc"
	"github.com/pyalex/ucu-raft/proto"
	context "golang.org/x/net/context"
	)

func main() {
	conn, err := grpc.Dial("localhost:3000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Dial failed: %v", err)
	}
	gcdClient := proto.NewRaftServiceClient(conn)
	res, err := gcdClient.AppendEntries(context.Background(), &proto.AppendEntries_Request{Term: 5})
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	log.Print(*res)
}
