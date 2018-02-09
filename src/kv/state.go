package kv

import (
	"../core"
	"sync"
	"log"
)

type RaftKV struct {
	sync.Mutex

	me      int
	id      string
	rf      *core.StateManager
	applyCh *chan core.LogEntry

	requestHandlers map[string]chan core.OpResult
	data            map[string]string
	//latestRequests  map[int64]int64 // Client ID -> Last applied Request ID
}

func MakeRaftKV(rf *core.StateManager, applyCh *chan core.LogEntry) *RaftKV {
	kv := &RaftKV{
		rf:              rf,
		applyCh:         applyCh,
		requestHandlers: make(map[string]chan core.OpResult),
		data:            make(map[string]string),
	}
	go kv.runApplyWorker()
	return kv
}

func (kv *RaftKV) runApplyWorker() {
	for {
		select {
		case e := <-*kv.applyCh:
			log.Printf("Revecied entry for applying index=%d, term=%d", e.Index, e.Term)

			switch op := e.Command.(type) {
			case core.Op:
				result := core.OpResult{Value: "OK"}

				switch op.Command {
				case core.Put:
					kv.data[op.Key] = op.Value
				case core.Append:
					kv.data[op.Key] += op.Value
				case core.Get:
					if val, isPresent := kv.data[op.Key]; isPresent {
						result = core.OpResult{Value: val}
					}else{
						result = core.OpResult{Value: "NotFound"}
					}
				}

				if handler, isPresent := kv.requestHandlers[op.RequestId]; isPresent {
					handler <- result
				}
			}

		}
	}
}
