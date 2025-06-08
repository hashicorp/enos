package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
)

type mockAPIError struct {
	code string
}

func (e *mockAPIError) Error() string                 { return e.code }
func (e *mockAPIError) ErrorCode() string             { return e.code }
func (e *mockAPIError) ErrorMessage() string          { return e.code }
func (e *mockAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

type mockKeyManager struct {
	keyExists    bool
	deleteCalled bool
	createCalled bool
}

func (m *mockKeyManager) DescribeKeyPairs(ctx context.Context, input *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error) {
	if m.keyExists {
		return &ec2.DescribeKeyPairsOutput{
			KeyPairs: []types.KeyPairInfo{
				{KeyName: &input.KeyNames[0]},
			},
		}, nil
	}
	return nil, &mockAPIError{code: "InvalidKeyPair.NotFound"}
}

func (m *mockKeyManager) DeleteKeyPair(ctx context.Context, input *ec2.DeleteKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.DeleteKeyPairOutput, error) {
	m.deleteCalled = true
	m.keyExists = false
	return &ec2.DeleteKeyPairOutput{}, nil
}

func (m *mockKeyManager) CreateKeyPair(ctx context.Context, input *ec2.CreateKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.CreateKeyPairOutput, error) {
	m.createCalled = true
	key := "mock-private-key"
	return &ec2.CreateKeyPairOutput{
		KeyName:     input.KeyName,
		KeyMaterial: &key,
	}, nil
}

func TestKeyPairExists(t *testing.T) {
	tests := []struct {
		name      string
		client    describeKeyPairsAPI
		expect    bool
		expectErr bool
	}{
		{
			name:      "key exists",
			client:    &mockKeyManager{keyExists: true},
			expect:    true,
			expectErr: false,
		},
		{
			name:      "key does not exist",
			client:    &mockKeyManager{keyExists: false},
			expect:    false,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := keyPairExists(tt.client, "enos-ec2-key")
			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}
			if got != tt.expect {
				t.Errorf("expected: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func TestRunScenarios(t *testing.T) {
	tempHome := t.TempDir()
	sshPath := filepath.Join(tempHome, ".ssh")
	err := os.MkdirAll(sshPath, 0700)
	if err != nil {
		t.Fatalf("failed to create temp ssh dir: %v", err)
	}
	os.Setenv("HOME", tempHome)

	tests := []struct {
		name        string
		force       bool
		keyExists   bool
		expectWrite bool
	}{
		{"Key exists, no force", false, true, false},
		{"Key exists, force", true, true, true},
		{"Key does not exist", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockKeyManager{keyExists: tt.keyExists}
			err := Run(client, "enos-ec2-key", sshPath, tt.force)
			if err != nil && tt.expectWrite {
				t.Fatalf("expected no error, got: %v", err)
			}

			pemPath := filepath.Join(sshPath, "enos-ec2-key.pem")
			_, err = os.Stat(pemPath)
			if tt.expectWrite && os.IsNotExist(err) {
				t.Errorf("expected key file to be written, but it wasn't")
			}
		})
	}
}
