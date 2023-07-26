package status

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/enos/proto/hashicorp/enos/v1/pb"
)

func TestOperationResponsesHandlesFailures(t *testing.T) {
	t.Parallel()

	for desc, test := range map[string]struct {
		res        *pb.OperationResponses
		failOnWarn bool
		shouldFail bool
	}{
		"decode has warnings with fail on warn": {
			res: &pb.OperationResponses{
				Decode: &pb.DecodeResponse{
					Diagnostics: []*pb.Diagnostic{
						{
							Severity: pb.Diagnostic_SEVERITY_WARNING,
							Summary:  "warning",
						},
					},
				},
			},
			failOnWarn: true,
			shouldFail: true,
		},
		"decode has warnings without fail on warn": {
			res: &pb.OperationResponses{
				Decode: &pb.DecodeResponse{
					Diagnostics: []*pb.Diagnostic{
						{
							Severity: pb.Diagnostic_SEVERITY_WARNING,
							Summary:  "warning",
						},
					},
				},
			},
		},
		"res has warnings with fail on warn": {
			res: &pb.OperationResponses{
				Diagnostics: []*pb.Diagnostic{
					{
						Severity: pb.Diagnostic_SEVERITY_WARNING,
						Summary:  "warning",
					},
				},
			},
			failOnWarn: true,
			shouldFail: true,
		},
		"res has warnings without fail on warn": {
			res: &pb.OperationResponses{
				Diagnostics: []*pb.Diagnostic{
					{
						Severity: pb.Diagnostic_SEVERITY_WARNING,
						Summary:  "warning",
					},
				},
			},
		},
		"res has ops with warning with fail on warn": {
			res: &pb.OperationResponses{
				Responses: []*pb.Operation_Response{
					{
						Diagnostics: []*pb.Diagnostic{
							{
								Severity: pb.Diagnostic_SEVERITY_WARNING,
								Summary:  "warning",
							},
						},
					},
				},
			},
			failOnWarn: true,
			shouldFail: true,
		},
		"res has ops with warning without fail on warn": {
			res: &pb.OperationResponses{
				Responses: []*pb.Operation_Response{
					{
						Diagnostics: []*pb.Diagnostic{
							{
								Severity: pb.Diagnostic_SEVERITY_WARNING,
								Summary:  "warning",
							},
						},
					},
				},
			},
		},
		"decode has errors": {
			res: &pb.OperationResponses{
				Decode: &pb.DecodeResponse{
					Diagnostics: []*pb.Diagnostic{
						{
							Severity: pb.Diagnostic_SEVERITY_ERROR,
							Summary:  "error",
						},
					},
				},
			},
			shouldFail: true,
		},
		"res has errors": {
			res: &pb.OperationResponses{
				Diagnostics: []*pb.Diagnostic{
					{
						Severity: pb.Diagnostic_SEVERITY_ERROR,
						Summary:  "error",
					},
				},
			},
			shouldFail: true,
		},
		"res has ops with error": {
			res: &pb.OperationResponses{
				Responses: []*pb.Operation_Response{
					{
						Diagnostics: []*pb.Diagnostic{
							{
								Severity: pb.Diagnostic_SEVERITY_ERROR,
								Summary:  "error",
							},
						},
					},
				},
			},
			shouldFail: true,
		},
	} {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

			err := OperationResponses(test.failOnWarn, test.res)
			if test.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
