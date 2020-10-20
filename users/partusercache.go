// +build !all_users

package users

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/runner-mei/moo/api"
	"github.com/runner-mei/moo/users/usermodels"
)

const (
	// NoExpiration for use with functions that take an expiration time.
	NoExpiration time.Duration = -1
	// DefaultExpiration for use with functions that take an expiration time. Equivalent to
	// passing in the same expiration duration as was given to New() or
	// NewFrom() when the cache was created (e.g. 5 minutes.)
	DefaultExpiration time.Duration = 0
)

type userCacheItem struct {
	Object     api.User
	Expiration int64
}

// Expired returns true if the item has expired.
func (item userCacheItem) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

type UserCacheBase struct {
	defaultExpiration time.Duration
	items             map[int64]userCacheItem
	mu                sync.RWMutex
	onEvicted         func(int64, api.User)
}

// Set add an item to the cache, replacing any existing item. If the duration is 0
// (DefaultExpiration), the cache's default expiration time is used. If it is -1
// (NoExpiration), the item never expires.
func (c *UserCacheBase) Set(k int64, x api.User, d time.Duration) {
	// "Inlining" of set
	var e int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.mu.Lock()
	c.items[k] = userCacheItem{
		Object:     x,
		Expiration: e,
	}
	// TODO: Calls to mu.Unlock are currently not deferred because defer
	// adds ~200 ns (as of go1.)
	c.mu.Unlock()
}

// SetDefault add an item to the cache, replacing any existing item, using the default
// expiration.
func (c *UserCacheBase) SetDefault(k int64, x api.User) {
	c.Set(k, x, DefaultExpiration)
}

func (c *UserCacheBase) set(k int64, x api.User, d time.Duration) {
	var e int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.items[k] = userCacheItem{
		Object:     x,
		Expiration: e,
	}
}

// Add an item to the cache only if an item doesn't already exist for the given
// key, or if the existing item has expired. Returns an error otherwise.
func (c *UserCacheBase) Add(k int64, x api.User, d time.Duration) error {
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
func (c *UserCacheBase) Replace(k int64, x api.User, d time.Duration) error {
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

func (c *UserCacheBase) isExists(k int64) bool {
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
func (c *UserCacheBase) Get(k int64) (api.User, bool) {
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
func (c *UserCacheBase) GetWithExpiration(k int64) (api.User, time.Time, bool) {
	ov, expiration, ok := c.GetWithExpirationNano(k)
	if expiration == 0 {
		return ov, time.Time{}, ok
	}
	return ov, time.Unix(0, expiration), ok
}

func (c *UserCacheBase) GetWithExpirationNano(k int64) (api.User, int64, bool) {
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
func (c *UserCacheBase) Delete(k int64) {
	c.mu.Lock()
	v, evicted := c.delete(k)
	c.mu.Unlock()
	if evicted {
		c.onEvicted(k, v)
	}
}

func (c *UserCacheBase) delete(k int64) (api.User, bool) {
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
func (c *UserCacheBase) DeleteExpired() {
	var keys []int64
	var items []api.User
	now := time.Now().UnixNano()

	c.mu.Lock()
	if len(c.items) == 0 {
		c.mu.Unlock()
		return
	}

	keys = make([]int64, 0, len(c.items))
	items = make([]api.User, 0, len(c.items))

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
func (c *UserCacheBase) OnEvicted(f func(int64, api.User)) {
	c.mu.Lock()
	c.onEvicted = f
	c.mu.Unlock()
}

// ForEach copies all unexpired items in the cache into a new map and returns it.
func (c *UserCacheBase) ForEach(f func(int64, time.Time, api.User)) {
	var keys []int64
	var items []userCacheItem
	now := time.Now().UnixNano()

	c.mu.RLock()
	if len(c.items) == 0 {
		c.mu.RUnlock()
		return
	}
	keys = make([]int64, 0, len(c.items))
	items = make([]userCacheItem, 0, len(c.items))
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
func (c *UserCacheBase) ItemCount() int {
	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}

// Flush delete all items from the cache.
func (c *UserCacheBase) Flush() {
	c.mu.Lock()
	c.items = map[int64]userCacheItem{}
	c.mu.Unlock()
}

type LoadUserFunc func(ctx context.Context, user *usermodels.User) (api.User, error)
type ReadUserByNameFunc func(ctx context.Context, userName string) (*usermodels.User, error)
type ReadUserByIDFunc func(ctx context.Context, userID int64) (*usermodels.User, error)

type UserCache struct {
	UserCacheBase

	nmu        sync.RWMutex
	name2id    map[string]int64
	findByName ReadUserByNameFunc
	findByID   ReadUserByIDFunc
	load       LoadUserFunc
}

func (c *UserCache) UserByName(ctx context.Context, username string, opts ...api.Option) (api.User, error) {
	var id int64
	c.nmu.RLock()
	if c.name2id != nil {
		id = c.name2id[username]
	}
	c.nmu.RUnlock()

	if id != 0 {
		u, err := c.userByID(ctx, id, false, c.loadByID, opts...)
		if err != nil {
			return nil, err
		}
		if u.Name() == username {
			return u, nil
		}
	}

	mu, err := c.findByName(ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUsernameNotFound(username)
		}
		return nil, err
	}
	id = mu.ID

	c.nmu.Lock()
	if c.name2id == nil {
		c.name2id = map[string]int64{}
	}
	c.name2id[username] = id
	c.nmu.Unlock()

	return c.userByID(ctx, id, true, func(ctx context.Context, userID int64) (api.User, error) {
		return c.load(ctx, mu)
	}, opts...)
}

func (c *UserCache) UserByID(ctx context.Context, userID int64, opts ...api.Option) (api.User, error) {
	return c.userByID(ctx, userID, false, c.loadByID, opts...)
}

func (c *UserCache) loadByID(ctx context.Context, userID int64) (api.User, error) {
	mu, err := c.findByID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserIDNotFound(userID)
		}
		return nil, err
	}
	return c.load(ctx, mu)
}

func (c *UserCache) userByID(ctx context.Context, userID int64, forceUpdate bool, read func(ctx context.Context, userID int64) (api.User, error), opts ...api.Option) (api.User, error) {
	u, ok := c.Get(userID)
	if !ok {
		var err error
		u, err = read(ctx, userID)
		if err != nil {
			return nil, err
		}
		c.SetDefault(userID, u)
	}

	var options = api.InternalApply(opts...)
	if options.UserIncludeDisabled {
		return u, nil
	}

	if u.(*user).IsDisabled() {
		return nil, ErrUserDisabled(u.Name())
	}
	return u, nil
}
