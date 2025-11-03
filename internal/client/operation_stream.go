// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/hashicorp/enos/internal/diagnostics"
	uipkg "github.com/hashicorp/enos/internal/ui"
	"github.com/hashicorp/enos/internal/ui/status"
	pb "github.com/hashicorp/enos/pb/hashicorp/enos/v1"
)

type opRes interface {
	GetDecode() *pb.DecodeResponse
	GetDiagnostics() []*pb.Diagnostic
	GetOperations() []*pb.Ref_Operation
}

// StreamOperations handles streaming responses from the server and writing
// their responses to the UI.
func (c *Connection) StreamOperations(
	ctx context.Context,
	opRes opRes,
	ui uipkg.View,
) *pb.OperationResponses {
	res := &pb.OperationResponses{
		Decode:      opRes.GetDecode(),
		Diagnostics: opRes.GetDiagnostics(),
		Responses:   []*pb.Operation_Response{},
	}

	if status.HasFailed(ui.Settings().GetFailOnWarnings(),
		res,
		res.GetDecode(),
	) {
		return res
	}

	var moreDiags []*pb.Diagnostic
	res.Responses, moreDiags = c.streamResponses(ctx, opRes.GetOperations(), ui)
	res.Diagnostics = append(res.GetDiagnostics(), moreDiags...)

	return res
}

// streamResponses takes a context, workspace, and slice of operation references
// and streams operation events to the ui. It will return a slice of operation
// responses for each stream that completes.
//
//nolint:cyclop // This could probably be refactored to be less complex but right now its inlined.
func (c *Connection) streamResponses(
	ctx context.Context,
	refs []*pb.Ref_Operation,
	ui uipkg.View,
) ([]*pb.Operation_Response, []*pb.Diagnostic) {
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	res := []*pb.Operation_Response{}

	diags := []*pb.Diagnostic{}
	diagC := make(chan *pb.Diagnostic)
	doneC := make(chan struct{})
	diagWg := sync.WaitGroup{}

	select {
	case <-ctx.Done():
		cause := context.Cause(ctx)
		if cause != nil && cause != context.Canceled {
			// We have a custom error or our deadline was exceeded.
			diags = append(diags, diagnostics.FromErr(cause)...)
		}

		return res, diags
	default:
	}

	// Start the error diagnostic routine. This collects diagnostics generated at the client level.
	// that are unexpected. Per stream request diagnostics will be scoped to each ref.
	diagWg.Add(1)
	go func() {
		defer diagWg.Done()

		drainDiags := func() {
			for {
				select {
				case diag := <-diagC:
					diags = append(diags, diag)

					continue
				default:
				}

				return
			}
		}

		checkCtx := func() {
			err := ctx.Err()
			if err == nil {
				return
			}
			cause := context.Cause(ctx)
			if cause != context.Canceled {
				// We have a custom error or our deadline was exceeded.
				diags = append(diags, diagnostics.FromErr(cause)...)
			}
		}

		for {
			select {
			case diag := <-diagC:
				diags = append(diags, diag)

				continue
			default:
			}

			select {
			case diag := <-diagC:
				diags = append(diags, diag)

				continue
			case <-ctx.Done():
				drainDiags()
				checkCtx()

				return
			case <-doneC:
				drainDiags()
				checkCtx()

				return
			}
		}
	}()

	for _, ref := range refs {
		wg.Add(1)
		go func() {
			defer wg.Done()

			c.Trace("starting event stream", "operation_id", ref.GetId())
			stream, err := c.Client.OperationEventStream(
				ctx,
				&pb.OperationEventStreamRequest{
					Op: ref,
				},
			)
			if err != nil {
				c.Log.Error("failed to start event stream",
					"operation_id", ref.GetId(),
					"error", err,
				)
				mu.Lock()
				res = append(res, &pb.Operation_Response{
					Diagnostics: diagnostics.FromErr(err),
					Op:          ref,
				})
				mu.Unlock()

				return
			}

			eventC := make(chan *pb.Operation_Event)
			errC := make(chan error)
			ticker := time.NewTicker(5 * time.Second)
			var lastEvent *pb.Operation_Event

			// Start the operation event stream poller
			go func() {
				for {
					eventRes, err := stream.Recv()
					if err != nil {
						errC <- err

						return
					}

					if eventRes == nil {
						continue
					}

					eventC <- eventRes.GetEvent()
				}
			}()

		LOOP:
			for {
				select {
				case <-ctx.Done():

					break LOOP
				case err := <-errC:
					if err != nil && err != io.EOF {
						err2 := ui.ShowError(err)
						if err2 != nil {
							c.Log.Error("failed to show error",
								"operation_id", ref.GetId(),
								"parent_error", err,
								"child_error", err2,
							)
						}
					}

					break LOOP
				case <-ticker.C:
					if lastEvent != nil {
						c.Trace("showing last event")
						ui.ShowOperationEvent(lastEvent)
					}
				case event := <-eventC:
					c.Trace("received event",
						"operation_id", ref.GetId(),
						"published_at", event.GetPublishedAt(),
					)

					if lastEvent == nil || event.GetPublishedAt().AsTime().After(lastEvent.GetPublishedAt().AsTime()) {
						// Because our events are not guaranteed to be in order
						// we'll only update our last event if it was published
						// more recently. This ensures that when we "replay"
						// events while waiting we always show the most recent.
						lastEvent = event
						ui.ShowOperationEvent(event)
						ticker.Reset(5 * time.Second)
					}
				}
			}

			// If the stream is closed or something went wrong then
			// we'll get and return the status of the operation.
			opRes, err := c.Client.Operation(
				ctx,
				&pb.OperationRequest{
					Op: ref,
				},
			)
			if err != nil && opRes.GetDiagnostics() != nil {
				opRes.Response.Diagnostics = append(
					opRes.GetResponse().GetDiagnostics(),
					diagnostics.FromErr(err)...,
				)
			}
			mu.Lock()
			res = append(res, opRes.GetResponse())
			mu.Unlock()
		}()
	}

	wg.Wait()
	close(doneC)
	diagWg.Wait()

	return res, diags
}
