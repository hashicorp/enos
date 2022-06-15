package operation

import (
	"context"
	"fmt"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
	"github.com/hashicorp/go-hclog"
)

// UnknownWorkFunc takes an operation request and returns a work func that will
// fail because it is unknown.
func UnknownWorkFunc(req *pb.Operation_Request) (WorkFunc, error) {
	err := fmt.Errorf("no worker func for request")

	return func(
		ctx context.Context,
		events chan *pb.Operation_Event,
		log hclog.Logger,
	) *pb.Operation_Response {
		// Create our new response from our request.
		res, err := NewResponseFromRequest(req)
		if err != nil {
			log.Debug("failed to create response", RequestDebugArgs(req)...)
			event, err := NewEventFromResponse(res)
			if err == nil {
				events <- event
			} else {
				log.Debug("failed to event", RequestDebugArgs(req)...)
			}

			return res
		}

		res.Status = pb.Operation_STATUS_RUNNING
		res.Diagnostics = diagnostics.FromErr(err)
		failed, err := NewEventFromResponse(res)
		if err == nil {
			events <- failed
		} else {
			log.Error("failed to generate event for failure", "error", err)
		}

		return res
	}, err
}
