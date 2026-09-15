package main

import (
	"errors"
	"sync"
)

type Subscriber struct {
	Topic  string
	ch     chan Message
	mu     sync.RWMutex
	closed bool
}

type Consumer interface {
	Messages() <-chan Message
}

func NewSubscriber(topic string) *Subscriber {
	return &Subscriber{
		Topic: topic,
		ch:    make(chan Message, 1000),
	}
}

type Producer interface {
	Publish(topic string, msg Message) error
}

type Message struct {
	ID   string
	Data string
}

type Broker interface {
	Subscribe(topic string) (*Subscriber, error)
	Unsubscribe(topic string, sub *Subscriber) error
	Shutdown() // we will think this later
}

type PubSubBroker struct {
	topics   map[string]map[*Subscriber]struct{}
	shutdown bool
	mu       sync.RWMutex
}

func NewPubSubBroker() *PubSubBroker {
	return &PubSubBroker{
		topics:   map[string]map[*Subscriber]struct{}{},
		shutdown: false,
	}
}

// Client facing APIS
func (c *Subscriber) Messages() <-chan Message {
	return c.ch
}

func (b *PubSubBroker) Publish(topic string, msg Message) error {
	b.mu.RLock()
	if b.shutdown {
		b.mu.RUnlock()
		return errors.New("broker is shutdown")
	}
	consumers := make([]*Subscriber, 0, len(b.topics[topic]))
	for consumer := range b.topics[topic] {
		consumers = append(consumers, consumer)
	}
	b.mu.RUnlock()
	for _, consumer := range consumers {
		consumer.mu.RLock()
		if !consumer.closed {
			select {
			case consumer.ch <- msg:
			default:
			}
		}
		consumer.mu.RUnlock()
	}
	return nil
}

func (b *PubSubBroker) Subscribe(topic string) (*Subscriber, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.shutdown {
		return nil, errors.New("broker is shutdown")
	}
	sub := NewSubscriber(topic)
	if _, exists := b.topics[topic]; !exists {
		b.topics[topic] = make(map[*Subscriber]struct{})
	}
	b.topics[topic][sub] = struct{}{}
	return sub, nil
}

func (b *PubSubBroker) Unsubscribe(topic string, sub *Subscriber) error {
	b.mu.Lock()
	subscribers, exists := b.topics[topic]
	if !exists {
		b.mu.Unlock()
		return errors.New("topic does not exist")
	}
	if _, exists := subscribers[sub]; !exists {
		b.mu.Unlock()
		return errors.New("subscriber does not belong to topic")
	}
	delete(subscribers, sub)
	if len(subscribers) == 0 {
		delete(b.topics, topic)
	}
	b.mu.Unlock()
	sub.mu.Lock()
	if !sub.closed {
		sub.closed = true
		close(sub.ch)
	}
	sub.mu.Unlock()
	return nil
}

func (b *PubSubBroker) Shutdown() {
	b.mu.Lock()
	if b.shutdown {
		b.mu.Unlock()
		return
	}
	b.shutdown = true
	var subscribers []*Subscriber
	for _, topicSubscribers := range b.topics {
		for sub := range topicSubscribers {
			subscribers = append(subscribers, sub)
		}
	}
	b.topics = make(map[string]map[*Subscriber]struct{})
	b.mu.Unlock()
	for _, sub := range subscribers {
		sub.mu.Lock()
		if !sub.closed {
			sub.closed = true
			close(sub.ch)
		}
		sub.mu.Unlock()
	}

}

func main() {

}
