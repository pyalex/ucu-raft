package main

import (
	"net"
	"log"
	"github.com/pyalex/ucu-raft/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"./core"
	"os"
	"strings"
	"strconv"
	"fmt"
	"./kv"
	"github.com/gin-gonic/gin"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("Raft Server Started (%d)", getMyId(1))

	listen, err := net.Listen("tcp", ":3000")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	applyCh := make(chan core.LogEntry)
	state := core.MakeRaftState(getPeers(), getMyId(1), getPersistPath(), &applyCh)

	manager := core.MakeStateManager(state)
	impl := &core.Server{manager}

	proto.RegisterRaftServiceServer(s, impl)
	reflection.Register(s)

	e := gin.Default()
	core.MakeApi(manager, e)
	kv.MakeKVServer(manager, &applyCh, e)
	go e.Run()

	if err := s.Serve(listen); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func getPeers() []core.PeerHost {
	hosts := strings.Split(os.Getenv("PEERS"), ",")
	peers := make([]core.PeerHost, len(hosts))
	for i := range hosts{
		peers[i] = core.PeerHost(hosts[i])
	}

	return peers
}

func getMyId(def uint64) uint64 {
	id, err := strconv.ParseUint(os.Getenv("MY_ID"), 0, 64)
	if err != nil {
		return def
	}
	return id
}

func getPersistPath() string {
	return fmt.Sprintf("/state/persist-state-%d.raft", getMyId(0))
}