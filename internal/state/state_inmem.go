// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"sync"

	"github.com/hashicorp/enos/internal/flightplan"
	"github.com/hashicorp/enos/internal/proto"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

var _ State = (*InMemoryState)(nil)

// InMemoryState is an implementation of our state that lives only in memory.
type InMemoryState struct {
	mu sync.RWMutex
	//	scenario ID -> operation id -> operation response
	responses map[string]map[string]*pb.Operation_Response
	//	scenario ID -> operation id -> operations events
	events map[string]map[string][]*pb.Operation_Event
}

// NewInMemoryState returns a new InMemoryState.
func NewInMemoryState() *InMemoryState {
	return &InMemoryState{
		mu:        sync.RWMutex{},
		responses: map[string]map[string]*pb.Operation_Response{},
		events:    map[string]map[string][]*pb.Operation_Event{},
	}
}

// GetOperationResponse takes a reference to an operation and returns the most
// recent response.
func (i *InMemoryState) GetOperationResponse(
	ref *pb.Ref_Operation,
) (
	*pb.Operation_Response,
	error,
) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.getOperationResponse(ref)
}

// GetOperationEvents takes a reference to an operation and returns the
// entire event history.
func (i *InMemoryState) GetOperationEvents(
	ref *pb.Ref_Operation,
) (
	[]*pb.Operation_Event,
	error,
) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.getOperationEvents(ref)
}

// UpsertOperationResponse takes and operation response and updates or inserts
// it into the response history.
func (i *InMemoryState) UpsertOperationResponse(res *pb.Operation_Response) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	var sid string
	if ref := res.GetOp().GetScenario(); ref != nil {
		scenario := flightplan.NewScenario()
		scenario.FromRef(ref)
		sid = scenario.UID()
	}
	if sid == "" {
		return errors.New("invalid scenario id")
	}

	resCopy := &pb.Operation_Response{}
	err := proto.Copy(res, resCopy)
	if err != nil {
		return err
	}

	oid := resCopy.GetOp().GetId()
	if oid == "" {
		return errors.New("invalid operation id")
	}

	_, ok := i.responses[sid]
	if !ok {
		i.responses[sid] = map[string]*pb.Operation_Response{oid: resCopy}

		return nil
	}

	i.responses[sid][oid] = resCopy

	return nil
}

// AppendOperationEvent takes an operation event and appends it into the operations
// event history.
func (i *InMemoryState) AppendOperationEvent(event *pb.Operation_Event) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	var sid string
	if ref := event.GetOp().GetScenario(); ref != nil {
		scenario := flightplan.NewScenario()
		scenario.FromRef(ref)
		sid = scenario.UID()
	}
	if sid == "" {
		return errors.New("invalid scenario id")
	}

	eventCopy := &pb.Operation_Event{}
	err := proto.Copy(event, eventCopy)
	if err != nil {
		return err
	}

	eid := eventCopy.GetOp().GetId()
	if eid == "" {
		return errors.New("invalid operation id")
	}

	eventsHistory, ok := i.events[sid]
	if !ok {
		i.events[sid] = map[string][]*pb.Operation_Event{eid: {eventCopy}}

		return nil
	}

	eventHistory, ok := eventsHistory[eid]
	if !ok {
		i.events[sid][eid] = []*pb.Operation_Event{eventCopy}

		return nil
	}

	i.events[sid][eid] = append(eventHistory, eventCopy)

	return nil
}

// getOperationResponse takes a reference to an operation and returns the most
// recent response.
func (i *InMemoryState) getOperationResponse(
	ref *pb.Ref_Operation,
) (
	*pb.Operation_Response,
	error,
) {
	scenario := flightplan.NewScenario()
	scenarioRef := ref.GetScenario()
	if scenarioRef == nil {
		return nil, errors.New("state cannot retrieve response record without scenario reference")
	}
	scenario.FromRef(scenarioRef)
	sid := scenario.UID()
	uid := ref.GetId()

	if sid == "" {
		return nil, errors.New("state cannot retrieve response record without scenario ID")
	}
	if uid == "" {
		return nil, errors.New("state cannot retrieve response record without operation ID")
	}

	_, ok := i.responses[sid]
	if !ok {
		return nil, errors.New("no operations matching scenario ID")
	}

	op, ok := i.responses[sid][uid]
	if !ok {
		return nil, errors.New("no operations matching scenario and operation IDs")
	}

	return op, nil
}

// getOperationEvent takes a reference to an operation and returns the event
// stream.
func (i *InMemoryState) getOperationEvents(
	ref *pb.Ref_Operation,
) (
	[]*pb.Operation_Event,
	error,
) {
	scenario := flightplan.NewScenario()
	scenarioRef := ref.GetScenario()
	if scenarioRef == nil {
		return nil, errors.New("state cannot retrieve event stream without scenario reference")
	}
	scenario.FromRef(scenarioRef)
	sid := scenario.UID()
	uid := ref.GetId()

	if sid == "" {
		return nil, errors.New("state cannot retrieve event stream without scenario ID")
	}
	if uid == "" {
		return nil, errors.New("state cannot retrieve event stream without operation ID")
	}

	_, ok := i.events[sid]
	if !ok {
		return nil, nil
	}

	op, ok := i.events[sid][uid]
	if !ok {
		return nil, nil
	}

	return op, nil
}
