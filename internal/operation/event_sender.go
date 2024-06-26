// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hashicorp/enos/internal/proto"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

type EventSender struct {
	events chan *pb.Operation_Event
}

func NewEventSender(events chan *pb.Operation_Event) *EventSender {
	return &EventSender{events: events}
}

func (e *EventSender) Publish(event *pb.Operation_Event) error {
	if event != nil {
		cpy := &pb.Operation_Event{}
		err := proto.Copy(event, cpy)
		if err != nil {
			return err
		}

		cpy.PublishedAt = timestamppb.Now()

		e.events <- cpy
	}

	return nil
}

func (e *EventSender) PublishResponse(res *pb.Operation_Response) error {
	event, err := NewEventFromResponse(res)
	if err != nil {
		return err
	}

	return e.Publish(event)
}
