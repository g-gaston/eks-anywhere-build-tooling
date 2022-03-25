package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/aws/eks-anywhere-build-tooling/projects/aws/eks-a-snow-admin-ami/pkg/snow"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:          "setup",
	Short:        "Setup the necessary infrastructure to build a Snow EKS-A Admin AMI",
	SilenceUsage: true,
	RunE:         setup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().StringVar(&input.S3Bucket, "bucket", "", "S3 Bucket to store AMI converted to RAW format")
}

func setup(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	return snow.SetupAdminAMIPipeline(ctx, input)
}
