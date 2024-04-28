package tcp

import "sync"

type Upstream struct {
	Conn

	qLock sync.Mutex
	queue chan Conn
}
