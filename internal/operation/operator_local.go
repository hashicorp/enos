// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/state"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
	"github.com/hashicorp/go-hclog"
)

var _ Operator = (*LocalOperator)(nil)

// LocalOperator is an in-memory implementation of the server Operator.
type LocalOperator struct {
	mu           sync.Mutex
	state        state.State
	running      bool
	ctxCancel    func()
	workerCount  int32
	workRequests chan *workReq
	workEvents   chan *pb.Operation_Event
	log          hclog.Logger
	publisher    *Publisher
}

// LocalOperatorOpt is a functional option to configure a new LocalOperator.
type LocalOperatorOpt func(*LocalOperator)

// NewLocalOperator returns a new instance of a LocalOperator.
func NewLocalOperator(opts ...LocalOperatorOpt) *LocalOperator {
	l := &LocalOperator{
		mu:           sync.Mutex{},
		state:        state.NewInMemoryState(),
		running:      false,
		workerCount:  DefaultOperatorWorkerCount,
		workRequests: make(chan *workReq, DefaultOperatorMaxOperationQueue),
		workEvents:   make(chan *pb.Operation_Event, DefaultOperatorMaxOperationEventQueue),
		log:          hclog.NewNullLogger(),
		publisher:    NewPublisher(),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// WithLocalOperatorState is a state setter for a new LocalOperator.
func WithLocalOperatorState(s state.State) LocalOperatorOpt {
	return func(l *LocalOperator) {
		l.state = s
	}
}

// WithLocalOperatorLog is a log setter for a new LocalOperator.
func WithLocalOperatorLog(log hclog.Logger) LocalOperatorOpt {
	return func(l *LocalOperator) {
		l.log = log

		if l.publisher != nil {
			l.publisher.log = log.Named("publisher")
		}
	}
}

// WithLocalOperatorConfig takes the operator configuration and sets it on
// the local operator.
func WithLocalOperatorConfig(cfg *pb.Operator_Config) LocalOperatorOpt {
	return func(l *LocalOperator) {
		if c := cfg.GetWorkerCount(); c != 0 {
			l.workerCount = c
		}
	}
}

// Start starts the operator.
func (o *LocalOperator) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return errors.New("already running")
	}
	o.log.Info("starting")
	ctx, o.ctxCancel = context.WithCancel(ctx)
	o.startEventHandler(ctx)

	for i := range o.workerCount {
		go newWorker(
			strconv.Itoa(int(i)),
			o.workRequests,
			o.workEvents,
			o.log.Named("worker").Named(strconv.Itoa(int(i))),
			func(res *pb.Operation_Response) error {
				return o.state.UpsertOperationResponse(res)
			},
		).run(ctx)
	}

	o.running = true

	return nil
}

// Stop stops the operator. It attempts to gracefully terminate all operations.
func (o *LocalOperator) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.running = false     // Prevent new work requests from being dispatched
	o.drainWorkReqQueue() // Drain any queued operations
	if o.ctxCancel != nil {
		o.ctxCancel() // Cancel in-flight operations, kill operation workers, drain the event queue
	}
	o.publisher.Stop() // Turn off event publisher

	return nil
}

// State returns the operators state.
func (o *LocalOperator) State() state.State {
	return o.state
}

// Dispatch takes an operation request and attempts to dispatch it for execution
// by the operators worker pool. If the request is successfully converted into
// a work operation and queued it will return a reference for the operation to
// the caller.
func (o *LocalOperator) Dispatch(
	req *pb.Operation_Request,
) (
	*pb.Ref_Operation,
	[]*pb.Diagnostic,
) {
	log := o.log.With(RequestDebugArgs(req)...)

	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		log.Error("failed to generate scenario reference from request", "error", err)

		return ref, diagnostics.FromErr(err)
	}

	if !o.running {
		err = errors.New("unable to dispatch new operations as operator is not running")
		log.Error("failed to dispatch operation", "error", err)

		return ref, diagnostics.FromErr(err)
	}

	scenario := flightplan.NewScenario()
	scenario.FromRef(req.GetScenario())

	opUUID, err := uuid.NewRandom()
	if err != nil {
		log.Error("failed to generate operation id", "error", err)

		return ref, diagnostics.FromErr(err)
	}

	// Make sure our request and response share the same operation id
	ref.Id = opUUID.String()
	req.Id = ref.GetId()

	// Create our worker request
	workReq, err := newWorkReqForOpReq(req)
	if err != nil {
		log.Error("failed to determine operation func for request", "error", err)

		return ref, diagnostics.FromErr(err)
	}

	// Set the status for the request as queued in the state and in the event
	// stream
	queueRes, err := NewResponseFromRequest(req)
	if err != nil {
		log.Error("unable to create new response for request")

		return ref, diagnostics.FromErr(err)
	}
	queueRes.Op = ref

	err = o.state.UpsertOperationResponse(queueRes)
	if err != nil {
		log.Error("failed to commit operation response", "error", err)

		return ref, diagnostics.FromErr(err)
	}

	// Dispatch our work request to our worker channel
	select {
	case o.workRequests <- workReq:
		log.Debug("queued operation")
	default:
		queueRes.Op = ref
		queueRes.Status = pb.Operation_STATUS_FAILED
		err = errors.New("failed to queue work request because the queue was full")
		log.Error("failed to queue work request", "error", err)
		diags := diagnostics.FromErr(err)

		err := o.state.UpsertOperationResponse(queueRes)
		diags = append(diags, diagnostics.FromErr(err)...)

		return ref, diags
	}

	return ref, nil
}

