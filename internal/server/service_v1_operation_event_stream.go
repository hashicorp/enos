// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/operation"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

// OperationEventStream takes an operation request with an operation ID and streams
// the operation response events. Messages will be populated with updates and
// the final message will be populated with a complete response. When the operation
// is not in a failed or completed state the stream will remain open until
// the operation is cancelled or completes.
func (s *ServiceV1) OperationEventStream(
	req *pb.OperationEventStreamRequest,
	stream pb.EnosService_OperationEventStreamServer,
) error {
	log := s.log.With(operation.ReferenceDebugArgs(req.GetOp())...)

	sub, unsub, err := s.operator.Stream(req.GetOp())
	log = log.With("subscriber_id", sub.ID)

	if err != nil {
		log.Error("failed to initialize stream", "error", err)

		return stream.Send(&pb.OperationEventStreamResponse{
			Diagnostics: diagnostics.FromErr(err),
		})
	}
	defer unsub()

	log.Debug("starting stream")

	for event := range sub.Events {
		err = stream.Send(&pb.OperationEventStreamResponse{
			Event: event,
		})
		if err != nil {
			log.Debug("failed to send event", "error", err)

			return err
		}
		if event.GetDone() {
			break
		}
	}

	log.Debug("stream completed")

	return nil
}
