package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestAppendEntries(t *testing.T) {
	s := RaftState{}
	logs := make([]LogEntry, 3)
	logs[0] = LogEntry{1, 1, "a"}
	logs[1] = LogEntry{2, 1, "b"}
	logs[2] = LogEntry{3, 1, "c"}

	s.appendEntries(0, logs)
	assert.Len(t, s.log, 3)
	assert.Contains(t, s.log, logs[0])

	s.appendEntries(0, logs)
	assert.Len(t, s.log, 3)

	logs[2] = LogEntry{4, 1, "d"}
	s.appendEntries(0, logs)
	assert.Len(t, s.log, 3)
	assert.NotContains(t, s.log, LogEntry{3, 1, "c"})
	assert.Contains(t, s.log, LogEntry{4, 1, "d"})

	logs = make([]LogEntry, 2)
	logs[0] = LogEntry{5, 1, "e"}
	logs[1] = LogEntry{6, 1, "f"}
	s.appendEntries(3, logs)
	assert.Len(t, s.log, 5)
	assert.Contains(t, s.log, logs[0])

	s.appendEntries(3, logs)
	assert.Len(t, s.log, 5)

	logs[1] = LogEntry{7, 1, "g"}
	s.appendEntries(3, logs)
	assert.Len(t, s.log, 5)
	assert.NotContains(t, s.log, LogEntry{6, 1, "f"})
	assert.Contains(t, s.log, LogEntry{7, 1, "g"})

	logs = make([]LogEntry, 1)
	logs[0] = LogEntry{8, 1, "h"}
	s.appendEntries(3, logs)
	assert.Len(t, s.log, 4)
	assert.NotContains(t, s.log, LogEntry{5, 1, "e"})
	assert.NotContains(t, s.log, LogEntry{7, 1, "g"})
	assert.Contains(t, s.log, LogEntry{8, 1, "h"})
}


func TestPrepareAppendRequest(t *testing.T) {
	s := RaftState{}

	s.nextIndex = []uint64{1}
	r := prepareAppendRequest(s, 0)
	assert.Equal(t, uint64(0), r.PrevLogIndex)
	assert.Equal(t, uint64(0), r.PrevLogTerm)

	s.log = []LogEntry{{2, 1, "a"}, {4, 1, "b"}}
	r = prepareAppendRequest(s, 0)
	assert.Equal(t, uint64(4), r.PrevLogIndex)
	assert.Equal(t, uint64(1), r.PrevLogTerm)

	s.nextIndex = []uint64{5}
	r = prepareAppendRequest(s, 0)
	assert.Equal(t, uint64(4), r.PrevLogIndex)
	assert.Equal(t, uint64(1), r.PrevLogTerm)
	assert.Len(t, r.Entries, 0)

	s.nextIndex = []uint64{4}
	r = prepareAppendRequest(s, 0)
	assert.Equal(t, uint64(2), r.PrevLogIndex)
	assert.Equal(t, uint64(1), r.PrevLogTerm)
	assert.Len(t, r.Entries, 1)

	s.nextIndex = []uint64{2}
	r = prepareAppendRequest(s, 0)
	assert.Equal(t, uint64(0), r.PrevLogIndex)
	assert.Equal(t, uint64(0), r.PrevLogTerm)
	assert.Len(t, r.Entries, 2)
	}