package tcp

import (
	"context"
	"fmt"
	"sync"
)

type Pool struct {
	// lock sync.Mutex

	waitLock  sync.Mutex
	waitQueue []chan Conn

	connect ConnProvider

	all   map[Conn]struct{}
	inUse map[Conn]struct{}
	free  map[Conn]struct{}
}

func NewPool() *Pool {
	return &Pool{
		all:   make(map[Conn]struct{}),
		inUse: make(map[Conn]struct{}),
		free:  make(map[Conn]struct{}),
	}
}

func (p *Pool) SetConnectionProvider(connect ConnProvider) {
	// p.lock.Lock()
	// defer p.lock.Unlock()
	p.connect = connect
}

func (p *Pool) Size() int {
	// p.lock.Lock()
	// defer p.lock.Unlock()
	return len(p.all)
}

func (p *Pool) Free() int {
	// p.lock.Lock()
	// defer p.lock.Unlock()
	return len(p.free)
}

func (p *Pool) Add(c Conn) {
	p.all[c] = struct{}{}
	p.free[c] = struct{}{}
}

func (p *Pool) Remove(c Conn) {
	delete(p.all, c)
	delete(p.free, c)
	delete(p.inUse, c)
}

func (p *Pool) Aquire() (Conn, error) {
	if len(p.free) == 0 {
		return nil, fmt.Errorf("no available connections")
	}
	var c Conn
	for f := range p.free {
		c = f
		break
	}
	delete(p.free, c)
	p.inUse[c] = struct{}{}
	return c, nil
}

func (p *Pool) Release(c Conn) {
	if _, has := p.inUse[c]; !has {
		return
	}
	delete(p.inUse, c)
	p.free[c] = struct{}{}
}

func (p *Pool) Notify(c Conn) bool {
	notified := false
	fmt.Println("waiting on lock to notify")
	p.waitLock.Lock()
	if len(p.waitQueue) > 0 {
		p.waitQueue[0] <- c
		close(p.waitQueue[0])
		p.waitQueue = p.waitQueue[1:]
		notified = true
	}
	fmt.Println("done notifying")
	p.waitLock.Unlock()
	return notified
}

func (p *Pool) Wait(ctx context.Context) (Conn, error) {
	p.waitLock.Lock()
	wait := make(chan Conn, 1)
	p.waitQueue = append(p.waitQueue, wait)
	p.waitLock.Unlock()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case c := <-wait:
		return c, nil
	}
}
