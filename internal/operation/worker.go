package operation

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// WorkFunc is a function that a worker can run
type WorkFunc func(
	context.Context,
	chan *pb.Operation_Event,
	hclog.Logger,
) *pb.Operation_Response

// worker is an operation worker. It listens pulls work requests from the
// the queue and executes them.
type worker struct {
	id        string
	ctx       context.Context
	requests  chan *workReq
	events    chan *pb.Operation_Event
	log       hclog.Logger
	saveState func(*pb.Operation_Response) error
}

type workReq struct {
	req *pb.Operation_Request
	f   WorkFunc
}

// newWorker takes a context, work request channel, and update channel and
// returns a new instance of a worker.
func newWorker(
	ctx context.Context,
	id string,
	requests chan *workReq,
	events chan *pb.Operation_Event,
	log hclog.Logger,
	saveState func(*pb.Operation_Response) error,
) *worker {
	return &worker{
		ctx:       ctx,
		id:        id,
		requests:  requests,
		events:    events,
		log:       log,
		saveState: saveState,
	}
}

func newWorkReqForOpReq(op *pb.Operation_Request) (*workReq, error) {
	var err error
	req := &workReq{
		req: op,
	}

	switch op.GetValue().(type) {
	case *pb.Operation_Request_Generate_:
		req.f = GenerateScenario(op)
	case *pb.Operation_Request_Check_:
		req.f = CheckScenario(op)
	case *pb.Operation_Request_Launch_:
		req.f = LaunchScenario(op)
	case *pb.Operation_Request_Destroy_:
		req.f = DestroyScenario(op)
	case *pb.Operation_Request_Run_:
		req.f = RunScenario(op)
	case *pb.Operation_Request_Exec_:
		req.f = ExecScenario(op)
	case *pb.Operation_Request_Output_:
		req.f = OutputScenario(op)
	default:
		req.f, err = UnknownWorkFunc(op)
	}

	return req, err
}

// Run runs the worker. It continuously polls the input channel for new work
// requests.
func (w *worker) run() {
	w.log.Debug("running")

	for {
		select {
		case <-w.ctx.Done():
			w.log.Debug("stopped")
			return
		default:
		}

		select {
		case <-w.ctx.Done():
			w.log.Debug("stopped")
			return
		case req := <-w.requests:
			w.runRequest(req)
		}
	}
}

func (w *worker) sendEvent(event *pb.Operation_Event, done bool) {
	event.Done = done
	w.events <- event
}

// runRequest is responsible for executing our WorkFunc. We execute the WorkFunc,
// filter and pass on the events to the events channel, persist the resulting
// response, and sending the done event.
func (w *worker) runRequest(req *workReq) {
	workCtx, workCancel := context.WithCancel(w.ctx)
	eventC := make(chan *pb.Operation_Event)
	resC := make(chan *pb.Operation_Response, 1)
	eWg := sync.WaitGroup{}
	rWg := sync.WaitGroup{}
	log := w.log.With(RequestDebugArgs(req.req)...)

	// Start the event sender
	eWg.Add(1)
	go func() {
		defer eWg.Done()
		for {
			select {
			case <-workCtx.Done():
				for {
					select {
					case event := <-eventC:
						w.sendEvent(event, false)
					default:
						return
					}
				}
			default:
			}

			select {
			case <-workCtx.Done():
				for {
					select {
					case event := <-eventC:
						w.sendEvent(event, false)
					default:
						return
					}
				}
			case event := <-eventC:
				w.sendEvent(event, false)
			}
		}
	}()

	// Run the operation
	rWg.Add(1)
	go func() {
		defer rWg.Done()

		log.Debug("running operation")
		select {
		case <-workCtx.Done():
			return
		default:
		}

		select {
		case <-workCtx.Done():
			return
		case resC <- req.f(w.ctx, eventC, log.Named(req.req.GetId())):
			return
		}
	}()

	// Wait for completion or cancellation
	rWg.Wait()

	// Kill the event writer. It should drain any pending events that haven't
	// been published.
	workCancel()
	eWg.Wait()

	// Finish the request
	select {
	case res := <-resC:
		if res == nil {
			res, err := NewResponseFromRequest(req.req)
			if err != nil {
				err = fmt.Errorf("work request did not return a response: %w", err)
			} else {
				err = fmt.Errorf("work request did not return a response")
			}
			res.Status = pb.Operation_STATUS_FAILED
			res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
		}
		w.completeRequest(res)
		log.Debug("worker operation completed")
	default:
		log.Debug("worker operation cancelled")
		res, err := NewResponseFromRequest(req.req)
		if err != nil {
			err = fmt.Errorf("work request did not return a response: %w", err)
		} else {
			err = fmt.Errorf("work request did not return a response")
		}
		res.Status = pb.Operation_STATUS_CANCELLED
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
		w.completeRequest(res)
	}
}

// completeRequest is responsible for persisting the operation response into
// the state and sending the done event.
func (w *worker) completeRequest(res *pb.Operation_Response) {
	err := w.saveState(res)
	log := w.log.With(ResponseDebugArgs(res)...)
	if err != nil {
		log.Error("unable to save response state", "error", err)
		res.Diagnostics = append(res.GetDiagnostics(), diagnostics.FromErr(err)...)
	}

	event, err := NewEventFromResponse(res)
	if err != nil {
		res.Status = pb.Operation_STATUS_FAILED
	}
	w.sendEvent(event, true)
}
