package services

import (
	"fmt"
	"sync"

	"github.com/runner-mei/log"
)

type FailCounter interface {
	Users() []string
	Fail(username string)
	Count(username string) int
	Zero(username string)
}

type memFailCounter struct {
	lock  sync.Mutex
	users map[string]int
}

func (mem *memFailCounter) Zero(username string) {
	mem.lock.Lock()
	defer mem.lock.Unlock()
	delete(mem.users, username)
}

func (mem *memFailCounter) Fail(username string) {
	mem.lock.Lock()
	defer mem.lock.Unlock()
	count := mem.users[username]
	count++
	mem.users[username] = count
}

func (mem *memFailCounter) Count(username string) int {
	mem.lock.Lock()
	defer mem.lock.Unlock()
	return mem.users[username]
}

func (mem *memFailCounter) Users() []string {
	mem.lock.Lock()
	defer mem.lock.Unlock()
	users := make([]string, 0, len(mem.users))
	for k := range mem.users {
		users = append(users, k)
	}
	return users
}

var CreateFailCounter = func() FailCounter {
	return &memFailCounter{users: map[string]int{}}
}

func ErrorCountCheck(um UserManager, counter FailCounter, maxLoginFailCount int) AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		if maxLoginFailCount <= 0 {
			maxLoginFailCount = 3
		}

		auth.OnBeforeLoad(AuthFunc(func(ctx *AuthContext) error {
			errCount := counter.Count(ctx.Request.Username)
			ctx.ErrorCount = errCount

			fmt.Println("=====", ctx.Request.Username, errCount, maxLoginFailCount)
			if errCount >= maxLoginFailCount {

				if err := um.Lock(ctx); err != nil {
					ctx.Logger.Error("出错次数太多，锁住用户失败", log.Error(err))
				} else {
					counter.Zero(ctx.Request.Username)
				}

				return ErrUserErrorCountExceedLimit
			}
			return nil
		}))

		auth.OnAfterAuth(func(ctx *AuthContext) error {
			fmt.Println("-----", ctx.Request.Username, ctx.Response.IsOK)
			if ctx.Response.IsOK {
				counter.Zero(ctx.Request.Username)
			} else {
				counter.Fail(ctx.Request.Username)

				errCount := counter.Count(ctx.Request.Username)
				ctx.ErrorCount = errCount
				if errCount >= maxLoginFailCount {
					if err := um.Lock(ctx); err != nil {
						ctx.Logger.Error("出错次数太多，锁信用户失败", log.Error(err))
					} else {
						counter.Zero(ctx.Request.Username)
					}
				}
			}
			return nil
		})
		return nil
	})
}
