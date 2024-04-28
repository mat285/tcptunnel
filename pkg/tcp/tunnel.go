package tcp

import (
	"context"
	"sync"
)

type Tunnel struct {
	lock         sync.Mutex
	running      bool
	stop         chan struct{}
	conn1, conn2 Conn
}

func NewTunnel(c1, c2 Conn) *Tunnel {
	if c1 == nil || c2 == nil {
		panic("stop")
	}
	// if c1.State() != ConnStateOpen || c2.State() != ConnStateOpen {
	// 	panic("stop again")
	// }
	return &Tunnel{
		conn1:   c1,
		conn2:   c2,
		running: false,
	}
}

func (t *Tunnel) Run(ctx context.Context) error {
	// t.lock.Lock()
	// if t.running {
	// 	t.lock.Unlock()
	// 	return fmt.Errorf("already running")
	// }
	// t.running = true
	errs := make(chan error, 2)
	stop := make(chan struct{})
	t.lock.Lock()
	t.stop = stop
	t.lock.Unlock()

	go func() {
		errs <- t.readAndForward(ctx, t.stop, t.conn1, t.conn2)
	}()
	go func() {
		errs <- t.readAndForward(ctx, t.stop, t.conn2, t.conn1)
	}()

	var err error
	select {
	case err = <-errs:
	case <-stop:
	}
	t.conn1.Close(ctx)
	t.conn2.Close(ctx)
	return err
	// t.lock.Lock()
	// if t.stop != nil {
	// 	close(t.stop)
	// 	t.stop = nil
	// }
	// t.lock.Unlock()
	// err2 := <-errs

	// t.lock.Lock()
	// t.conn1.Close(ctx)
	// t.conn2.Close(ctx)
	// t.lock.Unlock()

	// errStr := ""
	// if err1 != nil {
	// 	errStr += err1.Error()
	// }
	// if err2 != nil {
	// 	errStr += err2.Error()
	// }
	// if len(errStr) > 0 {
	// 	return fmt.Errorf(errStr)
	// }
	// return nil
}

func (t *Tunnel) Stop(ctx context.Context) error {
	// if !t.running {
	// 	t.lock.Unlock()
	// 	return nil
	// }
	// if t.stop != nil {
	// 	close(t.stop)
	// 	t.stop = nil
	// }
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.stop != nil {
		close(t.stop)
		t.stop = nil
	}
	return nil
}

func (t *Tunnel) readAndForward(ctx context.Context, stop chan struct{}, read, write Conn) error {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stop:
			return nil
		default:
		}

		n, err := read.Read(ctx, buf)
		if err != nil {
			return err
		}
		err = write.Write(ctx, buf[:n])
		if err != nil {
			return err
		}
	}
}
