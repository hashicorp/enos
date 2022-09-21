package state

import "github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"

// State is our server state
type State interface {
	// GetOperationResponse returns the most recent committed operation response.
	GetOperationResponse(*pb.Ref_Operation) (*pb.Operation_Response, error)
	// UpsertOperationResponse updates or inserts the operation response.
	UpsertOperationResponse(*pb.Operation_Response) error
	// GetOperationEvents returns an event history for the operation
	GetOperationEvents(*pb.Ref_Operation) ([]*pb.Operation_Event, error)
	// AppendOperationEvent appends an event into the operation event history
	AppendOperationEvent(*pb.Operation_Event) error
}
