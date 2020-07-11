// +build all_users

package users

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"time"

	"github.com/runner-mei/errors"
	"github.com/runner-mei/goutils/syncx"
	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/users/usermodels"
)

type groupEntries struct {
	logger           log.Logger
	byID             map[int64]Usergroup
	byName           map[string]Usergroup
	allUsergroups    []Usergroup
	enableUsergroups []Usergroup
	timestamp        time.Time
}

func (entries *groupEntries) isTimeout(timeout time.Duration) bool {
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return time.Now().Sub(entries.timestamp) > timeout
}

type UsergroupCache struct {
	timeout      time.Duration
	usergroupDao usermodels.UsergroupQueryer

	lastErr       syncx.ErrorValue
	allUsergroups atomic.Value
	lock          sync.Mutex
	isLoading     int64
}

func (cache *UsergroupCache) getUserEntries() *groupEntries {
	o := cache.allUsergroups.Load()
	if o == nil {
		return nil
	}
	u, _ := o.(*groupEntries)
	return u
}

func (cache *UsergroupCache) setUserEntries(u *groupEntries) {
	cache.allUsergroups.Store(u)
}

func (cache *UsergroupCache) read() (*groupEntries, error) {
	u := cache.getUserEntries()
	if u != nil {
		if u.isTimeout(cache.timeout) && atomic.CompareAndSwapInt64(&cache.isLoading, 0, 1) {
			go func() {
				defer atomic.StoreInt64(&cache.isLoading, 0)
				_, err := cache.load(false)
				if err != nil {
					cache.lastErr.Set(err)
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

func (cache *UsergroupCache) loadUsergroups(ctx context.Context) ([]Usergroup, error) {
	next, closer := cache.usergroupDao.GetUsergroups(context.Background())
	defer util.CloseWith(closer)

	var allList = make([]Usergroup, 0, 64)
	for {
		u := &usergroup{}
		ok, err := next(&u.ug)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, errors.Wrap(err, "read usergroups fail")
		}

		if !ok {
			break
		}
		allList = append(allList, u)
	}

	unext, ucloser := cache.usergroupDao.GetUserAndGroupList(context.Background())
	defer util.CloseWith(ucloser)
	for {
		uu := usermodels.UserAndUserGroup{}
		ok, err := unext(&uu)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, errors.Wrap(err, "read userroles fail")
		}

		if !ok {
			break
		}

		for i := range allList {
			if allList[i].ID() == uu.GroupID {
				allList[i].(*usergroup).userids = append(allList[i].(*usergroup).userids, uu.UserID)
				break
			}
		}
	}
	return allList, nil
}

func (cache *UsergroupCache) load(checkCached bool) (*groupEntries, error) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if checkCached {
		u := cache.getUserEntries()
		if u != nil {
			return u, nil
		}
	}

	allList, err := cache.loadUsergroups(context.Background())
	if err != nil {
		return nil, err
	}

	enabledList := make([]Usergroup, 0, len(allList))
	byID := make(map[int64]Usergroup, len(allList))
	byName := make(map[string]Usergroup, len(allList))
	for idx := range allList {
		ug := allList[idx]

		byID[ug.ID()] = ug
		byName[ug.Name()] = ug
		if !ug.(*usergroup).IsDisabled() {
			enabledList = append(enabledList, ug)
		}
	}

	entries := &groupEntries{
		byID:             byID,
		byName:           byName,
		allUsergroups:    allList,
		enableUsergroups: enabledList,
		timestamp:        time.Now(),
	}
	cache.setUserEntries(entries)
	return entries, nil
}

func (cache *UsergroupCache) Usergroups(ctx context.Context, opts ...Option) ([]Usergroup, error) {
	if e := cache.lastErr.Get(); e != nil {
		return nil, e
	}
	entries, err := cache.read()
	if err != nil {
		return nil, err
	}
	var options = api.InternalApply(opts)
	if !options.IncludeDisabled {
		return entries.enableUsergroups, nil
	}
	return entries.allUsergroups, nil
}

func (cache *UsergroupCache) UsergroupsByUser(ctx context.Context, userID int64, opts ...Option) ([]Usergroup, error) {
	if e := cache.lastErr.Get(); e != nil {
		return nil, e
	}

	entries, err := cache.read()
	if err != nil {
		return nil, err
	}

	var options = api.InternalApply(opts)

	var groups []Usergroup
	for idx := range entries.allUsergroups {
		ok := entries.allUsergroups[idx].HasUser(ctx, userID)
		if !ok {
			continue
		}
		if !options.IncludeDisabled && entries.allUsergroups[idx].(*usergroup).IsDisabled() {
			continue
		}
		groups = append(groups, entries.allUsergroups[idx])
	}
	return groups, nil
}

func (cache *UsergroupCache) UsergroupByName(ctx context.Context, usergroupname string, opts ...Option) (Usergroup, error) {
	if e := cache.lastErr.Get(); e != nil {
		return nil, e
	}

	entries, err := cache.read()
	if err != nil {
		return nil, err
	}
	u := entries.byName[usergroupname]
	if u == nil {
		return nil, ErrUsergroupnameNotFound(usergroupname)
	}

	var options = api.InternalApply(opts)
	if options.IncludeDisabled {
		return u, nil
	}

	if u.(*usergroup).IsDisabled() {
		return nil, ErrUsergroupDisabled(usergroupname)
	}
	return u, nil
}

func (cache *UsergroupCache) UsergroupByID(ctx context.Context, usergroupID int64, opts ...Option) (Usergroup, error) {
	if e := cache.lastErr.Get(); e != nil {
		return nil, e
	}

	entries, err := cache.read()
	if err != nil {
		return nil, err
	}
	u := entries.byID[usergroupID]
	if u == nil {
		return nil, ErrUsergroupIDNotFound(usergroupID)
	}

	var options = api.InternalApply(opts)
	if options.IncludeDisabled {
		return u, nil
	}

	if u.(*usergroup).IsDisabled() {
		return nil, ErrUsergroupDisabled(u.Name())
	}
	return u, nil
}
