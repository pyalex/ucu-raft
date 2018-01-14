package core

import (
	"github.com/pyalex/ucu-raft/proto"
	"context"
	"time"
	"log"
	"google.golang.org/grpc"
	"bytes"
	"encoding/gob"
)

func BeginElection(state RaftState, manager *StateManager) {
	results := make(chan *proto.RequestVote_Response, len(state.peers))

	var lastLog LogEntry
	if len(state.log) != 0 {
		lastLog = state.log[len(state.log)-1]
	} else {
		lastLog = LogEntry{}
	}

	args := proto.RequestVote_Request{
		Term:         state.currentTerm,
		ServerId:     state.id,
		LastLogIndex: lastLog.Index,
		LastLogTerm:  lastLog.Term,
	}

	log.Printf("Start sending request vote to other peers %d", len(state.peers))

	request := func(host PeerHost) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		p := getPeer(host)
		res, err := p.RequestVote(ctx, &args)
		if err == nil {
			results <- res
		} else {
			log.Printf("Error %v received from peer %v", err, host)
			results <- &proto.RequestVote_Response{}
		}
	}

	for i := range state.peers {
		go request(state.peers[i])
	}

	voted := 1

	for range state.peers {
		resp := <-results
		log.Printf("Vote Response recieved %+v", resp)

		if resp.Term > args.Term {
			manager.Ask(CheckState{reqTerm: resp.Term})
			break
		}

		if resp.Term == args.Term && resp.Granted {
			voted += 1
		}

		log.Printf("Collected %d votes", voted)

		if voted > len(state.peers)/2 {
			manager.Ask(PromoteToLeader{args.Term})
			break
		}
	}
}

func LeaderBroadcaster(state RaftState, manager *StateManager) {
	log.Print("Broadcaster started")
	ticker := time.NewTicker(100 * time.Millisecond)

	request := func(peerIdx int) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		p := getPeer(state.peers[peerIdx])
		req := prepareAppendRequest(state, peerIdx)
		if len(req.Entries) > 0 {
			log.Printf("New Request for %d prepared %+v", peerIdx, req)
		}
		resp, err := p.AppendEntries(ctx, req)
		if err != nil {
			log.Printf("Error %v received from peer %v", err, state.peers[peerIdx])
			return
		}

		if resp.Term != state.currentTerm {
			return
		}

		if resp.Success && len(req.Entries) > 0 {
			log.Printf("Peer %d succesfully feeded", peerIdx)
			lastEntry := state.log[len(state.log) - 1]
			manager.Ask(UpdateIndexes{peerIdx, lastEntry})
		}

		if resp.Success == false {
			log.Printf("Peer's log %d is not synhronized. Will lookup for right index", peerIdx)
			manager.Ask(LookupNextIndex{peerIdx})
			*manager.appendCh <- peerIdx
		}
	}

	go func() {
		*manager.appendCh <- struct {}{}
		log.Print("Initial heartbeat")
	}()

	for {
		state = manager.Ask(CheckState{reqTerm: state.currentTerm})
		if state.state != Leader {
			log.Print("Broadcaster finished")
			ticker.Stop()
			break
		}

		select {
		case r := <-*manager.appendCh:
			log.Printf("AppendRequest will be send for %+v", r)
			switch r := r.(type) {
			case int:
				go request(r)
			default:
				for i := range state.peers {
					go request(i)
				}
			}


		case <-ticker.C:

			// keep itself aware :)
			*manager.heartbeatCh <- struct{}{}

			for i := range state.peers {
				go request(i)
			}
		}
	}
}

func prepareAppendRequest(state RaftState, peerIdx int) *proto.AppendEntries_Request {
	nextIndex := state.nextIndex[peerIdx]
	var prevLog LogEntry
	var entries []*proto.Entry
	next := len(state.log)

	for i, v := range state.log {
		if v.Index == nextIndex {
			next = i

			if i > 0 {
				prevLog = state.log[i-1]
			} else {
				prevLog = LogEntry{}
			}
			break
		}

	}

	entries = make([]*proto.Entry, len(state.log)-next)
	for i, v := range state.log[next:] {
		entries[i] = &proto.Entry{
			Term: v.Term, Index: v.Index, Data: getBytes(v.Command),
		}
	}

	return &proto.AppendEntries_Request{
		Term:         state.currentTerm,
		ServerId:     state.id,
		CommitIndex:  state.commitIndex,
		PrevLogTerm:  prevLog.Term,
		PrevLogIndex: prevLog.Index,
		Entries:      entries,
	}
}

func getPeer(host PeerHost) proto.RaftServiceClient {
	conn, err := grpc.Dial(string(host), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Dial failed: %v %v", err, host)
	}
	return proto.NewRaftServiceClient(conn)

}

func getBytes(key interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(key)
	return buf.Bytes()
}
