package moo

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"
)

// BusHandler is a receiver for event reference with the given regex pattern
type BusHandler struct {
	Handle  func(ctx context.Context, topicName string, value interface{}) // handler func to process events
	Matcher string                                                         // topic matcher as regex pattern
}

// topic structure
type topic struct {
	name     string
	handlers []*BusHandler
}

type container struct {
	topics   map[string]*topic
	handlers map[string]*BusHandler
}

// Bus is a message bus
type Bus struct {
	sync.Mutex
	data atomic.Value
}

// NewBus inits a new bus
func NewBus() *Bus {
	bus := &Bus{}
	bus.data.Store(&container{
		topics:   make(map[string]*topic),
		handlers: make(map[string]*BusHandler),
	})
	return bus
}

func (b *Bus) setContainer(c *container) {
	b.data.Store(c)
}

func (b *Bus) getContainer() *container {
	return b.data.Load().(*container)
}

func (b *Bus) copyContainer() *container {
	c := b.data.Load().(*container)
	copyed := &container{
		topics:   make(map[string]*topic),
		handlers: make(map[string]*BusHandler),
	}
	for key, t := range c.topics {
		copyed.topics[key] = t
	}
	for key, h := range c.handlers {
		copyed.handlers[key] = h
	}
	return copyed
}

// Emit inits a new event and delivers to the interested in handlers
func (b *Bus) Emit(ctx context.Context, topicName string, data interface{}) error {
	c := b.getContainer()

	t, ok := c.topics[topicName]
	if !ok {
		return fmt.Errorf("bus: topic(%s) not found", topicName)
	}

	for _, h := range t.handlers {
		h.Handle(ctx, topicName, data)
	}
	return nil
}

// Topics lists the all registered topics
func (b *Bus) Topics() []string {
	c := b.getContainer()

	topics, index := make([]string, len(c.topics)), 0
	for topicName := range c.topics {
		topics[index] = topicName
		index++
	}
	return topics
}

// RegisterTopics registers topics and fullfills handlers
func (b *Bus) RegisterTopics(topicNames ...string) {
	b.Lock()
	defer b.Unlock()

	c := b.copyContainer()
	for _, n := range topicNames {
		b.registerTopic(c, n)
	}
	b.setContainer(c)
}

// DeregisterTopics deletes topic
func (b *Bus) DeregisterTopics(topicNames ...string) {
	b.Lock()
	defer b.Unlock()

	c := b.copyContainer()
	for _, n := range topicNames {
		b.deregisterTopic(c, n)
	}
	b.setContainer(c)
}

// TopicHandlers returns all handlers for the topic
func (b *Bus) TopicHandlers(topicName string) []*BusHandler {
	c := b.getContainer()
	return c.topics[topicName].handlers
}

// HandlerKeys returns list of registered handler keys
func (b *Bus) HandlerKeys() []string {
	c := b.getContainer()

	keys, index := make([]string, len(c.handlers)), 0

	for k := range c.handlers {
		keys[index] = k
		index++
	}
	return keys
}

// HandlerTopicSubscriptions returns all topic subscriptions of the handler
func (b *Bus) HandlerTopicSubscriptions(handlerKey string) []string {
	c := b.getContainer()
	return b.handlerTopicSubscriptions(c, handlerKey)
}

func (b *Bus) handlerTopicSubscriptions(c *container, handlerKey string) []string {
	var subscriptions []string
	h, ok := c.handlers[handlerKey]
	if !ok {
		return subscriptions
	}
	for _, t := range c.topics {
		if matched, _ := regexp.MatchString(h.Matcher, t.name); matched {
			subscriptions = append(subscriptions, t.name)
		}
	}
	return subscriptions
}

// On re/register the handler to the registry
func (b *Bus) On(key string, h *BusHandler) {
	b.Lock()
	defer b.Unlock()

	c := b.copyContainer()
	b.registerHandler(c, key, h)
	b.setContainer(c)
}

// Off deletes handler from the registry
func (b *Bus) Off(key string) {
	b.Lock()
	defer b.Unlock()

	c := b.copyContainer()
	b.deregisterHandler(c, key)
	b.setContainer(c)
}

func (b *Bus) registerHandler(c *container, key string, h *BusHandler) {
	b.deregisterHandler(c, key)
	c.handlers[key] = h
	for _, t := range b.handlerTopicSubscriptions(c, key) {
		b.registerTopicHandler(c.topics[t], h)
	}
}

func (b *Bus) deregisterHandler(c *container, handlerKey string) {
	if h, ok := c.handlers[handlerKey]; ok {
		for _, t := range b.handlerTopicSubscriptions(c, handlerKey) {
			b.deregisterTopicHandler(c.topics[t], h)
		}
		delete(c.handlers, handlerKey)
	}
}

func (b *Bus) registerTopicHandlers(c *container, t *topic) {
	for _, h := range c.handlers {
		if matched, _ := regexp.MatchString(h.Matcher, t.name); matched {
			b.registerTopicHandler(t, h)
		}
	}
}

func (b *Bus) registerTopicHandler(t *topic, h *BusHandler) {
	t.handlers = append(t.handlers, h)
}

func (b *Bus) deregisterTopicHandler(t *topic, h *BusHandler) {
	for i, handler := range t.handlers {
		if handler == h {
			t.handlers[i] = t.handlers[len(t.handlers)-1]
			t.handlers[len(t.handlers)-1] = nil
			t.handlers = t.handlers[:len(t.handlers)-1]
			break
		}
	}
}

func (b *Bus) registerTopic(c *container, topicName string) {
	if _, ok := c.topics[topicName]; ok {
		return
	}
	t := &topic{name: topicName, handlers: []*BusHandler{}}
	b.registerTopicHandlers(c, t)
	c.topics[topicName] = t
}

func (b *Bus) deregisterTopic(c *container, topicName string) {
	delete(c.topics, topicName)
}
