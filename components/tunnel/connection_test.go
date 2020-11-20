package tunnel

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	_ "net/http/pprof"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/runner-mei/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestConnectionSimple(t *testing.T) {
	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	l, err := logConfig.Build()
	if err != nil {
		t.Error(err)
		return
	}
	logger := log.NewLogger(l)

	var tunnelSrv *TunnelServer

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/stub/listener":
			tunnelSrv.ServeHTTP(w, req)
		default:
			http.DefaultServeMux.ServeHTTP(w, req)
		}
	})
	hsrv := httptest.NewServer(handler)
	defer hsrv.Close()

	//context := "this is test."
	tunnelSrv, err = NewTunnelServer(logger, 1, 0, nil)
	if nil != err {
		t.Error(err)
		return
	}
	defer tunnelSrv.Close()

	var listener, e = Listen(logger,
		1,
		"tcp",
		strings.TrimPrefix(hsrv.URL, "http://"),
		"/stub/listener?engine_name=test")
	if nil != e {
		t.Error(e)
		return
	}

	// go http.ListenAndServe(":", nil)
	var wait sync.WaitGroup
	defer wait.Wait()

	defer listener.Close()

	wait.Add(1)
	go func() {
		defer wait.Done()
		conn, e := listener.Accept()
		if nil != e {
			t.Error(e)
			return
		}

		buf := make([]byte, 1024)
		for {
			if n, e := conn.Read(buf); nil != e {
				if io.EOF != e {
					t.Error("[server] read", e)
				}
				break
			} else if _, e := conn.Write(buf[:n]); nil != e {
				if io.EOF != e {
					t.Error("[server] write", e)
				}
				break
			}
		}
	}()

	conn, e := tunnelSrv.Dial("test")
	if nil != e {
		t.Error(e)
		return
	}
	defer conn.Close()

	txt := "1234567890"
	if _, e := io.WriteString(conn, txt); nil != e {
		t.Error(e)
		return
	}
	buf := make([]byte, 1024)
	n, e := conn.Read(buf)
	if nil != e {
		t.Error("[server] read", e)
		return
	}

	if txt != string(buf[:n]) {
		t.Error(string(buf[:n]))
	}

	listener.Close()
	conn.Close()
}

func TestConnectionConnectNotFound(t *testing.T) {
	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	l, err := logConfig.Build()
	if err != nil {
		t.Error(err)
		return
	}
	logger := log.NewLogger(l)

	tunnelSrv, err := NewTunnelServer(logger, 1, 1*time.Second, nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer tunnelSrv.Close()

	conn, err := tunnelSrv.Dial("test")
	if err == nil {
		conn.Close()
		t.Error("excepted is error")
		return
	}
	if !strings.Contains(err.Error(), ErrTimeout.Error()) {
		t.Error(err)
		return
	}
}

func TestConnectionAcceptNotFound(t *testing.T) {

	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	l, err := logConfig.Build()
	if err != nil {
		t.Error(err)
		return
	}
	logger := log.NewLogger(l)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Println(req.RequestURI)
		//w.WriteHeader(http.StatusOK)
		//io.WriteString(w, context)
		http.DefaultServeMux.ServeHTTP(w, req)
	}))
	defer srv.Close()

	server_addr, _ := url.Parse(srv.URL)
	listener, err := Listen(logger, 1, "tcp", server_addr.Host, "/stub/listener?engine_name=test")
	if err != nil {
		t.Error(err)
		return
	}
	defer listener.Close()

	err = listener.connect()
	if err == nil {
		t.Error("excepted is error")
		return
	}
	if !strings.Contains(err.Error(), "404 Not Found") {
		t.Error(err)
		return
	}
}
