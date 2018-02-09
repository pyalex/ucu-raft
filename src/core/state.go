package core

import (
	"log"
	"sort"
)

type ServerState string
type PeerHost string

const (
	Follower  ServerState = "Follower"
	Candidate             = "Candidate"
	Leader                = "Leader"
)
const NoVote = 999

type LogEntry struct {
	Index   uint64
	Term    uint64
	Command interface{}
}

type RaftState struct {
	state ServerState
	//me string
	id uint64

	peers       []PeerHost
	persistPath string

	currentTerm uint64
	votedFor    uint64
	leaderID    string

	log         []LogEntry
	commitIndex uint64
	lastApplied uint64

	// Leader state
	nextIndex  []uint64 // For each peer, index of next log entry to send that server
	matchIndex []uint64 // For each peer, index of highest entry known log entry known to be replicated on peer

	applyCh *chan LogEntry
}

func MakeRaftState(peers []PeerHost, myId uint64, persistPath string, applyCh *chan LogEntry) *RaftState {
	s := &RaftState{
		peers:       peers,
		id:          myId,
		nextIndex:   make([]uint64, len(peers)),
		matchIndex:  make([]uint64, len(peers)),
		persistPath: persistPath,
		applyCh:     applyCh,
	}
	// PersistLoad(s.persistPath, s)
	return s
}

func (s *RaftState) becomeFollower(newTerm uint64) {
	s.currentTerm = newTerm
	s.votedFor = NoVote
	s.state = Follower
	//PersistSave(s.persistPath, s)
	log.Printf("Becoming follower with term %d", s.currentTerm)
}

func (s *RaftState) startElection() {
	s.currentTerm += 1
	s.votedFor = s.id
	s.state = Candidate
	//PersistSave(s.persistPath, s)
	log.Printf("Starting new election with term %d", s.currentTerm)
}

func (s *RaftState) becomeLeader() {
	s.state = Leader

	// When a leader first comes to power,
	// it initializes all nextIndex values to the index just after the
	// last one in its log
	var lastLog LogEntry
	if len(s.log) > 0 {
		lastLog = s.log[len(s.log)-1]
	} else {
		lastLog = LogEntry{}
	}
	for i := range s.peers {
		s.nextIndex[i] = lastLog.Index + 1
	}

	//PersistSave(s.persistPath, s)
	log.Printf("State Leader commited with nextIndex %v", s.nextIndex)
}

func (s *RaftState) FindLogEntryPos(index uint64, term uint64) int {
	for i, v := range s.log {
		if v.Index == index {
			if v.Term == term {
				return i
			}
			return -1
		}
	}
	return -1
}

func (s *RaftState) appendEntries(startPos int, entries []LogEntry) {
	common := 0
	log.Printf("Appending %d entries to log(%d) starting from %d", len(entries), len(s.log), startPos)

	// compare existing log and new entries and find maximum common index
	for i := 0; i < len(s.log)-startPos; i++ {
		if i >= len(entries) || entries[i].Index != s.log[startPos+i].Index || entries[i].Term != s.log[startPos+i].Term {
			break
		}
		common = i + 1
	}

	log.Printf("%d already existing log found. %d are the same", len(s.log)-startPos, common)
	log.Printf("%d logs will be deleted. %d will be appended", len(s.log)-startPos-common, len(entries)-common)

	// If an existing entry conflicts with a new one (same index
	// but different terms), delete the existing entry and all that
	// follow it (§5.3)
	if common < len(s.log)-startPos {
		s.log = s.log[:startPos+common]
	}

	//  Append any new entries not already in the log
	s.log = append(s.log, entries[common:]...)
	//PersistSave(s.persistPath, s)
}

func (s *RaftState) checkUpToDate(lastLogIndex uint64, lastLogTerm uint64) bool {
	if len(s.log) == 0 {
		return true
	}

	lastLog := s.log[len(s.log)-1]
	if lastLog.Term == lastLogTerm {
		return lastLog.Index <= lastLogIndex
	}
	return lastLog.Term < lastLogTerm
}

func (s *RaftState) commit(commitIndex uint64) {
	if len(s.log) == 0 {
		return
	}

	previousIndex := s.commitIndex

	lastLog := s.log[len(s.log)-1]
	if lastLog.Index < commitIndex {
		s.commitIndex = lastLog.Index
	} else {
		s.commitIndex = commitIndex
	}

	if s.commitIndex > previousIndex {
		go s.sendToApply(previousIndex, s.commitIndex)
	}
	log.Printf("CommitIndex was updated to %d", s.commitIndex)
}

func (s *RaftState) updateIndexes(peerIdx int, lastReplicated LogEntry) {
	previous := s.matchIndex[peerIdx]

	s.matchIndex[peerIdx] = lastReplicated.Index
	s.nextIndex[peerIdx] = lastReplicated.Index + 1

	if s.matchIndex[peerIdx] != previous {
		log.Printf("Indexes were updated %v %v", s.nextIndex, s.matchIndex)
	}
}

func (s *RaftState) updateCommitIndex() bool {
	// §5.3/5.4: If there exists an N such that N > commitIndex,
	// a majority of matchIndex[i] ≥ N, and log[N].term == currentTerm: set commitIndex = N

	// Let's find median
	// In array [3, 5, 2, 4, 6] median is 4 and it's lowest common index that was delivered to majority

	matchedMedian := median(s.matchIndex...)
	if matchedMedian <= s.commitIndex {
		// nothing to commit
		return false
	}

	log.Printf("Candidate for commitIndex %d, current %d", matchedMedian, s.commitIndex)

	for _, v := range s.log {
		if v.Index == matchedMedian && v.Term == s.currentTerm {
			previous := s.commitIndex
			s.commitIndex = matchedMedian

			go s.sendToApply(previous, s.commitIndex)
			log.Print("CommitIndex updated")

			return true
		}
	}

	return false
}

func (s *RaftState) sendToApply(previous uint64, current uint64) {
	start := s.getPos(previous) + 1
	stop := s.getPos(current) + 1

	log.Printf("Sending entries to apply from %d to %d [%d:%d]", previous, current, start, stop)

	for _, entry := range s.log[start:stop] {
		*s.applyCh <- entry
	}
}

func (s *RaftState) getLastLog() LogEntry {
	if len(s.log) == 0 {
		return LogEntry{}
	}

	return s.log[len(s.log)-1]
}

func (s *RaftState) getPos(index uint64) int {
	for i, entry := range s.log {
		if entry.Index == index {
			return i
		}
	}
	return -1
}

func (s *RaftState) createLog(command interface{}) {
	lastLog := s.getLastLog()
	s.log = append(s.log, LogEntry{
		Command: command,
		Term:    s.currentTerm,
		Index:   lastLog.Index + 1,
	})
	log.Printf("New LogEntry appended %+v", s.getLastLog())
}

func median(arr ...uint64) uint64 {
	ints := make([]int, len(arr))
	for i, v := range arr {
		ints[i] = int(v)
	}
	sort.Ints(ints)
	return uint64(ints[len(arr)/2])
}
