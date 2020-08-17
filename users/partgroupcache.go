// +build !all_users

package users

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/runner-mei/goutils/util"
	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/users/usermodels"
)

type usergroupCacheItem struct {
	Object     Usergroup
	Expiration int64
}

// Expired returns true if the item has expired.
func (item usergroupCacheItem) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

type UsergroupCacheBase struct {
	defaultExpiration time.Duration
	items             map[int64]usergroupCacheItem
	mu                sync.RWMutex
	onEvicted         func(int64, Usergroup)
}

// Set add an item to the cache, replacing any existing item. If the duration is 0
// (DefaultExpiration), the cache's default expiration time is used. If it is -1
// (NoExpiration), the item never expires.
func (c *UsergroupCacheBase) Set(k int64, x Usergroup, d time.Duration) {
	// "Inlining" of set
	var e int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.mu.Lock()
	c.items[k] = usergroupCacheItem{
		Object:     x,
		Expiration: e,
	}
	// TODO: Calls to mu.Unlock are currently not deferred because defer
	// adds ~200 ns (as of go1.)
	c.mu.Unlock()
}

// SetDefault add an item to the cache, replacing any existing item, using the default
// expiration.
func (c *UsergroupCacheBase) SetDefault(k int64, x Usergroup) {
	c.Set(k, x, DefaultExpiration)
}

func (c *UsergroupCacheBase) set(k int64, x Usergroup, d time.Duration) {
	var e int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.items[k] = usergroupCacheItem{
		Object:     x,
		Expiration: e,
	}
}

// Add an item to the cache only if an item doesn't already exist for the given
// key, or if the existing item has expired. Returns an error otherwise.
func (c *UsergroupCacheBase) Add(k int64, x Usergroup, d time.Duration) error {
	c.mu.Lock()
	found := c.isExists(k)
	if found {
		c.mu.Unlock()
		return fmt.Errorf("Item %d already exists", k)
	}
	c.set(k, x, d)
	c.mu.Unlock()
	return nil
}

// Replace a new value for the cache key only if it already exists, and the existing
// item hasn't expired. Returns an error otherwise.
func (c *UsergroupCacheBase) Replace(k int64, x Usergroup, d time.Duration) error {
	c.mu.Lock()
	found := c.isExists(k)
	if !found {
		c.mu.Unlock()
		return fmt.Errorf("Item %d doesn't exist", k)
	}
	c.set(k, x, d)
	c.mu.Unlock()
	return nil
}

func (c *UsergroupCacheBase) isExists(k int64) bool {
	item, found := c.items[k]
	if !found {
		return false
	}
	// "Inlining" of Expired
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return false
		}
	}
	return true
}

// Get an item from the cache. Returns the item or nil, and a bool indicating
// whether the key was found.
func (c *UsergroupCacheBase) Get(k int64) (Usergroup, bool) {
	c.mu.RLock()
	// "Inlining" of get and Expired
	item, found := c.items[k]
	if !found {
		c.mu.RUnlock()
		return nil, false
	}
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			c.mu.RUnlock()
			return nil, false
		}
	}
	c.mu.RUnlock()
	return item.Object, true
}

// GetWithExpiration returns an item and its expiration time from the cache.
// It returns the item or nil, the expiration time if one is set (if the item
// never expires a zero value for time.Time is returned), and a bool indicating
// whether the key was found.
func (c *UsergroupCacheBase) GetWithExpiration(k int64) (Usergroup, time.Time, bool) {
	ov, expiration, ok := c.GetWithExpirationNano(k)
	if expiration == 0 {
		return ov, time.Time{}, ok
	}
	return ov, time.Unix(0, expiration), ok
}

func (c *UsergroupCacheBase) GetWithExpirationNano(k int64) (Usergroup, int64, bool) {
	c.mu.RLock()
	// "Inlining" of get and Expired
	item, found := c.items[k]
	if !found {
		c.mu.RUnlock()
		return nil, 0, false
	}

	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			c.mu.RUnlock()
			return nil, 0, false
		}

		// Return the item and the expiration time
		c.mu.RUnlock()
		return item.Object, item.Expiration, true
	}

	// If expiration <= 0 (i.e. no expiration time set) then return the item
	// and a zeroed time.Time
	c.mu.RUnlock()
	return item.Object, 0, true
}

// Delete an item from the cache. Does nothing if the key is not in the cache.
func (c *UsergroupCacheBase) Delete(k int64) {
	c.mu.Lock()
	v, evicted := c.delete(k)
	c.mu.Unlock()
	if evicted {
		c.onEvicted(k, v)
	}
}

func (c *UsergroupCacheBase) delete(k int64) (Usergroup, bool) {
	if c.onEvicted != nil {
		if v, found := c.items[k]; found {
			delete(c.items, k)
			return v.Object, true
		}
	}
	delete(c.items, k)
	return nil, false
}

// DeleteExpired delete all expired items from the cache.
func (c *UsergroupCacheBase) DeleteExpired() {
	var keys []int64
	var items []Usergroup
	now := time.Now().UnixNano()

	c.mu.Lock()
	if len(c.items) == 0 {
		c.mu.Unlock()
		return
	}

	keys = make([]int64, 0, len(c.items))
	items = make([]Usergroup, 0, len(c.items))

	for k, v := range c.items {
		// "Inlining" of expired
		if v.Expiration > 0 && now > v.Expiration {
			ov, evicted := c.delete(k)
			if evicted {
				keys = append(keys, k)
				items = append(items, ov)
			}
		}
	}
	c.mu.Unlock()

	for idx, v := range items {
		c.onEvicted(keys[idx], v)
	}
}

