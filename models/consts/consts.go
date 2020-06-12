package consts

// server status
const (
	Idle = iota
	Working
)

// connection type
const (
	CtlConn = iota
	WorkConn
)

const (
	// msg from client to server
	C2SHeartBeatReq = 1
	// msg from server to client
	S2CHeartBeatRes = 100
)
