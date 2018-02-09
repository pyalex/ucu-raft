package core

import (
	"time"
	"log"
	"math/rand"
)


type CheckState struct { reqTerm uint64; append bool; serverId uint64 }
type StartElection struct {}
type PromoteToLeader struct {currentTerm uint64}
type VoteFor struct { candidateId uint64; lastLogIndex uint64; lastLogTerm uint64}
type AppendEntries struct {startPos int; entries []LogEntry}
type CommitEntries struct {commitIndex uint64}
type UpdateIndexes struct {peerIdx int; lastReplicated LogEntry}
type LookupNextIndex struct {peerIdx int}
type CreateEntry struct {Command interface{}}


type Request struct {
	req interface {}
	respCh chan RaftState
}


type StateManager struct {
	requestCh *chan Request
	heartbeatCh *chan struct{}
	appendCh *chan interface{}
	state *RaftState
}

func (m *StateManager) Ask(req interface{}) RaftState {
	respCh := make(chan RaftState, 1)
	*m.requestCh <- Request{req, respCh}
	return <- respCh
}

func (m *StateManager) electionMonitor() {
	electionTimeout := func() time.Duration {
		rand.Seed(time.Now().UnixNano())
		return (300 + time.Duration(rand.Intn(500))) * time.Millisecond
	}

	for {
		select {
		case <-time.After(electionTimeout()):
			log.Print("Election timer raises")

			state := m.Ask(StartElection{})
			if state.state == Candidate {
				BeginElection(state, m)
			}

		case <-*m.heartbeatCh:
			//log.Printf("Received heartbeat")
		}
	}
}

func (m *StateManager) manager() {
	for r := range *m.requestCh {
		switch req := r.req.(type) {
		case CheckState:
			if req.reqTerm > m.state.currentTerm {
				m.state.becomeFollower(req.reqTerm)
			}

			if m.state.state != Follower && req.append && req.reqTerm == m.state.currentTerm {
				log.Printf("Appraising new Leader %v", req.serverId)
				m.state.state = Follower
			}

		case StartElection:
			m.state.startElection()

		case PromoteToLeader:
			if m.state.state != Candidate || m.state.currentTerm != req.currentTerm {
				continue
			}

			m.state.becomeLeader()
			go LeaderBroadcaster(*m.state, m)
		case VoteFor:
			if m.state.votedFor == NoVote && m.state.checkUpToDate(req.lastLogIndex, req.lastLogTerm) {
				m.state.votedFor = req.candidateId
			}

		case AppendEntries:
			if m.state.state == Follower {
				m.state.appendEntries(req.startPos, req.entries)
			}
		case CommitEntries:
			if m.state.state == Follower {
				m.state.commit(req.commitIndex)
			}
		case UpdateIndexes:
			if m.state.state == Leader {
				m.state.updateIndexes(req.peerIdx, req.lastReplicated)
				m.state.updateCommitIndex()
			}
		case LookupNextIndex:
			m.state.nextIndex[req.peerIdx]--
			log.Printf("Next Index updated %+v", m.state.nextIndex)
		case CreateEntry:
			m.state.createLog(req.Command)
			if m.state.state == Leader {
				*m.appendCh <- struct {}{}
			}
		}

		r.respCh <- *m.state
	}
}

func MakeStateManager(s *RaftState) *StateManager {
	reqCh := make(chan Request)
	heartbeatCh := make(chan struct{})
	appendCh := make(chan interface{})

	manager := StateManager{
		requestCh: &reqCh,
		heartbeatCh: &heartbeatCh,
		appendCh: &appendCh,
		state: s,
	}

	go manager.manager()
	go manager.electionMonitor()

	return &manager
}