// OnEvicted sets an (optional) function that is called with the key and value when an
// item is evicted from the cache. (Including when it is deleted manually, but
// not when it is overwritten.) Set to nil to disable.
func (c *UsergroupCacheBase) OnEvicted(f func(int64, Usergroup)) {
	c.mu.Lock()
	c.onEvicted = f
	c.mu.Unlock()
}

// ForEach copies all unexpired items in the cache into a new map and returns it.
func (c *UsergroupCacheBase) ForEach(f func(int64, time.Time, Usergroup)) {
	var keys []int64
	var items []usergroupCacheItem
	now := time.Now().UnixNano()

	c.mu.RLock()
	if len(c.items) == 0 {
		c.mu.RUnlock()
		return
	}
	keys = make([]int64, 0, len(c.items))
	items = make([]usergroupCacheItem, 0, len(c.items))
	for k, v := range c.items {
		// "Inlining" of Expired
		if v.Expiration > 0 {
			if now > v.Expiration {
				continue
			}
		}
		keys = append(keys, k)
		items = append(items, v)
	}
	c.mu.RUnlock()

	for idx, item := range items {
		f(keys[idx], time.Unix(0, item.Expiration), item.Object)
	}
}

// ItemCount returns the number of items in the cache. This may include items that have
// expired, but have not yet been cleaned up.
func (c *UsergroupCacheBase) ItemCount() int {
	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}

// Flush delete all items from the cache.
func (c *UsergroupCacheBase) Flush() {
	c.mu.Lock()
	c.items = map[int64]usergroupCacheItem{}
	c.mu.Unlock()
}

type UsergroupCache struct {
	UsergroupCacheBase
	userManager api.UserManager
	queryer     usermodels.UsergroupQueryer

	nmu     sync.RWMutex
	name2id map[string]int64
}

func (c *UsergroupCache) UsergroupsByUserID(ctx context.Context, userID int64, opts ...Option) ([]Usergroup, error) {
	next, closer := c.queryer.GetUserAndGroupList(ctx, sql.NullInt64{Valid: true, Int64: userID}, false)
	defer util.CloseWith(closer)

	var groups = make([]Usergroup, 0, 8)
	for {
		var u2g usermodels.UserAndUsergroup
		ok, err := next(&u2g)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}

		group, err := c.usergroupByID(ctx, u2g.GroupID, false, opts, c.loadByID)
		if err != nil {
			if IsUsergroupDisabled(err) {
				continue
			}
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (c *UsergroupCache) UsergroupByName(ctx context.Context, usergroupname string, opts ...Option) (Usergroup, error) {
	var id int64
	c.nmu.RLock()
	if c.name2id != nil {
		id = c.name2id[usergroupname]
	}
	c.nmu.RUnlock()

	if id != 0 {
		u, err := c.usergroupByID(ctx, id, false, opts, c.loadByID)
		if err != nil {
			return nil, err
		}
		if u.Name() == usergroupname {
			return u, nil
		}
	}

	var group usergroup
	err := c.queryer.GetUsergroupByName(ctx, usergroupname)(&group.ug)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUsergroupnameNotFound(usergroupname)
		}
		return nil, err
	}

	id = group.ug.ID
	c.nmu.Lock()
	if c.name2id == nil {
		c.name2id = map[string]int64{}
	}
	c.name2id[usergroupname] = id
	c.nmu.Unlock()

	return c.usergroupByID(ctx, id, true, opts, func(ctx context.Context, id int64) (Usergroup, error) {
		return c.load(ctx, &group)
	})
}

func (c *UsergroupCache) UsergroupByID(ctx context.Context, id int64, opts ...Option) (Usergroup, error) {
	return c.usergroupByID(ctx, id, false, opts, c.loadByID)
}

func (c *UsergroupCache) loadByID(ctx context.Context, id int64) (Usergroup, error) {
	var group usergroup
	err := c.queryer.GetUsergroupByID(ctx, id)(&group.ug)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUsergroupIDNotFound(id)
		}
		return nil, err
	}
	return c.load(ctx, &group)
}

func (c *UsergroupCache) load(ctx context.Context, group *usergroup) (Usergroup, error) {
	userids, err := c.queryer.GetUserIDsByGroupIDs(ctx, []int64{group.ug.ID}, false, sql.NullBool{})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	group.userManager = c.userManager
	group.usergroupManager = c
	group.userids = userids
	return group, nil
}

func (c *UsergroupCache) usergroupByID(ctx context.Context, id int64, forceUpdate bool, opts []Option, read func(ctx context.Context, id int64) (Usergroup, error)) (Usergroup, error) {
	u, ok := c.Get(id)
	if ok {
		var err error
		u, err = read(ctx, id)
		if err != nil {
			return nil, err
		}
		c.SetDefault(id, u)
	}

	var options = api.InternalApply(opts...)
	if options.GroupIncludeDisabled {
		return u, nil
	}

	if u.(*usergroup).IsDisabled() {
		return nil, ErrUsergroupDisabled(u.Name())
	}
	return u, nil
}
