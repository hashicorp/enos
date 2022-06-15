package server

import (
	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/internal/operation"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
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
	sub, unsub, err := s.operator.Stream(req.GetOp())
	if err != nil {
		s.log.Error("failed to initialize stream", append(
			operation.ReferenceDebugArgs(req.GetOp()),
			"subscriber_id", sub.ID,
			"err", err,
		)...)

		return stream.Send(&pb.OperationEventStreamResponse{
			Diagnostics: diagnostics.FromErr(err),
		})
	}
	defer unsub()

	s.log.Debug("starting stream", append(
		operation.ReferenceDebugArgs(req.GetOp()),
		"subscriber_id", sub.ID,
	)...)

	for event := range sub.Events {
		err = stream.Send(&pb.OperationEventStreamResponse{
			Event: event,
		})
		if err != nil {
			s.log.Debug("failed to send event", append(
				operation.ReferenceDebugArgs(req.GetOp()),
				"subscriber_id", sub.ID,
				"err", err,
			)...)

			return err
		}
		if event.GetDone() {
			break
		}
	}

	s.log.Debug("stream completed", append(
		operation.ReferenceDebugArgs(req.GetOp()),
		"subscriber_id", sub.ID,
	)...)

	return nil
}
