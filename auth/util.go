package auth

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

const (
	defaultMemory = 32 << 20 // 32 MB

	HeaderAccept              = "Accept"
	HeaderAcceptEncoding      = "Accept-Encoding"
	HeaderContentType         = "Content-Type"
	HeaderXHTTPMethodOverride = "X-HTTP-Method-Override"
	HeaderXForwardedFor       = "X-Forwarded-For"
	HeaderXRealIP             = "X-Real-IP"

	MIMEApplicationJSON = "application/json"
	MIMEApplicationXML  = "application/xml"
	MIMETextXML         = "text/xml"
	MIMEApplicationForm = "application/x-www-form-urlencoded"
	MIMEMultipartForm   = "multipart/form-data"
)

func ParseTime(layout, s string) time.Time {
	if layout != "" {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t
		}
	}

	for _, layout := range []string{time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999Z07:00"} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

// ReplacePlaceholders 将 sql 语句中的 ? 改成 $x 形式
func ReplacePlaceholders(sql string) string {
	buf := &bytes.Buffer{}
	i := 0
	for {
		p := strings.Index(sql, "?")
		if p == -1 {
			break
		}

		if len(sql[p:]) > 1 && sql[p:p+2] == "??" { // escape ?? => ?
			buf.WriteString(sql[:p])
			buf.WriteString("?")
			if len(sql[p:]) == 1 {
				break
			}
			sql = sql[p+2:]
		} else {
			i++
			buf.WriteString(sql[:p])
			fmt.Fprintf(buf, "$%d", i)
			sql = sql[p+1:]
		}
	}

	buf.WriteString(sql)
	return buf.String()
}

func StringWith(params map[string]interface{}, key, defaultValue string) (string, bool) {
	return stringWith(params, key, defaultValue)
}

func stringWith(params map[string]interface{}, key, defaultValue string) (string, bool) {
	o, ok := params[key]
	if !ok || o == nil {
		return defaultValue, true
	}

	s, ok := o.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultValue, true
	}
	return s, true
}

// NullTime represents a time.Time that may be null. NullTime implements the
// sql.Scanner interface so it can be used as a scan destination, similar to
// sql.NullString.
type NullTime struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	switch s := value.(type) {
	case time.Time:
		nt.Time = s
		nt.Valid = true
		return nil
	case string:
		return nt.Parse(s)
	case []byte:
		return nt.Parse(string(s))
	default:
		return errors.New("unknow value - " + fmt.Sprintf("%T %s", value, value))
	}
}

func (nt NullTime) Parse(s string) error {
	for _, layout := range []string{} {
		t, err := time.Parse(layout, s)
		if err == nil {
			nt.Time = t
			nt.Valid = true
			return nil
		}
	}
	return errors.New("unknow value - " + s)
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

var (
	// objectIDCounter is atomically incremented when generating a new ObjectId
	// using NewObjectId() function. It's used as a counter part of an id.
	objectIDCounter uint32 = 0

	// machineID stores machine id generated once and used in subsequent calls
	// to NewObjectId function.
	machineID  = readMachineID()
	currentPid = os.Getpid()
)

// initMachineId generates machine id and puts it into the machineID global
// variable. If this function fails to get the hostname, it will cause
// a runtime error.
func readMachineID() []byte {
	var sum [3]byte
	id := sum[:]
	hostname, err1 := os.Hostname()
	if err1 != nil {
		_, err2 := io.ReadFull(rand.Reader, id)
		if err2 != nil {
			panic(fmt.Errorf("cannot get hostname: %v; %v", err1, err2))
		}
		return id
	}
	hw := md5.New()
	hw.Write([]byte(hostname))
	copy(id, hw.Sum(nil))
	return id
}

// GenerateID returns a new unique ObjectId.
// This function causes a runtime error if it fails to get the hostname
// of the current machine.
func GenerateID() string {
	var b [12]byte
	// Timestamp, 4 bytes, big endian
	binary.BigEndian.PutUint32(b[:], uint32(time.Now().Unix()))
	// Machine, first 3 bytes of md5(hostname)
	b[4] = machineID[0]
	b[5] = machineID[1]
	b[6] = machineID[2]
	// Pid, 2 bytes, specs don't specify endianness, but we use big endian.
	b[7] = byte(currentPid >> 8)
	b[8] = byte(currentPid)
	// Increment, 3 bytes, big endian
	i := atomic.AddUint32(&objectIDCounter, 1)
	b[9] = byte(i >> 16)
	b[10] = byte(i >> 8)
	b[11] = byte(i)
	return hex.EncodeToString(b[:])
}

func RealIP(req *http.Request) string {
	ra := req.RemoteAddr
	if ip := req.Header.Get(HeaderXForwardedFor); ip != "" {
		ra = ip
	} else if ip := req.Header.Get(HeaderXRealIP); ip != "" {
		ra = ip
	} else {
		ra, _, _ = net.SplitHostPort(ra)
	}
	return ra
}

func IsConsumeJSON(r *http.Request) bool {
	accept := r.Header.Get(HeaderAccept)
	contentType := r.Header.Get(HeaderContentType)
	return strings.Contains(contentType, MIMEApplicationJSON) &&
		strings.Contains(accept, MIMEApplicationJSON)
}
