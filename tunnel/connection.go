package tunnel

import (
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
)

// 常见错误
var (
	ErrTimeout          = errors.ErrTimeout
	ErrEngineDuplicated = errors.BadArgumentWithMessage("'engine_id' is duplicated")
	ErrEngineEmpty      = errors.NewError(http.StatusInternalServerError, "node is empty")
	ErrOverflow         = errors.NewError(http.StatusInternalServerError, "queue is overflow")
	ErrNotActive        = errors.NewError(http.StatusInternalServerError, "engine node isn't alive")
	ErrEngineNotFound   = errors.New("engine isn't found")
	ErrOriginNull       = errors.New("null origin")
)

// 因为读http响应需要 bufio.Reader 对象，可是这个对象读 http 响应时可能会多读取后
// 续的数据，导致后续操作不能正常进行，所以我在 http 响应后每加上一些无用的定长数
// 据，并确保这个定长的数据大于 bufio.Reader 中绶存块的长度，这样在读取完 http 响
// 应后，只要再读取除 bufio.Reader 中绶存块中之外的数据就好了。
var (
	response_connected = "200 Connected to Hengwei proxy"
	response_body      = "Welcome hengwei proxy ......"
	response_full      = []byte("HTTP/1.0 " + response_connected + "\n\n" + response_body)
)

func init() {
	response_length := len([]byte(response_body))
	if response_length < 16 { // 16 is bufio.minReadBufferSize
		response_body = response_body + strings.Repeat(" ", 16-response_length) + "\n"
		response_full = []byte("HTTP/1.0 " + response_connected + "\n\n" + response_body)
	}
}

// TunnelAddr represents a network end point address.
type TunnelAddr struct {
	network string
	address string
	path    string
}

func (addr *TunnelAddr) Network() string {
	return "tunnel"
}

func (addr *TunnelAddr) String() string {
	return "tunnel:" + addr.network + "://" + addr.address + addr.path
}

type TunnelListener struct {
	logger log.Logger
	closed int32

	maxThreads uint32
	c          chan net.Conn
	addr       TunnelAddr
}

func Listen(logger log.Logger, threads int, network, address, path string) (*TunnelListener, error) {
	if 0 == threads {
		threads = 10
	}
	if threads > 100 {
		threads = 100
	}
	if address == "" {
		return nil, errors.New("address is missing")
	}
	if path == "" {
		return nil, errors.New("path is missing")
	}
	if network == "" {
		network = "tcp"
	}

	srv := &TunnelListener{
		logger: logger,
		c:      make(chan net.Conn),
		addr: TunnelAddr{
			network: network,
			address: address,
			path:    path,
		},
	}

	for i := 0; i < threads; i++ {
		go srv.run(i)
	}

	return srv, nil
}

// Accept waits for and returns the next connection to the listener.
func (listener *TunnelListener) Accept() (net.Conn, error) {
	if 1 == atomic.LoadInt32(&listener.closed) {
		return nil, &net.OpError{
			Op:   "dial-http",
			Net:  listener.addr.Network(),
			Addr: &listener.addr,
			Err:  errors.New("listener is closed"),
		}
	}
	conn, ok := <-listener.c
	if ok {
		listener.logger.Info("accept from '" + listener.addr.String() + "'.")
		return conn, nil
	}
	if 1 == atomic.LoadInt32(&listener.closed) {
		return nil, &net.OpError{
			Op:   "dial-http",
			Net:  listener.addr.Network(),
			Addr: &listener.addr,
			Err:  errors.New("listener is closed"),
		}
	} else {
		return nil, &net.OpError{
			Op:   "dial-http",
			Net:  listener.addr.Network(),
			Addr: &listener.addr,
			Err:  temporaryError("listener is closed"),
		}
	}
}

func (listener *TunnelListener) run(idx int) {
	errorCount := 0
	for 0 == atomic.LoadInt32(&listener.closed) {
		if e := listener.connect(); e != nil {
			errorCount++
			if errorCount%20 < 3 {
				if atomic.LoadInt32(&listener.closed) != 0 {
					break
				}
				listener.logger.Info("connect to '"+listener.addr.String()+"'", log.Error(e))
			}
			time.Sleep(1 * time.Second)
		} else {
			errorCount = 0
		}
	}
}

func (listener *TunnelListener) connect() error {
	conn, err := net.Dial(listener.addr.network, listener.addr.address)
	if err != nil {
		return err
	}
	io.WriteString(conn, "CONNECT "+listener.addr.path+" HTTP/1.0\n\n")

	// Require successful HTTP response
	// before switching to RPC protocol.
	//
	// 注意缓冲的问题。
	reader := bufio.NewReaderSize(conn, 16) // 16 is bufio.minReadBufferSize
	resp, err := http.ReadResponse(reader, &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == response_connected {
		limitReader := io.LimitReader(conn, int64(len(response_body)-reader.Buffered()))
		if _, err := io.Copy(ioutil.Discard, limitReader); nil != err {
			conn.Close()
			return err
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case listener.c <- conn:
			timer.Stop()
			return nil
		case <-timer.C:
			conn.Close()
			return ErrTimeout
		}
	}
	conn.Close()

	if err == nil {
		return errors.New("unexpected HTTP response: " + resp.Status)
	}
	return err
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (listener *TunnelListener) Close() error {
	if atomic.CompareAndSwapInt32(&listener.closed, 0, 1) {
		close(listener.c)
	}
	return nil
}

// Addr returns the listener's network address.
func (listener *TunnelListener) Addr() net.Addr {
	return &listener.addr
}

type temporaryError string

func (e temporaryError) Error() string   { return string(e) }
func (e temporaryError) Timeout() bool   { return false }
func (e temporaryError) Temporary() bool { return true }
