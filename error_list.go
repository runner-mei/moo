package moo

import (
	"sync"
	"time"

	"github.com/runner-mei/errors"
	"go.uber.org/fx"
)

type ErrLevel string

const (
	ErrWarn  ErrLevel = "warn"
	ErrError ErrLevel = "error"
	ErrFatal ErrLevel = "fatal"
)

type Error struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Level     ErrLevel  `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type ErrorPlaceholder interface {
	SetError(level ErrLevel, msg string)
}

type placeholder struct {
	mu *sync.Mutex

	err Error
}

func (ph *placeholder) SetError(level ErrLevel, msg string) {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	if msg != "" && ph.err.Message == "" {
		ph.err.CreatedAt = time.Now()
	}
	ph.err.Message = msg
}

func (ph *placeholder) isError() bool {
	return ph.err.Message != ""
}

func (ph *placeholder) toError() Error {
	return ph.err
}

type ErrorList struct {
	mu           sync.Mutex
	list         []Error
	placeholders []placeholder
}

func (list *ErrorList) Placeholder(id, source string) ErrorPlaceholder {
	if id == "" {
		panic(errors.New("id is missing"))
	}
	list.mu.Lock()
	defer list.mu.Unlock()

	for idx, ph := range list.placeholders {
		if ph.err.ID == id {
			return &list.placeholders[idx]
		}
	}

	list.placeholders = append(list.placeholders, placeholder{
		mu: &list.mu,
		err: Error{
			ID:     id,
			Source: source,
		},
	})
	return &list.placeholders[len(list.placeholders)-1]
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
	for idx := range list.placeholders {
		if list.placeholders[idx].isError() {
			results = append(results, list.placeholders[idx].toError())
		}
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
