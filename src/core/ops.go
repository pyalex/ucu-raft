package core


type CommandType int
type Err string

const (
	Put CommandType = iota
	Append
	Get
)

type Op struct {
	Command   CommandType
	Key       string
	Value     string
	RequestId string
	ClientId  int64
}


type OpResult struct {
	Value string
}
