package auth

import (
	"context"
	"errors"

	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/moo/auth/services"
)

type ErrOnline struct {
	OnlineList []SessionInfo
}

func (err *ErrOnline) Error() string {
	if len(err.OnlineList) == 1 {
		return "用户已在 " + err.OnlineList[0].Address +
			" 上登录，最后一次活动时间为 " +
			err.OnlineList[0].UpdatedAt.Format("2006-01-02 15:04:05Z07:00")

	}
	return "用户已在其他机器上登录"
}

func IsOnlinedError(err error) ([]SessionInfo, bool) {
	for err != nil {
		oe, ok := err.(*ErrOnline)
		if ok {
			return oe.OnlineList, true
		}
		err = errors.Unwrap(err)
	}
	return nil, false
}

type SessionInfo struct {
	UUID      string
	UserID    interface{}
	Username  string
	Address   string
	CreatedAt util.UnixTime
	UpdatedAt util.UnixTime
}

type Sessions interface {
	Login(ctx context.Context, userid interface{}, username, address string) (string, error)
	Logout(ctx context.Context, key string) error

	services.OnlineChecker
	Get(ctx context.Context, id string) (*SessionInfo, error)
	All(ctx context.Context) ([]SessionInfo, error)
	UpdateNow(ctx context.Context, key string) error
}

type SessionsForTest interface {
	Sessions
	
	Count(ctx context.Context, username string, address string) (int, error)
}
