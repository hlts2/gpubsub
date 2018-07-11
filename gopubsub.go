package gopubsub

import (
	"sync"
)

const (
	defaultPubSubTopicCapacity = 100000
	initialSubscriberCapacity  = 100
)

// PubSub is pubsub messasing object
type PubSub struct {
	mu         *sync.Mutex
	registries []*registry
}

type registry struct {
	topic       string
	subscribers subscribers
}

type subscriber struct {
	ch  chan interface{}
	pos int
}

type subscribers []*subscriber

// Subscriber is interface that wraps the Read methods
type Subscriber interface {
	Read() <-chan interface{}
}

// Read returns the channel that receive the message
func (s *subscriber) Read() <-chan interface{} {
	return s.ch
}

// NewPubSub returns PubSub object
func NewPubSub() *PubSub {
	ps := &PubSub{
		mu:         new(sync.Mutex),
		registries: make([]*registry, 0, defaultPubSubTopicCapacity),
	}

	for i := 0; i < defaultPubSubTopicCapacity; i++ {
		ps.registries = append(ps.registries, &registry{
			topic:       "",
			subscribers: make(subscribers, 0, initialSubscriberCapacity),
		})
	}

	return ps
}

// Subscribe subscribes to a topic
func (p *PubSub) Subscribe(topic string) Subscriber {
	subscriber := &subscriber{
		ch: make(chan interface{}, 1),
	}

	p.mu.Lock()

	p.subscribe(topic, subscriber)

	p.mu.Unlock()

	return subscriber
}

func (p *PubSub) subscribe(topic string, subscriber *subscriber) {
	if topic == "" {
		return
	}

	hash := int(generateHash(topic)) % len(p.registries)

	for i := hash; i < len(p.registries); i++ {
		registory := p.registries[i]

		if registory.topic == "" {
			registory.topic = topic
			subscriber.pos = 0
		} else if registory.topic == topic {
			subscriber.pos = len(registory.subscribers) - 1
		} else {
			continue
		}

		// TODO if capacity is insufficient, reserve again
		registory.subscribers = append(registory.subscribers, subscriber)
		break
	}
}

// Publish sends a message to subscribers subscribing to topic
func (p *PubSub) Publish(topic string, message interface{}) {
	if topic == "" {
		return
	}

	p.mu.Lock()

	subscribers := p.subscribers(topic)
	if subscribers != nil {
		for _, subscriber := range subscribers {
			subscriber.ch <- message
		}
	}

	p.mu.Unlock()
}

func (p *PubSub) subscribers(topic string) subscribers {
	hash := int(generateHash(topic)) % len(p.registries)

	for i := hash; i < len(p.registries); i++ {
		registory := p.registries[i]
		if registory.topic == topic {
			return registory.subscribers
		}
	}

	return nil
}

// UnSubscribe unsubscribes topic
func (p *PubSub) UnSubscribe(topic string, target Subscriber) {
	if topic == "" {
		return
	}

	if ss, ok := target.(*subscriber); ok {
		p.mu.Lock()

		p.unSubscribe(topic, ss)

		p.mu.Unlock()
	}
}

func (p *PubSub) unSubscribe(topic string, subscriber *subscriber) {
	subscribers := p.subscribers(topic)
	if subscribers != nil {
		return
	}

	pos := subscriber.pos
	subscribers = append(subscribers[:pos], subscribers[pos+1:]...)
}
