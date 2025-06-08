package cmd

import (
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hashicorp/enos/internal/bootstrap"
	"github.com/spf13/cobra"
)

var force bool
var keypairName string
var sshDir string

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Initialize and configure required dependencies (e.g., AWS SSH key)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(cmd.Context())
		if err != nil {
			return err
		}
		client := ec2.NewFromConfig(cfg)
		return bootstrap.Run(client, keypairName, sshDir, force)
	},
}

func init() {
	bootstrapCmd.Flags().BoolVar(&force, "force", false, "Force creation even if key already exists")
	bootstrapCmd.Flags().StringVar(&keypairName, "keypair-name", "enos-ec2-key", "Name of the SSH key pair")
	bootstrapCmd.Flags().StringVar(&sshDir, "ssh-dir", filepath.Join(os.Getenv("HOME"), ".ssh"), "Directory to save the private key")
	rootCmd.AddCommand(bootstrapCmd)
}
