package core

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type APIContext struct {
	manager *StateManager
}

func (a *APIContext) New(c *gin.Context) {
	state := a.manager.Ask(CheckState{})
	if state.state != "Leader" {
		c.String(http.StatusConflict, "Not a Leader")
	}

	a.manager.Ask(CreateEntry{c.Param("command")})
}

func (a *APIContext) State(c *gin.Context) {
	state := a.manager.Ask(CheckState{})

	c.JSON(http.StatusOK, gin.H{
		"state": state.state,
		"term": state.currentTerm,
		"commitIndex": state.commitIndex,
	})
}

func (a *APIContext) Log(c *gin.Context) {
	state := a.manager.Ask(CheckState{})
	logs := make([]gin.H, len(state.log))
	for i, v := range state.log {
		logs[i] = gin.H{
			"term": v.Term,
			"index": v.Index,
			"command": v.Command,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
	})
}

func MakeApi(m *StateManager, e *gin.Engine) {
	a := APIContext{manager: m}
	e.GET("/state", a.State)
	e.POST("/new/:command", a.New)
	e.GET("/logs", a.Log)
}


