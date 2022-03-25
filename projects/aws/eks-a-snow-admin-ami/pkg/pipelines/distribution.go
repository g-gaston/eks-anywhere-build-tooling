package pipelines

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/imagebuilder"
	"github.com/pkg/errors"

	"github.com/aws/eks-anywhere-build-tooling/projects/aws/eks-a-snow-admin-ami/pkg/session"
)

func (p *Pipeline) setupDistributionConfig(ctx context.Context, session *session.Session) (arn string, err error) {
	builder := imagebuilder.New(session)
	name := fmt.Sprintf("%s-distribution-config", p.ValidNameForARN())

	_, err = builder.CreateDistributionConfigurationWithContext(ctx, &imagebuilder.CreateDistributionConfigurationInput{
		Description: aws.String("Distribute AMI to bucket in RAW format"),
		Name:        aws.String(name),
		Distributions: []*imagebuilder.Distribution{
			{
				Region: session.Config.Region,
				S3ExportConfiguration: &imagebuilder.S3ExportConfiguration{
					RoleName:        aws.String("vmimport"),
					DiskImageFormat: aws.String(p.ConversionFormat),
					S3Bucket:        aws.String(p.S3Bucket),
					S3Prefix:        aws.String(p.S3Prefix),
				},
			},
		},
	})

	if err != nil && !isAlreadyExist(err) {
		return "", errors.Wrapf(err, "failed creating distribution config for pipeline %s", p.Name)
	}

	log.Printf("Searching for distribution config [%s] for pipeline [%s]\n", name, p.Name)
	distributionConfigList, err := builder.ListDistributionConfigurationsWithContext(ctx, &imagebuilder.ListDistributionConfigurationsInput{
		Filters: []*imagebuilder.Filter{
			{
				Name:   aws.String("name"),
				Values: aws.StringSlice([]string{name}),
			},
		},
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed searching for infra config [%s] ARN for pipeline [%s]", name, p.Name)
	}

	if len(distributionConfigList.DistributionConfigurationSummaryList) == 0 {
		return "", errors.Errorf("could not find infra config [%s] %v", name, distributionConfigList)
	}

	for _, d := range distributionConfigList.DistributionConfigurationSummaryList {
		if *d.Name == name {
			return *d.Arn, nil
		}
	}

	return "", errors.Errorf("no matching distribution config [%s] %v", name, distributionConfigList)
}
