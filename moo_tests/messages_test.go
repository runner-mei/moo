package moo_tests

import (
	"context"
	"testing"
	"time"
	"reflect"

	"github.com/runner-mei/moo"
	"github.com/runner-mei/moo/api"
)

func TestMessageList(t *testing.T) {
	var bus *moo.Bus
	var msgList *moo.MessageList
	var app = NewTestApp(t)
	app.Read(&bus)
	app.Read(&msgList)
	app.Start(t)

	var created *moo.Message
	var onCreated = func(ctx context.Context, a *moo.Message) {
		created = a
	}
	var updated *moo.Message
	var onUpdated = func(ctx context.Context, a *moo.Message) {
		updated = a
	}
	var deleted *moo.Message
	var onDeleted = func(ctx context.Context, a *moo.Message) {
		deleted = a
	}

	bus.Register("test1", &moo.BusHandler{
		Matcher: api.BusMessageEventCreated,
		Handle: func(ctx context.Context, topicName string, value interface{}) {
			msg := value.(*moo.Message)
			onCreated(ctx, msg)
		},
	})

	bus.Register("test2", &moo.BusHandler{
		Matcher: api.BusMessageEventUpdated,
		Handle: func(ctx context.Context, topicName string, value interface{}) {
			msg := value.(*moo.Message)
			onUpdated(ctx, msg)
		},
	})

	bus.Register("test3", &moo.BusHandler{
		Matcher: api.BusMessageEventDeleted,
		Handle: func(ctx context.Context, topicName string, value interface{}) {
			msg := value.(*moo.Message)
			onDeleted(ctx, msg)
		},
	})

	excepted := &moo.Message{
		ID: "abc",
		Source: "source",
		Level: moo.MsgInfo,
		Content: "ttesta",
		CreatedAt: time.Now(),
	}

	created = nil
	msgList.Add(excepted)
	time.Sleep(1 * time.Microsecond)
	if !reflect.DeepEqual(excepted, created) {
		t.Error("want", excepted, "got", created)
	}


	deleted = nil
	msgList.Remove(excepted.ID)
	time.Sleep(1 * time.Microsecond)
	if !reflect.DeepEqual(excepted, deleted) {
		t.Error("want", excepted, "got", deleted)
	}

	ph := msgList.Placeholder(excepted.ID, excepted.Source)

	created = nil
	ph.Set(excepted.Level, excepted.Content)
	time.Sleep(1 * time.Microsecond)
	if created == nil {
		t.Error("created is nil")
	} else {
		excepted.CreatedAt = created.CreatedAt
		if !reflect.DeepEqual(excepted, created) {
			t.Error("want", excepted, "got", created)
		}
	}
	updated = nil
	excepted.Content = "abc"
	ph.Set(excepted.Level, excepted.Content)
	time.Sleep(1 * time.Microsecond)
	excepted.CreatedAt = updated.CreatedAt
	if !reflect.DeepEqual(excepted, updated) {
		t.Error("want", excepted, "got", updated)
	}


	deleted = nil
	ph.Clear()
	time.Sleep(1 * time.Microsecond)
	if !reflect.DeepEqual(excepted, deleted) {
		t.Error("want", excepted, "got", deleted)
	}

	created = nil
	ph.Set(excepted.Level, excepted.Content)
	time.Sleep(1 * time.Microsecond)
	excepted.CreatedAt = created.CreatedAt
	if !reflect.DeepEqual(excepted, created) {
		t.Error("want", excepted, "got", created)
	}

	deleted = nil
	ph.Set(excepted.Level, "")
	time.Sleep(1 * time.Microsecond)
	// excepted.CreatedAt = deleted.CreatedAt
	if !reflect.DeepEqual(excepted, deleted) {
		t.Error("want", excepted, "got", deleted)
	}

	created = nil
	updated = nil
	deleted = nil

	var provider = &testMessageProvider{}
	msgList.SetupProvider(provider)
	provider.cb(moo.MessageChangeRecheck, nil)

	if created != nil {
		t.Error("created isnot nil")
	}
	if updated != nil {
		t.Error("updated isnot nil")
	}
	if deleted != nil {
		t.Error("deleted isnot nil")
	}
}


var _ moo.MessageChangeListenerSetter  = &testMessageProvider{}

type testMessageProvider struct {
	cb moo.MessageChangeListenerFunc
}
func ( *testMessageProvider) Get() []moo.Message {
	return nil
}
func (t *testMessageProvider) SetMessageChangeListener(cb moo.MessageChangeListenerFunc) {
	t.cb = cb
}
