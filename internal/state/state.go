// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"

// State is our server state.
type State interface {
	// GetOperationResponse returns the most recent committed operation response.
	GetOperationResponse(ref *pb.Ref_Operation) (*pb.Operation_Response, error)
	// UpsertOperationResponse updates or inserts the operation response.
	UpsertOperationResponse(res *pb.Operation_Response) error
	// GetOperationEvents returns an event history for the operation
	GetOperationEvents(ref *pb.Ref_Operation) ([]*pb.Operation_Event, error)
	// AppendOperationEvent appends an event into the operation event history
	AppendOperationEvent(ev *pb.Operation_Event) error
}
