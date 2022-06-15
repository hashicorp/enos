package operation

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/state"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

var _ Operator = (*LocalOperator)(nil)

// LocalOperator is an in-memory implementation of the server Operator
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

// LocalOperatorOpt is a functional option to configure a new LocalOperator
type LocalOperatorOpt func(*LocalOperator)

// NewLocalOperator returns a new instance of a LocalOperator
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

// WithLocalOperatorState is a state setter for a new LocalOperator
func WithLocalOperatorState(s state.State) LocalOperatorOpt {
	return func(l *LocalOperator) {
		l.state = s
	}
}

// WithLocalOperatorLog is a log setter for a new LocalOperator
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
func (r *LocalOperator) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("already running")
	}
	r.log.Info("starting")
	ctx, r.ctxCancel = context.WithCancel(ctx)
	r.startEventHandler(ctx)

	for i := int32(0); i < r.workerCount; i++ {
		go newWorker(
			ctx,
			fmt.Sprintf("%d", i),
			r.workRequests,
			r.workEvents,
			r.log.Named("worker").Named(fmt.Sprintf("%d", i)),
			func(res *pb.Operation_Response) error {
				return r.state.UpsertOperationResponse(res)
			},
		).run()
	}

	r.running = true

	return nil
}

// Stop stops the operator. It attempts to gracefully terminate all operations.
func (r *LocalOperator) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.publisher.Stop()
	r.running = false
	r.ctxCancel()
	return nil
}

// State returns the operators state
func (r *LocalOperator) State() state.State {
	return r.state
}

// Dispatch dispatches an  If successful it will return a reference
// to an operation including its ID and any diagnostics encountered when attempting
// to dispatch the
func (r *LocalOperator) Dispatch(
	req *pb.Operation_Request,
) (
	*pb.Ref_Operation,
	[]*pb.Diagnostic,
) {
	ref, err := NewReferenceFromRequest(req)
	if err != nil {
		return ref, diagnostics.FromErr(err)
	}

	scenario := flightplan.NewScenario()
	scenario.FromRef(req.GetScenario())

	opUUID, err := uuid.NewRandom()
	if err != nil {
		r.log.Error("failed to generate operation id",
			append(RequestDebugArgs(req), "err", err)...,
		)
		return ref, diagnostics.FromErr(err)
	}

	// Make sure our request and response share the same operation id
	ref.Id = opUUID.String()
	req.Id = ref.Id

	// Create our worker request
	workReq, err := newWorkReqForOpReq(req)
	if err != nil {
		r.log.Error("failed to determine operation func for request",
			append(RequestDebugArgs(req), "err", err)...,
		)
		return ref, diagnostics.FromErr(err)
	}

	// Set the status for the request as queued in the state and in the event
	// stream
	queueRes, err := NewResponseFromRequest(req)
	if err != nil {
		r.log.Error("unable to create new response for request",
			ResponseDebugArgs(queueRes)...,
		)
		return ref, diagnostics.FromErr(err)
	}
	queueRes.Op = ref

	err = r.state.UpsertOperationResponse(queueRes)
	if err != nil {
		r.log.Error("failed to commit operation response",
			append(ResponseDebugArgs(queueRes), "err", err)...,
		)
		return ref, diagnostics.FromErr(err)
	}

	// Dispatch our work request to our worker channel
	select {
	case r.workRequests <- workReq:
		r.log.Debug("queued operation", RequestDebugArgs(workReq.req)...)
	default:
		queueRes.Op = ref
		queueRes.Status = pb.Operation_STATUS_FAILED
		r.log.Error("failed to queue work request",
			ResponseDebugArgs(queueRes)...,
		)
		diags := diagnostics.FromErr(err)

		err := r.state.UpsertOperationResponse(queueRes)
		diags = append(diags, diagnostics.FromErr(err)...)

		return ref, diags
	}

	return ref, nil
}

// Stream takes a reference to an operation and returns channels where operation
// events are returned, along with an Unsubscriber function that can be called
// to close the stream on the operator. The stream will remain active and until
// the Unsubscriber is called.
func (r *LocalOperator) Stream(
	op *pb.Ref_Operation,
) (*Subscriber, Unsubscriber, error) {
	sub, err := NewSubscriber(op,
		WithSubscriberLog(r.log.Named("subscriber").Named(op.GetId())),
	)
	if err != nil {
		r.log.Error("unable to create new operation subscriber", "error", err)
		return nil, nil, err
	}

	unsubscribe := r.publisher.Subscribe(sub)

	// Send any existing events to the events channel
	events, err := r.state.GetOperationEvents(op)
	if err != nil {
		return sub, unsubscribe, err
	}
	go func() {
		var err error
		for _, event := range events {
			r.log.Debug("publishing historical event to stream",
				EventDebugArgs(event)...,
			)
			err = r.publisher.Publish(event)
			if err != nil {
				r.log.Error("unable to publish event", "error", err)
			}
		}
	}()

	return sub, unsubscribe, nil
}

// Response takes a reference to an operation and retuns the response. If no
// response is found nil will be returned.
func (r *LocalOperator) Response(op *pb.Ref_Operation) (*pb.Operation_Response, error) {
	return r.state.GetOperationResponse(op)
}

func (r *LocalOperator) startEventHandler(ctx context.Context) {
	log := r.log.Named("event_handler")
	log.Debug("starting")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Debug("stopped")
				return
			default:
			}

			select {
			case <-ctx.Done():
				log.Debug("stopped")
				return
			case event := <-r.workEvents:
				// Add our event to our event history
				err := r.state.AppendOperationEvent(event)
				if err != nil {
					log.Error("failed to append event to state", append(
						EventDebugArgs(event), "error", err)...,
					)
				}

				// Publish our updates to any stream subscribers
				err = r.publisher.Publish(event)
				if err != nil {
					log.Error("failed to publish event to listeners", append(
						EventDebugArgs(event), "error", err)...,
					)
				}
			}
		}
	}()
}
