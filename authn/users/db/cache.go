package dbusers

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/runner-mei/log"
	"github.com/runner-mei/moo/api"
)

type userEntries struct {
	byID        map[int64]api.User
	byName      map[string]api.User
	allUsers    []api.User
	enableUsers []api.User
	timestamp   time.Time
}

func (entries *userEntries) isTimeout(timeout time.Duration) bool {
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return time.Now().Sub(entries.timestamp) > timeout
}

type UserCache struct {
	logger    log.Logger
	timeout   time.Duration
	loadUsers func(context.Context) ([]api.User, error)

	allUsers  atomic.Value
	lock      sync.Mutex
	isLoading int64
}

func (cache *UserCache) getUserEntries() *userEntries {
	o := cache.allUsers.Load()
	if o == nil {
		return nil
	}
	u, _ := o.(*userEntries)
	return u
}

func (cache *UserCache) setUserEntries(u *userEntries) {
	cache.allUsers.Store(u)
}

func (cache *UserCache) read() (*userEntries, error) {
	u := cache.getUserEntries()
	if u != nil {
		if u.isTimeout(cache.timeout) && atomic.CompareAndSwapInt64(&cache.isLoading, 0, 1) {
			go func() {
				defer atomic.StoreInt64(&cache.isLoading, 0)
				_, err := cache.load(false)
				if err != nil {
					cache.logger.Error("load users fail", log.Error(err))
				}
			}()
		}
		return u, nil
	}
	if atomic.CompareAndSwapInt64(&cache.isLoading, 0, 1) {
		defer atomic.StoreInt64(&cache.isLoading, 0)

		return cache.load(false)
	}
	return cache.load(true)
}

func (cache *UserCache) load(checkCached bool) (*userEntries, error) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if checkCached {
		u := cache.getUserEntries()
		if u != nil {
			return u, nil
		}
	}

	allList, err := cache.loadUsers(context.Background())
	if err != nil {
		return nil, err
	}

	enabledList := make([]api.User, 0, len(allList))
	byID := make(map[int64]api.User, len(allList))
	byName := make(map[string]api.User, len(allList))
	for idx := range allList {
		u := allList[idx].(*user)

		if !u.u.IsDisabled() {
			enabledList = append(enabledList, u)
		}

		byID[u.ID()] = u
		byName[u.Name()] = u
	}

	entries := &userEntries{
		byID:        byID,
		byName:      byName,
		allUsers:    allList,
		enableUsers: enabledList,
		timestamp:   time.Now(),
	}
	cache.setUserEntries(entries)
	return entries, nil
}

func (cache *UserCache) Users(ctx context.Context, opts ...api.Option) ([]api.User, error) {
	entries, err := cache.read()
	if err != nil {
		return nil, err
	}
	var options = api.InternalApply(opts...)
	if options.IncludeDisabled {
		return entries.allUsers, nil
	}
	return entries.enableUsers, nil
}

func (cache *UserCache) UserByName(ctx context.Context, username string, opts ...api.Option) (api.User, error) {
	entries, err := cache.read()
	if err != nil {
		return nil, err
	}
	u := entries.byName[username]
	if u == nil {
		return nil, ErrUsernameNotFound(username)
	}

	var options = api.InternalApply(opts...)
	if options.IncludeDisabled {
		return u, nil
	}

	if u.(*user).IsDisabled() {
		return nil, ErrUserDisabled(username)
	}
	return u, nil
}

func (cache *UserCache) UserByID(ctx context.Context, userID int64, opts ...api.Option) (api.User, error) {
	entries, err := cache.read()
	if err != nil {
		return nil, err
	}
	u := entries.byID[userID]
	if u == nil {
		return nil, ErrUserIDNotFound(userID)
	}

	var options = api.InternalApply(opts...)
	if options.IncludeDisabled {
		return u, nil
	}

	if u.(*user).IsDisabled() {
		return nil, ErrUserDisabled(u.Name())
	}
	return u, nil
}
