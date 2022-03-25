package snow

import (
	"context"

	"github.com/aws/eks-anywhere-build-tooling/projects/aws/eks-a-snow-admin-ami/pkg/session"
)

type SetupAMIInput struct {
	S3Bucket       string
}

func SetupAdminAMIPipeline(ctx context.Context, input *AdminAMIInput) error {
	pipeline := snowAdminAMIPipelineForEKSA(input)

	session, err := session.New(ctx)
	if err != nil {
		return err
	}

	return pipeline.Deploy(ctx, session)
}
