package core

import (
	"github.com/pyalex/ucu-raft/proto"
	context "golang.org/x/net/context"
	"log"
	//b64 "encoding/base64"
)

type Server struct{
	StateManager *StateManager
}

func (s *Server) AppendEntries(ctx context.Context, req *proto.AppendEntries_Request) (*proto.AppendEntries_Response, error) {
	//log.Printf("AppendEntries %v", s.StateManager)
	state := s.StateManager.Ask(CheckState{reqTerm: req.Term, append: true})

	if state.currentTerm > req.Term {
		// Reply false if term < currentTerm (§5.1)
		return &proto.AppendEntries_Response{Term: state.currentTerm, Success: false}, nil
	}

	*s.StateManager.heartbeatCh <- struct {}{}

	pos := state.FindLogEntryPos(req.PrevLogIndex, req.PrevLogTerm)
	if req.PrevLogIndex != 0 && pos == -1 {
		// Reply false if log doesn’t contain an entry at prevLogIndex
		// whose term matches prevLogTerm (§5.3)
		log.Printf("prevLogIndex needs to be synchronized %d %d", req.PrevLogIndex, req.PrevLogTerm)
		return &proto.AppendEntries_Response{Term: state.currentTerm, Success: false}, nil
	}

	entries := make([]LogEntry, len(req.Entries))
	for i, e := range req.Entries {
		//var buf []byte
		//dec := gob.NewDecoder(&buf)
		//dec.Decode(e.Data)
		//b64.StdEncoding.Decode(buf, e.Data)
		log.Printf("Incoming data %#v", e.Data)
		entries[i] = LogEntry{
			Term: e.Term, Index: e.Index, Command: string(e.Data),
		}
	}

	if len(entries) > 0 {
		pos++

		state = s.StateManager.Ask(AppendEntries{pos, entries})
		if len(state.log) != pos + len(req.Entries) {
			log.Println("Something failed on AppendEntries")
			return &proto.AppendEntries_Response{Term: state.currentTerm, Success: false}, nil
		}
	}

	if state.commitIndex != req.CommitIndex {
		s.StateManager.Ask(CommitEntries{req.CommitIndex})
	}

	return &proto.AppendEntries_Response{Term: state.currentTerm, Success: true}, nil
}

func (s *Server) RequestVote(ctx context.Context, req *proto.RequestVote_Request) (*proto.RequestVote_Response, error) {
	log.Printf("Incoming vote request %+v", *req)

	state := s.StateManager.Ask(CheckState{reqTerm: req.Term})
	if state.currentTerm > req.Term {
		log.Print("Vote request rejected")
		return &proto.RequestVote_Response{Term: state.currentTerm, Granted: false}, nil
	}

	state = s.StateManager.Ask(
		VoteFor{candidateId: req.ServerId,lastLogTerm: req.LastLogTerm, lastLogIndex: req.LastLogIndex})
	log.Printf("Voted for %d", state.votedFor)
	if state.votedFor == req.ServerId {
		return &proto.RequestVote_Response{Term: state.currentTerm, Granted: true}, nil
	}

	return &proto.RequestVote_Response{Term: state.currentTerm, Granted: false}, nil
}


