package operation

import (
	"context"

	"github.com/hashicorp/enos/internal/state"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

var (
	// These have all been made up and have no scientific backing.

	// DefaultOperatorWorkerCount is how many workers to run. The number of
	// parallel operations is limited to the number of workers.
	DefaultOperatorWorkerCount int32 = 4
	// DefaultOperatorMaxOperationQueue is default maximum number of queued
	// operations.
	DefaultOperatorMaxOperationQueue = 10000
	// DefaultOperatorMaxOperationEventQueue is the maximum number of events
	// that can be queued before being persisted in the state by the event
	// handler.
	DefaultOperatorMaxOperationEventQueue = 1000
)

// Operator is the server operation handler.
type Operator interface {
	Dispatch(*pb.Operation_Request) (*pb.Ref_Operation, []*pb.Diagnostic)
	Stream(*pb.Ref_Operation) (*Subscriber, Unsubscriber, error)
	Response(*pb.Ref_Operation) (*pb.Operation_Response, error)
	Stop() error
	Start(context.Context) error
	State() state.State
}