// Stream takes a reference to an operation and returns an event subscriber for
// the operation, an unsubscriber function, and an error. The subscriber can be
// used to publish and receive events from the event stream. When the unsubscriber
// function is called the operators event handler will stop publishing events to
// the subscriber.
func (o *LocalOperator) Stream(op *pb.Ref_Operation) (*Subscriber, Unsubscriber, error) {
	sub, err := NewSubscriber(op,
		WithSubscriberLog(o.log.Named("subscriber").Named(op.GetId())),
	)
	if err != nil {
		o.log.Error("unable to create new operation subscriber", "error", err)

		return nil, nil, err
	}

	unsubscribe := o.publisher.Subscribe(sub)

	// Send any existing events to the events channel
	events, err := o.state.GetOperationEvents(op)
	if err != nil {
		return sub, unsubscribe, err
	}
	go func() {
		var err error
		for _, event := range events {
			o.log.Debug("publishing historical event to stream",
				EventDebugArgs(event)...,
			)
			err = o.publisher.Publish(event)
			if err != nil {
				o.log.Error("unable to publish event", "error", err)
			}
		}
	}()

	return sub, unsubscribe, nil
}

// Response takes a reference to an operation and returns the response. If no
// response is found nil will be returned.
func (o *LocalOperator) Response(op *pb.Ref_Operation) (*pb.Operation_Response, error) {
	return o.state.GetOperationResponse(op)
}

func (o *LocalOperator) drainWorkReqQueue() {
	cancelWorkReq := func(req *workReq) {
		log := o.log.With(RequestDebugArgs(req.req)...)

		log.Debug("worker operation cancelled")
		res, err := NewResponseFromRequest(req.req)
		if err != nil {
			err = fmt.Errorf("work request did not return a response: %w", err)
		} else {
			err = errors.New("work request did not return a response")
		}
		res.Status = pb.Operation_STATUS_CANCELLED
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)

		if err != nil {
			log.Error("unable to create response from request", "error", err)
		}

		err = o.state.UpsertOperationResponse(res)
		if err != nil {
			log.Error("unable to save response state", "error", err)
			res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
		}

		event, err := NewEventFromResponse(res)
		if err != nil {
			log.Error("unable to create event from response", "error", err)

			return
		}

		// Send the cancelled event
		o.workEvents <- event

		// Send the done event
		event.Done = true
		o.workEvents <- event
	}

	gracefulCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for {
		select {
		case <-gracefulCtx.Done():
			o.log.Error("failed to drain work request queue")

			return
		default:
		}

		select {
		case <-gracefulCtx.Done():
			o.log.Error("failed to drain work request queue")

			return
		case event := <-o.workRequests:
			cancelWorkReq(event)
		default:
			return
		}
	}
}

func (o *LocalOperator) startEventHandler(ctx context.Context) {
	log := o.log.Named("event_handler")
	log.Debug("starting")

	handleEvent := func(event *pb.Operation_Event) {
		log := log.With(EventDebugArgs(event)...)

		// Add our event to our event history
		err := o.state.AppendOperationEvent(event)
		if err != nil {
			log.Error("failed to append event to state", "error", err)
		}

		// Publish our updates to any stream subscribers
		err = o.publisher.Publish(event)
		if err != nil {
			log.Error("failed to publish event to state", "error", err)
		}
	}

	drainEventQueue := func() {
		gracefulCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		for {
			select {
			case <-gracefulCtx.Done():
				log.Error("failed to drain event queue, stopping")

				return
			default:
			}

			select {
			case <-gracefulCtx.Done():
				log.Error("failed to drain event queue, stopping")

				return
			case event := <-o.workEvents:
				handleEvent(event)
			default:
				log.Debug("stopped")

				return
			}
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				drainEventQueue()
			default:
			}

			select {
			case <-ctx.Done():
				drainEventQueue()
			case event := <-o.workEvents:
				handleEvent(event)
			}
		}
	}()
}
