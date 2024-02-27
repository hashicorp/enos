// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"sync"

	"github.com/hashicorp/enos/internal/proto"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// Unsubscriber is a func that unsubscribes that subscriber from the publisher.
type Unsubscriber func()

// Publisher is the operation event publisher.
type Publisher struct {
	subscribers map[string]Subscribers // operation id -> subscribers
	mu          sync.RWMutex
	log         hclog.Logger
}

// PublisherOpt is a NewPublisher option.
type PublisherOpt func(*Publisher)

// NewPublisher returns a new instance of the publisher.
func NewPublisher(opts ...PublisherOpt) *Publisher {
	p := &Publisher{
		subscribers: map[string]Subscribers{},
		log:         hclog.NewNullLogger(),
		mu:          sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WithPublisherLog sets the logger on the publisher.
func WithPublisherLog(log hclog.Logger) PublisherOpt {
	return func(p *Publisher) {
		p.log = log
	}
}

// Subscribe adds a new subscribers for the operation and returns a closer
// function.
func (p *Publisher) Subscribe(s *Subscriber) Unsubscriber {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.subscribers[s.ref.GetId()] == nil {
		p.subscribers[s.ref.GetId()] = Subscribers{}
	}
	p.subscribers[s.ref.GetId()][s.ID] = s

	p.log.Debug("added subscriber",
		"operation_id", s.ref.GetId(),
		"subscriber_id", s.ID,
	)

	return func() {
		s.Close()
		p.Unsubscribe(s)
	}
}

// Unsubscribe unsubscribes a subscriber.
func (p *Publisher) Unsubscribe(s *Subscriber) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if subs, ok := p.subscribers[s.ref.GetId()]; ok {
		if _, ok := subs[s.ID]; ok {
			delete(p.subscribers[s.ref.GetId()], s.ID)
		}
	}

	p.log.Debug("removed subscriber",
		"operation_id", s.ref.GetId(),
		"subscriber_id", s.ID,
	)
}

// Publish publishes an operation event to listeners for the operationID.
func (p *Publisher) Publish(event *pb.Operation_Event) error {
	if event == nil {
		return nil
	}
	log := p.log.With(EventDebugArgs(event)...)
	log.Debug("publishing event")

	p.mu.Lock()
	defer p.mu.Unlock()

	subs := p.subscribers[event.GetOp().GetId()]
	for _, s := range subs {
		s.mu.RLock()
		if !s.active {
			s.mu.RUnlock()

			return nil
		}
		s.mu.RUnlock()

		// always copy the event so we don't send copies of mutexes through
		// goroutines
		newEvent := &pb.Operation_Event{}
		err := proto.Copy(event, newEvent)
		if err != nil {
			log.Error("unable to copy operation", "err", err)

			return err
		}

		go (func(s *Subscriber) {
			s.Send(newEvent)
		})(s)
	}

	return nil
}

// Stop closes all subscribers and removes and clears the subscribers list.
func (p *Publisher) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, opSubs := range p.subscribers {
		for _, sub := range opSubs {
			sub.Close()
		}
	}

	p.subscribers = map[string]Subscribers{}
	p.log.Debug("stopped")
}
