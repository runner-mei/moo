package inmem

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/moo/authn"
	"github.com/runner-mei/moo"
	"go.uber.org/fx"
)


func init() {
	moo.On(func(*moo.Environment) moo.Option {
		return fx.Provide(func(lifecycle fx.Lifecycle, env *moo.Environment) (authn.Sessions, authn.SessionsForTest) {
			mgr := &SessionManager{
				expiresNano: env.Config.Int64WithDefault("sessions.inmem.expires", 0) * int64(time.Second),
				list: map[string]*authn.SessionInfo{},
			}

			var timer util.Timer

			lifecycle.Append(fx.Hook{
				OnStart: func(context.Context) error {
					timer.Start(env.Config.DurationWithDefault("sessions.inmem.check_interval", 60 * time.Second), 
					func() bool {
						mgr.DeleteExpired(context.Background())
						return true
					})
					return nil
				},
				OnStop: func(context.Context) error {
					timer.Stop()
					return nil
				},	
			})
			return mgr, mgr
		})
	})
}

type SessionManager struct {
	expiresNano int64
	mu   sync.RWMutex
	list map[string]*authn.SessionInfo
}

func (mgr *SessionManager) Count(ctx context.Context, username string, address string) (int, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	var count = 0

	filter := func(si *authn.SessionInfo) bool {
		return true
	}
	if username != "" {
		if address != "" {
			filter = func(si *authn.SessionInfo) bool {
				return si.Username == username && si.Address == address
			}
		} else {
			filter = func(si *authn.SessionInfo) bool {
				return si.Username == username
			}
		}
	} else if address != "" {
		filter = func(si *authn.SessionInfo) bool {
			return si.Address == address
		}
	}
	for _, s := range mgr.list {
		if filter(s) {
			count ++
		}
	}
	return count, nil
}

func (mgr *SessionManager) UpdateNow(ctx context.Context, id string) error {
	mgr.mu.RLock()
	s := mgr.list[id]
	mgr.mu.RUnlock()

	s.UpdatedAt.AtomicSet(time.Now())
	return nil
}

func (mgr *SessionManager) Get(ctx context.Context, id string) (*authn.SessionInfo, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.list[id], nil
}

func (mgr *SessionManager) All(ctx context.Context) ([]authn.SessionInfo, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	var results = make([]authn.SessionInfo, 0, len(mgr.list))

	for _, s := range mgr.list {
		results = append(results, *s)
	}
	return results, nil
}

func (mgr *SessionManager) Login(ctx context.Context, userid interface{}, username, loginAddress string) (string, error) {
	if userid == nil {
		return "", errors.New("userid is missing")
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	var old *authn.SessionInfo

	for _, s := range mgr.list {
		if s.Username == username && s.Address == loginAddress {
			old = s
			break
		}
	}
	if old != nil {
		return old.UUID, nil
	}

	uuid := authn.GenerateID()
	mgr.list[uuid] = &authn.SessionInfo{
		UUID:      uuid,
		UserID:    userid,
		Username:  username,
		Address:   loginAddress,
		CreatedAt: util.ToUnixTime(time.Now()),
		UpdatedAt: util.ToUnixTime(time.Now()),
	}
	return uuid, nil
}

func (mgr *SessionManager) Logout(ctx context.Context, id string) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	delete(mgr.list, id)
	return nil
}

func (mgr *SessionManager) IsOnlineExists(ctx context.Context, userid interface{}, username, loginAddress string) error {
	// 判断用户是不是已经在其它主机上登录
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	var onlineList = make([]authn.SessionInfo, 0, 4)

	for _, s := range mgr.list {
		if s.Username == username {
			if s.Address == loginAddress {
				return nil
			}
			onlineList = append(onlineList, *s)
		}
	}

	if len(onlineList) > 0 {
		return &authn.ErrOnline{OnlineList: onlineList}
	}
	return nil
}

func (mgr *SessionManager) DeleteExpired(ctx context.Context) error {
	if mgr.expiresNano <= 0 {
		return nil
	}

	idlist := func() []string {
		now := time.Now().UnixNano()

		mgr.mu.RLock()
		defer mgr.mu.RUnlock()

		var idlist []string
		for id, s := range mgr.list {
			if (now - s.UpdatedAt.UnixNano()) > mgr.expiresNano {
				idlist = append(idlist, id)
			}
		}
		return idlist
	}()

	if len(idlist) == 0 {
		return nil
	}

	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	for _, id := range idlist {
		delete(mgr.list, id)
	}
	return nil
}
