// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/smithy-go"
)

type describeKeyPairsAPI interface {
	DescribeKeyPairs(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error)
	DeleteKeyPair(ctx context.Context, params *ec2.DeleteKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.DeleteKeyPairOutput, error)
	CreateKeyPair(ctx context.Context, params *ec2.CreateKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.CreateKeyPairOutput, error)
}

func Run(client describeKeyPairsAPI, keypairName, sshDir string, force bool) error {
	exists, err := keyPairExists(client, keypairName)
	if err != nil {
		return err
	}

	if exists && !force {
		fmt.Printf("Key pair %q already exists. Use --force to recreate.\n", keypairName)
		return nil
	}

	if exists && force {
		fmt.Printf("Deleting existing key pair...\n")
		_, _ = client.DeleteKeyPair(context.TODO(), &ec2.DeleteKeyPairInput{
			KeyName: aws.String(keypairName),
		})
	}

	fmt.Printf("Creating new key pair %q...\n", keypairName)
	out, err := client.CreateKeyPair(context.TODO(), &ec2.CreateKeyPairInput{
		KeyName: aws.String(keypairName),
	})
	if err != nil {
		return fmt.Errorf("failed to create key pair: %w", err)
	}

	err = os.MkdirAll(sshDir, 0o700)
	if err != nil {
		return fmt.Errorf("failed to create ssh directory: %w", err)
	}

	expandedPath := filepath.Join(sshDir, keypairName+".pem")
	err = os.WriteFile(expandedPath, []byte(*out.KeyMaterial), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	fmt.Printf(`
Key pair %q created and saved to:

    %s

Please update your enos-local.vars.hcl with the following:

    aws_ssh_keypair_name       = "%s"
    aws_ssh_private_key_path   = "%s"

`, keypairName, expandedPath, keypairName, expandedPath)

	return nil
}

func keyPairExists(client describeKeyPairsAPI, name string) (bool, error) {
	_, err := client.DescribeKeyPairs(context.TODO(), &ec2.DescribeKeyPairsInput{
		KeyNames: []string{name},
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "InvalidKeyPair.NotFound" {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
