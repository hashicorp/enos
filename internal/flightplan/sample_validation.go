// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package flightplan

import (
	"context"
	"errors"
	"runtime"
	"sync"

	"github.com/hashicorp/enos/internal/diagnostics"
	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

// SampleValidationReq is a request to validate samples decode and have valid sub-frames.
type SampleValidationReq struct {
	Ws          *pb.Workspace
	Filter      *pb.Sample_Filter
	WorkerCount int
}

// SampleValidationOpt is a functional option for a NewSampleValidationReq.
type SampleValidationOpt func(*SampleValidationReq)

// NewSampleValidationReq takes optional NewSampleValidationOpt's and returns a new SampleValidationReq.
// Some validation is performed to ensure a valid request but not all validation can happen until
// the observation request is executed.
func NewSampleValidationReq(opts ...SampleValidationOpt) (*SampleValidationReq, error) {
	req := &SampleValidationReq{
		WorkerCount: runtime.NumCPU(),
	}

	for i := range opts {
		opts[i](req)
	}

	if req.Ws == nil {
		return nil, errors.New("cannot sample without a configured workspace")
	}

	if req.Ws.GetFlightplan() == nil {
		return nil, errors.New("cannot sample without a configured flightplan")
	}

	return req, nil
}

// Validate takes a sample observation request and validates that the sample can be decoded and that
// there are no empty sub-frames in the sample frame.
func (s *SampleValidationReq) Validate(ctx context.Context) *pb.DecodeResponse {
	fp, decRes := decodeSamples(ctx, s.Ws, s.Filter)
	if diagnostics.HasFailed(
		s.Ws.GetTfExecCfg().GetFailOnWarnings(),
		decRes.GetDiagnostics(),
	) {
		return decRes
	}

	samples := fp.Samples
	if s.Filter != nil && s.Filter.GetSample().GetId().GetName() != "" {
		sample, diags := findSampleByRef(fp, s.Filter.GetSample())
		decRes.Diagnostics = append(decRes.GetDiagnostics(), diags...)
		if diagnostics.HasFailed(
			s.Ws.GetTfExecCfg().GetFailOnWarnings(),
			decRes.GetDiagnostics(),
		) {
			return decRes
		}
		samples = []*Sample{sample}
	}

	var fDecRes *pb.DecodeResponse
	if s.WorkerCount < 2 || len(samples) < 4 {
		fDecRes = s.validateSamplesSerial(ctx, samples)
	} else {
		fDecRes = s.validateSamplesConcurrent(ctx, samples)
	}
	decRes.Diagnostics = append(decRes.GetDiagnostics(), fDecRes.GetDiagnostics()...)

	return decRes
}

// decodeScenariosConcurrent decodes scenario variants concurrently. This is for improved speeds
// when fully decoding lots of scenarios.
func (s *SampleValidationReq) validateSamplesConcurrent(ctx context.Context, samples []*Sample) *pb.DecodeResponse {
	res := &pb.DecodeResponse{}

	bossCtx, cancelBoss := context.WithCancel(ctx)
	defer cancelBoss()

	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	resCtx, cancelResCollector := context.WithCancel(ctx)
	defer cancelResCollector()

	workerWg := sync.WaitGroup{}
	jobsWg := sync.WaitGroup{}
	jobsC := make(chan *Sample)
	resC := make(chan *pb.DecodeResponse)

	defer func() {
		close(jobsC)
		close(resC)
	}()

	// Start the response collector that updates our aggregate response with diagnostics.
	collectResponses := func() {
		for {
			select {
			case vres := <-resC:
				res.Diagnostics = append(res.GetDiagnostics(), vres.GetDiagnostics()...)
				jobsWg.Done()
				if diagnostics.HasFailed(
					s.Ws.GetTfExecCfg().GetFailOnWarnings(),
					res.GetDiagnostics(),
				) {
					cancelBoss() // We ran into a validation issue so we'll cancel submitting more work
				}

				continue
			default:
			}

			select {
			case <-resCtx.Done():
				defer workerWg.Done()
				return
			case decRes := <-resC:
				res.Diagnostics = append(res.GetDiagnostics(), decRes.GetDiagnostics()...)
				jobsWg.Done()
				if diagnostics.HasFailed(
					s.Ws.GetTfExecCfg().GetFailOnWarnings(),
					res.GetDiagnostics(),
				) {
					cancelBoss() // We ran into a validation issue so we'll cancel submitting more work
				}
			}
		}
	}
	workerWg.Add(1)
	go collectResponses()

	// Start our validation workers. They'll validate a sample and send responses to the collector.
	startValidationWorker := func() {
		for {
			select {
			case <-workerCtx.Done():
				defer workerWg.Done()
				return
			case sample := <-jobsC:
				frame, decRes := sample.Frame(ctx, s.Ws, s.Filter)
				if diagnostics.HasFailed(
					s.Ws.GetTfExecCfg().GetFailOnWarnings(),
					decRes.GetDiagnostics(),
				) {
					resC <- decRes

					continue
				}

				err := frame.Validate()
				if err != nil {
					decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromErr(err)...)
				}

				resC <- decRes
			}
		}
	}

	workers := 2
	if s.WorkerCount > workers {
		workers = s.WorkerCount
	}
	for range workers {
		workerWg.Add(1)
		go startValidationWorker()
	}

	// Create our work for workers
OUTER:
	for _, sample := range samples {
		select {
		case <-bossCtx.Done():
			break OUTER
		default:
		}

		select {
		case <-bossCtx.Done():
			break OUTER
		default:
			jobsWg.Add(1)
			jobsC <- sample
		}
	}

	jobsWg.Wait()
	cancelWorkers()
	cancelResCollector()
	workerWg.Wait()

	return res
}

func (s *SampleValidationReq) validateSamplesSerial(ctx context.Context, samples []*Sample) *pb.DecodeResponse {
	decRes := &pb.DecodeResponse{}

	for _, sample := range samples {
		frame, framDecRes := sample.Frame(ctx, s.Ws, s.Filter)
		decRes.Diagnostics = append(decRes.GetDiagnostics(), framDecRes.GetDiagnostics()...)
		if diagnostics.HasFailed(
			s.Ws.GetTfExecCfg().GetFailOnWarnings(),
			decRes.GetDiagnostics(),
		) {
			return decRes
		}

		err := frame.Validate()
		if err != nil {
			decRes.Diagnostics = append(decRes.GetDiagnostics(), diagnostics.FromErr(err)...)
			return decRes
		}
	}

	return decRes
}

func WithSampleValidationReqWorkSpace(ws *pb.Workspace) SampleValidationOpt {
	return func(req *SampleValidationReq) {
		req.Ws = ws
	}
}

func WithSampleValidationReqFilter(f *pb.Sample_Filter) SampleValidationOpt {
	return func(req *SampleValidationReq) {
		req.Filter = f
	}
}

func WithSampleValidationWorkerCount(c int) SampleValidationOpt {
	return func(req *SampleValidationReq) {
		req.WorkerCount = c
	}
}
