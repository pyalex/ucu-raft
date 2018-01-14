package core

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type APIContext struct {
	manager *StateManager
}

func (a *APIContext) New(c *gin.Context) {
	a.manager.Ask(CreateEntry{c.Param("command")})
}

func (a *APIContext) State(c *gin.Context) {
	state := a.manager.Ask(CheckState{})

	c.JSON(http.StatusOK, gin.H{
		"state": state.state,
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

func MakeApi(m *StateManager) {
	r := gin.Default()
	a := APIContext{manager: m}
	r.GET("/state", a.State)
	r.POST("/new/:command", a.New)
	r.GET("/logs", a.Log)
	r.Run()
	//r.GET("/gcd/:a/:b", func(c *gin.Context) {
	//	// Parse parameters
	//	a, err := strconv.ParseUint(c.Param("a"), 10, 64)
	//	if err != nil {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameter A"})
	//		return
	//	}
	//	b, err := strconv.ParseUint(c.Param("b"), 10, 64)
	//	if err != nil {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameter B"})
	//		return
	//	}
	//	// Call GCD service
	//	req := &pb.GCDRequest{A: a, B: b}
	//	if res, err := gcdClient.Compute(c, req); err == nil {
	//		c.JSON(http.StatusOK, gin.H{
	//			"result": fmt.Sprint(res.Result),
	//		})
	//	} else {
	//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	//	}
	//})
}


