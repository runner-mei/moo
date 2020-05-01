package moo_test

import (
	"context"
	"testing"

	 "github.com/runner-mei/moo"
	"github.com/stretchr/testify/assert"
)

type ctxKey string

const (
	// CtxKeyTxID tx id context key
	CtxKeyTxID = ctxKey("bus.txID")
)

func TestEmit(t *testing.T) {
	b := setup("comment.created", "comment.deleted")
	defer tearDown(b, "comment.created", "comment.deleted")

	t.Run("with handler", func(t *testing.T) {
		ctx := context.Background()
		registerFakeHandler(b, "test", t)

		err := b.Emit(ctx, "comment.created", "my comment with handler")
		if err != nil {
			t.Fatalf("emit failed: %v", err)
		}
		b.Off("test")
	})
	t.Run("with unknown topic", func(t *testing.T) {
		ctx := context.Background()
		err := b.Emit(ctx, "comment.updated", "my comment")

		assert := assert.New(t)
		assert.NotNil(err)
		assert.Equal("bus: topic(comment.updated) not found", err.Error())
	})
}

func TestTopics(t *testing.T) {
	topicNames := []string{"user.created", "user.deleted"}
	b := setup(topicNames...)
	defer tearDown(b, topicNames...)

	assert.ElementsMatch(t, topicNames, b.Topics())
}

func TestRegisterTopics(t *testing.T) {
	b := setup()
	defer tearDown(b)

	topicNames := []string{"user.created", "user.deleted"}
	defer b.DeregisterTopics(topicNames...)

	t.Run("register topics", func(t *testing.T) {
		b.RegisterTopics(topicNames...)
		assert.ElementsMatch(t, topicNames, b.Topics())
	})
	t.Run("does not register a topic twice", func(t *testing.T) {
		assert := assert.New(t)
		assert.Len(b.Topics(), 2)
		b.RegisterTopics("user.created")
		assert.Len(b.Topics(), 2)
		assert.ElementsMatch(topicNames, b.Topics())
	})
}

func TestDeregisterTopics(t *testing.T) {
	b := setup()
	defer tearDown(b)

	topicNames := []string{"user.created", "user.deleted", "user.updated"}
	defer b.DeregisterTopics(topicNames...)

	b.RegisterTopics(topicNames...)
	b.DeregisterTopics("user.created", "user.updated")
	assert := assert.New(t)
	assert.ElementsMatch([]string{"user.deleted"}, b.Topics())
}

func TestTopicHandlers(t *testing.T) {
	b := setup()
	defer tearDown(b)
	defer b.Off("test.handler/1")
	defer b.Off("test.handler/2")

	handler := fakeHandler(".*")
	b.On("test.handler/1", &handler)
	b.On("test.handler/2", &handler)
	b.RegisterTopics("user.created")

	assert := assert.New(t)
	for _, h := range b.TopicHandlers("user.created") {
		assert.Equal(&handler, h)
	}
}

func TestHandlerKeys(t *testing.T) {
	b := setup("comment.created", "comment.deleted")
	defer tearDown(b, "comment.created", "comment.deleted")
	defer b.Off("test.key.1")
	defer b.Off("test.key.2")

	h := fakeHandler(".*")
	b.On("test.key.1", &h)
	b.On("test.key.2", &h)

	want := []string{"test.key.1", "test.key.2"}
	assert.ElementsMatch(t, want, b.HandlerKeys())
}

func TestHandlerTopicSubscriptions(t *testing.T) {
	b := setup("comment.created", "comment.deleted")
	defer tearDown(b, "comment.created", "comment.deleted")

	tests := []struct {
		handler          moo.BusHandler
		handlerKey       string
		handlerLookupKey string
		want             []string
	}{
		{fakeHandler(".*"), "test.handler.1", "test.handler.1", []string{"comment.created", "comment.deleted"}},
		{fakeHandler("user.updated"), "test.handler.2", "test.handler.2", []string{}},
		{fakeHandler(".*"), "test.handler.3", "test.handler.NA", []string{}},
	}

	for _, test := range tests {
		b.On(test.handlerKey, &test.handler)

		assert.ElementsMatch(t, test.want, b.HandlerTopicSubscriptions(test.handlerLookupKey))
	}
}

func TestOn(t *testing.T) {
	b := setup("comment.created", "comment.deleted")
	defer tearDown(b, "comment.created", "comment.deleted")
	defer b.Off("test.handler")

	h := fakeHandler(".*created$")
	b.On("test.handler", &h)

	t.Run("registers handler key", func(t *testing.T) {
		assert.True(t, isHandlerKeyExists(b, "test.handler"))
	})
	t.Run("adds handler references to the matched topics", func(t *testing.T) {
		t.Run("when topic is matched", func(t *testing.T) {
			assert.True(t, isTopicHandler(b, "comment.created", &h))
		})
		t.Run("when topic is not matched", func(t *testing.T) {
			assert.False(t, isTopicHandler(b, "comment.deleted", &h))
		})
	})
}

func TestOff(t *testing.T) {
	b := setup("comment.created", "comment.deleted")
	defer tearDown(b, "comment.created", "comment.deleted")

	h := fakeHandler(".*")
	b.On("test.handler", &h)
	b.Off("test.handler")

	t.Run("deletes handler key", func(t *testing.T) {
		assert.False(t, isHandlerKeyExists(b, "test.handler"))
	})
	t.Run("deletes handler references from the topics", func(t *testing.T) {
		assert := assert.New(t)
		for _, topic := range b.Topics() {
			assert.False(isTopicHandler(b, topic, &h))
		}
	})
}

func setup(topicNames ...string) *moo.Bus {
	b := moo.NewBus()
	b.RegisterTopics(topicNames...)
	return b
}

func tearDown(b *moo.Bus, topicNames ...string) {
	b.DeregisterTopics(topicNames...)
}

func fakeHandler(matcher string) moo.BusHandler {
	return moo.BusHandler{Handle: func(ctx context.Context, topic string, data interface{}) {}, Matcher: matcher}
}

func registerFakeHandler(b *moo.Bus, key string, t *testing.T) {
	fn := func(ctx context.Context, topic string, data interface{}) {
		t.Run("receives right event", func(t *testing.T) {
			assert := assert.New(t)
			assert.NotNil(ctx)
			assert.Equal("comment.created", topic)
			assert.Equal("my comment with handler", data)
		})
	}
	h := moo.BusHandler{Handle: fn, Matcher: ".*created$"}
	b.On(key, &h)
}

func isTopicHandler(b *moo.Bus, topicName string, h *moo.BusHandler) bool {
	for _, th := range b.TopicHandlers(topicName) {
		if h == th {
			return true
		}
	}
	return false
}

func isHandlerKeyExists(b *moo.Bus, key string) bool {
	for _, k := range b.HandlerKeys() {
		if k == key {
			return true
		}
	}
	return false
}
