package operation

import (
	"sync"

	"github.com/google/uuid"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// Subscribers are operation event subscribers
type Subscribers map[string]*Subscriber

// Subscriber is an event subscriber
type Subscriber struct {
	mu     sync.RWMutex
	ID     string
	ref    *pb.Ref_Operation
	Events chan *pb.Operation_Event
	active bool
	log    hclog.Logger
	once   sync.Once
}

// SubscriberOpt is a new subscriber option
type SubscriberOpt func(*Subscriber)

// NewSubscriber takes an operation request and returns a new subscriber instance
func NewSubscriber(
	ref *pb.Ref_Operation,
	opts ...SubscriberOpt,
) (
	*Subscriber,
	error,
) {
	subUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	s := &Subscriber{
		ID:     subUUID.String(),
		mu:     sync.RWMutex{},
		ref:    ref,
		Events: make(chan *pb.Operation_Event, 10),
		active: true,
		log:    hclog.NewNullLogger(),
		once:   sync.Once{},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// WithSubscriberLog sets the subscriber logger
func WithSubscriberLog(log hclog.Logger) SubscriberOpt {
	return func(s *Subscriber) {
		s.log = log
	}
}

// Close closes the subscribers
func (s *Subscriber) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = false
	s.once.Do(func() { close(s.Events) })
	s.log.Debug("closed", "subscriber_id", s.ID)
}

// Send sends an event to the subscribers event channel. Send will block until
// something pulls from the events channel.
func (s *Subscriber) Send(event *pb.Operation_Event) {
	// Gets the message from the channel
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.active {
		s.log.Debug("sending event", append(EventDebugArgs(event),
			"subscriber_id", s.ID)...,
		)
		s.Events <- event
	}
}
