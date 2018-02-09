package core

import (
	"os"
	"encoding/gob"
)

type PersistState struct {
	CurrentTerm uint64
	Log         []LogEntry
	VotedFor    uint64
}



func PersistSave(path string, s *RaftState) error {
	file, err := os.Create(path)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(&PersistState{
			CurrentTerm: s.currentTerm,
			Log: s.log,
			VotedFor: s.votedFor,
		})
	}
	file.Close()
	return err
}


func PersistLoad(path string, s *RaftState) error {
	file, err := os.Open(path)
	obj := PersistState{}
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&obj)
	}
	s.currentTerm, s.votedFor = obj.CurrentTerm, obj.VotedFor
	s.log = append(s.log, obj.Log...)
	file.Close()
	return err
}
