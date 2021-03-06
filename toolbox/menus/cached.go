// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package menus

import (
	"sync/atomic"
	"time"
)

type CachedValue struct {
	MaxAge int64
	data   atomic.Value
}

type cachedData struct {
	value     map[string][]Menu
	timestamp int64
}

func (cv *CachedValue) Get() map[string][]Menu {
	var v map[string][]Menu
	return cv.Read(v, cv.MaxAge)
}

func (cv *CachedValue) GetWithDefault(v map[string][]Menu) map[string][]Menu {
	return cv.Read(v, cv.MaxAge)
}

func (cv *CachedValue) Read(v map[string][]Menu, maxAge int64) map[string][]Menu {
	o := cv.data.Load()
	if o == nil {
		return v
	}
	cdata, ok := o.(*cachedData)
	if !ok {
		return v
	}
	if (cdata.timestamp + maxAge) < time.Now().Unix() {
		return v
	}
	return cdata.value
}

func (cv *CachedValue) Set(v map[string][]Menu, t time.Time) {
	cv.data.Store(&cachedData{
		value:     v,
		timestamp: t.Unix(),
	})
}
