package tunnel

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/runner-mei/log"
)

type waitingConn struct {
	c    net.Conn
	addr string
}
type dialQueue struct {
	closed  int32
	name    string
	c       chan waitingConn
	timeout time.Duration
}

func (queue *dialQueue) DialContext(ctx context.Context) (conn net.Conn, err error) {
	var c <-chan struct{}
	if ctx != nil {
		c = ctx.Done()
	}
	timer := time.NewTimer(queue.timeout)
	for {
		select {
		case conn, ok := <-queue.c:
			if !ok {
				return nil, context.Canceled
			}
			timer.Stop()
			return conn.c, nil
		case <-c:
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
			return nil, ErrTimeout
		}
	}
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (queue *dialQueue) Close() error {
	if !atomic.CompareAndSwapInt32(&queue.closed, 0, 1) {
		return nil
	}

	for {
		select {
		case conn := <-queue.c:
			conn.c.Close()
		default:
			close(queue.c)
			return nil
		}
	}
}

type TunnelServer struct {
	logger     log.Logger
	closed     int32
	maxThreads uint32
	mu         sync.RWMutex
	dialQueues map[string]*dialQueue
	timeout    time.Duration
	signel     func(name string)
}

func (srv *TunnelServer) createQueue(engineName string) (*dialQueue, error) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	if 0 != atomic.LoadInt32(&srv.closed) {
		return nil, ErrNotActive
	}

	var queue *dialQueue
	if nil == srv.dialQueues {
		srv.dialQueues = map[string]*dialQueue{}
	} else {
		queue = srv.dialQueues[engineName]
	}
	if queue == nil {
		queue = &dialQueue{
			name: engineName,
			c:    make(chan waitingConn, srv.maxThreads),
		}
		srv.dialQueues[engineName] = queue
	}
	return queue, nil
}
func (srv *TunnelServer) Dial(engineName string) (net.Conn, error) {
	if 0 != atomic.LoadInt32(&srv.closed) {
		return nil, ErrNotActive
	}

	var queue *dialQueue
	srv.mu.RLock()
	if srv.dialQueues != nil {
		queue = srv.dialQueues[engineName]
	}
	srv.mu.RUnlock()

	if queue == nil {
		var err error
		queue, err = srv.createQueue(engineName)
		if err != nil {
			return nil, err
		}
	}

	if srv.signel != nil {
		srv.signel(engineName) // send connect to engine node.
	}

	timer := time.NewTimer(srv.timeout)
	for {
		select {
		case conn, ok := <-queue.c:
			if !ok {
				timer.Stop()
				return nil, ErrNotActive
			}

			bs := response_full
			for len(bs) > 0 {
				n, err := conn.c.Write(bs)
				if err != nil {
					srv.logger.Info("send response fail",
						log.String("engine_name", engineName),
						log.String("remote_addr", conn.addr),
						log.Error(err))
					continue
				}
				bs = bs[n:]
			}

			return conn.c, nil
		case <-timer.C:
			return nil, ErrTimeout
		}
	}
}

// ServeHTTP implements an http.Handler that answers Connect requests.
func (srv *TunnelServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "405 must CONNECT\n")
		return
	}
	if 1 == atomic.LoadInt32(&srv.closed) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "Closed\n")
		return
	}
	queryParams := req.URL.Query()
	engineName := queryParams.Get("engine_name")
	if "" == engineName {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "engine_name is missing.\n")

		srv.logger.Info("engine_name is missing",
			log.String("engine_name", engineName),
			log.String("remote_addr", req.RemoteAddr))
		return
	}

	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, err.Error())

		srv.logger.Info("hijacking error",
			log.String("engine_name", engineName),
			log.String("remote_addr", req.RemoteAddr),
			log.Error(err))
		return
	}

	queue, err := srv.createQueue(engineName)
	if err != nil {
		srv.logger.Info("create queue error",
			log.String("engine_name", engineName),
			log.String("remote_addr", req.RemoteAddr),
			log.Error(err))

		io.WriteString(conn, "HTTP/1.0 500 "+err.Error()+"\n\n")
		conn.Close()
		return
	}
	if queue == nil {
		srv.logger.Info("create queue fail and error is nil",
			log.String("engine_name", engineName),
			log.String("remote_addr", req.RemoteAddr))

		io.WriteString(conn, "HTTP/1.0 504 Gateway Timeout\n\n")
		conn.Close()
		return
	}

	//log.Println("accept connection -", engineName)
	timer := time.NewTimer(2 * time.Second)
	select {
	case queue.c <- waitingConn{
		c:    conn,
		addr: req.RemoteAddr,
	}:
		timer.Stop()
		return
	case <-timer.C:
		srv.logger.Info("queue timeout",
			log.String("engine_name", engineName),
			log.String("remote_addr", req.RemoteAddr))

		io.WriteString(conn, "HTTP/1.0 504 Gateway Timeout\n\n")
		conn.Close()
		return
	}
}

func (srv *TunnelServer) Close() error {
	if atomic.CompareAndSwapInt32(&srv.closed, 0, 1) {
		srv.mu.Lock()
		for _, queue := range srv.dialQueues {
			queue.Close()
		}
		srv.mu.Unlock()
	}
	return nil
}

func NewTunnelServer(logger log.Logger, maxThreads uint32, timeout time.Duration, signel func(name string)) (*TunnelServer, error) {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &TunnelServer{
		logger:     logger,
		maxThreads: maxThreads,
		timeout:    timeout,
		// c:      make(chan string),
		signel: signel}, nil
}

func GetTimeout(r *http.Request, value time.Duration) time.Duration {
	s := r.URL.Query().Get("timeout")
	if "" == s {
		return value
	}
	t, e := time.ParseDuration(s)
	if nil != e {
		return value
	}
	return t
}
