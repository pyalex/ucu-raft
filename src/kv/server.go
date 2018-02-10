package kv

import (
	"../core"
	"github.com/satori/go.uuid"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"log"
)

const waitTimeout = 500 * time.Millisecond

type ServerContext struct {
	manager *core.StateManager
	kv      *RaftKV
}

func (s *ServerContext) await(op core.Op, c *gin.Context) {
	await := make(chan core.OpResult)

	log.Printf("Creating new handler %s", op.RequestId)

	s.kv.Lock()
	s.kv.requestHandlers[op.RequestId] = await
	s.kv.Unlock()

	log.Printf("Creating new log %+v", op)

	s.manager.Ask(core.CreateEntry{op})
	select {
	case r := <-await:
		c.String(http.StatusOK, "%s\n", r.Value)
	case <-time.After(waitTimeout):
		c.String(http.StatusInternalServerError, "%s\n", "Timeout")
	}

	log.Printf("Operation finished %s", op.RequestId)

	s.kv.Lock()
	delete(s.kv.requestHandlers, op.RequestId)
	s.kv.Unlock()
}

func (s *ServerContext) Get(c *gin.Context) {
	log.Printf("GET %s", c.Param("key"))

	id, _ := uuid.NewV4()
	op := core.Op{Command: core.Get, Key: c.Param("key"), RequestId: id.String()}

	s.await(op, c)
}

func (s *ServerContext) Put(c *gin.Context) {
	log.Printf("PUT %s=%s", c.Param("key"), c.Param("value"))

	id, _ := uuid.NewV4()
	op := core.Op{Command: core.Put, Key: c.Param("key"), Value: c.Param("value"), RequestId: id.String()}

	s.await(op, c)
}

func (s *ServerContext) Append(c *gin.Context) {
	log.Printf("POST %s=%s", c.Param("key"), c.Param("value"))

	id, _ := uuid.NewV4()
	op := core.Op{Command: core.Append, Key: c.Param("key"), Value: c.Param("value"), RequestId: id.String()}

	s.await(op, c)
}

func MakeKVServer(m *core.StateManager, ch *chan core.LogEntry, e *gin.Engine) {
	kv := MakeRaftKV(m, ch)
	s := ServerContext{manager: m, kv: kv}

	e.GET("/kv/:key", s.Get)
	e.PUT("/kv/:key/:value", s.Put)
	e.POST("/kv/:key/:value", s.Append)
}
