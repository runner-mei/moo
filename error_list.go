package moo

import (
	"sync"
	"time"

	"github.com/runner-mei/errors"
	"go.uber.org/fx"
)

type Error struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type ErrorList struct {
	mu   sync.Mutex
	list []Error
}

func (list *ErrorList) Add(err *Error) {
	if err.ID == "" {
		panic(errors.New("id is missing"))
	}
	list.mu.Lock()
	defer list.mu.Unlock()

	for _, e := range list.list {
		if e.ID == err.ID {
			return
		}
	}

	list.list = append(list.list, *err)
}

func (list *ErrorList) Remove(id string) {
	list.mu.Lock()
	defer list.mu.Unlock()

	for idx, e := range list.list {
		if e.ID == id {
			copy(list.list[idx:], list.list[idx+1:])
			break
		}
	}
}

func (list *ErrorList) All() []Error {
	list.mu.Lock()
	defer list.mu.Unlock()

	var results = make([]Error, len(list.list))
	for idx := range list.list {
		results[idx] = list.list[idx]
	}
	return results
}

func init() {
	On(func() Option {
		return fx.Provide(func() *ErrorList {
			return &ErrorList{}
		})
	})
}
