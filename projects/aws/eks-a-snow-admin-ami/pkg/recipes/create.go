package recipes

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/imagebuilder"
	"github.com/pkg/errors"

	"github.com/aws/eks-anywhere-build-tooling/projects/aws/eks-a-snow-admin-ami/pkg/session"
)

func (r *Recipe) Create(ctx context.Context, session *session.Session) (arn string, err error) {
	componentsConfigurations := make([]*imagebuilder.ComponentConfiguration, 0, len(r.Components))
	for _, c := range r.Components {
		componentsConfigurations = append(componentsConfigurations, &imagebuilder.ComponentConfiguration{
			ComponentArn: aws.String(c.LastVersionARN(session.Account, session.Region())),
			Parameters:   c.parameterToAPI(),
		})
	}

	builder := imagebuilder.New(session)

	recipeARN := r.ARN(session.Account, session.Region())
	_, err = builder.GetImageRecipe(&imagebuilder.GetImageRecipeInput{ImageRecipeArn: aws.String(recipeARN)})

	if err != nil && !isNotFound(err) {
		return "", errors.Wrapf(err, "checking if recipe %s exists", r.Name)
	}

	if err == nil {
		if err = r.Delete(ctx, session); err != nil {
			return "", err
		}
	}

	log.Printf("Creating recipe [%s] version [%s]\n", r.Name, r.Version)
	_, err = builder.CreateImageRecipeWithContext(ctx, &imagebuilder.CreateImageRecipeInput{
		Name:            aws.String(r.Name),
		Description:     aws.String(r.Description),
		Components:      componentsConfigurations,
		ParentImage:     aws.String(session.ARNForRegion(r.ParentImage)),
		SemanticVersion: aws.String(r.Version),
		BlockDeviceMappings: []*imagebuilder.InstanceBlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &imagebuilder.EbsInstanceBlockDeviceSpecification{
					VolumeSize:          aws.Int64(32),
					DeleteOnTermination: aws.Bool(true),
					VolumeType:          aws.String("gp2"),
					
				},
			},
		},
	})
	if err != nil && !isAlreadyExist(err) {
		return "", errors.Wrapf(err, "failed creating recipe %s", r.Name)
	}

	return recipeARN, nil
}

func (r *Recipe) Delete(ctx context.Context, session *session.Session) error {
	log.Printf("Deleting existing recipe [%s] version [%s]\n", r.Name, r.Version)
	builder := imagebuilder.New(session)
	_, err := builder.DeleteImageRecipe(&imagebuilder.DeleteImageRecipeInput{ImageRecipeArn: aws.String(r.ARN(session.Account, session.Region()))})
	if err != nil {
		return errors.Wrapf(err, "deleting recipe %s", r.Name)
	}

	return nil
}

func isAlreadyExist(err error) bool {
	e := &imagebuilder.ResourceAlreadyExistsException{}
	return errors.As(err, &e)
}

func isNotFound(err error) bool {
	e := &imagebuilder.ResourceNotFoundException{}
	return errors.As(err, &e)
}
