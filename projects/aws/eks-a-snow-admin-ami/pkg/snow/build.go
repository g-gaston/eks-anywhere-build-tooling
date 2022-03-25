package snow

import (
	"context"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"

	"github.com/aws/eks-anywhere-build-tooling/projects/aws/eks-a-snow-admin-ami/pkg/pipelines"
	"github.com/aws/eks-anywhere-build-tooling/projects/aws/eks-a-snow-admin-ami/pkg/session"
)

type AdminAMIInput struct {
	EKSAVersion    string
	EKSAReleaseURL string
	S3Bucket       string
}

func BuildAdminAMI(ctx context.Context, input *AdminAMIInput) error {
	log.Printf("Building AMI for EKSA %s from manifest [%s]\n", input.EKSAVersion, input.EKSAReleaseURL)
	pipeline := snowAdminAMIPipelineForEKSA(input)

	session, err := session.New(ctx)
	if err != nil {
		return err
	}

	if err = pipeline.UpdateRecipe(ctx, session); err != nil {
		return err
	}

	if err = pipeline.Run(ctx, session); err != nil {
		return err
	}

	if err = copyConvertedRAWToLatest(ctx, session, pipeline, input); err != nil {
		return err
	}

	return nil
}

func copyConvertedRAWToLatest(ctx context.Context, session *session.Session, pipeline *pipelines.Pipeline, input *AdminAMIInput) error {
	objects, err := listObjects(ctx, session, pipeline.S3Bucket, pipeline.S3Prefix)
	if err != nil {
		return err
	}

	var lastRawImage *s3.Object

	for _, o := range objects {
		if strings.HasSuffix(*o.Key, ".raw") {
			if lastRawImage == nil || o.LastModified.After(*lastRawImage.LastModified) {
				lastRawImage = o
			}
		}
	}

	if lastRawImage == nil {
		return errors.New("could not not find any converted raw image")
	}

	log.Printf("Last converted raw image [%s]", *lastRawImage.Key)

	latestBuiltImageURL := "s3://" + filepath.Join(pipeline.S3Bucket, *lastRawImage.Key)
	latestImageDst := "snow-admin-ami/latest/snow-admin.raw"
	latestImageURL := "s3://" + filepath.Join(pipeline.S3Bucket, latestImageDst)

	log.Printf("Copying converted last raw image to %s", latestImageDst)
	// Using the AWS cli here because copying objects bigger than 5GB requires using the the multipart API
	// This is doable with the golang sdk, but it requires a decent amount of work
	// The AWS cli must implement this logic already under the hood, so taking advantage of that
	// The drawback is now this code is dependent on having the cli installed in path
	if err = exec.CommandContext(ctx, "aws", "s3", "cp", "--acl", "public-read", latestBuiltImageURL, latestImageURL).Run(); err != nil {
		return errors.Wrap(err, "copying latest raw image to 'latest' folder")
	}

	versionDstFolder := filepath.Join("snow-admin-ami", input.EKSAVersion, "snow-admin.raw")
	versionDstURL := "s3://" + filepath.Join(pipeline.S3Bucket, versionDstFolder)
	log.Printf("Moving converted last raw image to %s", versionDstFolder)
	if err = exec.CommandContext(ctx, "aws", "s3", "mv", latestBuiltImageURL, versionDstURL).Run(); err != nil {
		return errors.Wrap(err, "copying latest raw image to 'latest' folder")
	}

	return nil
}

func getLastestBuiltRawImage(ctx context.Context, session *session.Session, pipeline *pipelines.Pipeline) (*s3.Object, error) {
	objects, err := listObjects(ctx, session, pipeline.S3Bucket, pipeline.S3Prefix)
	if err != nil {
		return nil, err
	}

	var lastRawImage *s3.Object

	for _, o := range objects {
		if strings.HasSuffix(*o.Key, ".raw") {
			if lastRawImage == nil || o.LastModified.After(*lastRawImage.LastModified) {
				lastRawImage = o
			}
		}
	}

	if lastRawImage == nil {
		return nil, errors.New("could not not find any converted raw image")
	}

	return lastRawImage, nil
}

func listObjects(ctx context.Context, session *session.Session, bucket string, prefix string) (listedObjects []*s3.Object, err error) {
	var nextToken *string
	var objects []*s3.Object

	s := s3.New(session)
	log.Printf("Listing objects in bucket [%s] with prefix [%s]", bucket, prefix)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),

		ContinuationToken: nextToken,
	}

	for {
		l, err := s.ListObjectsV2WithContext(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list objects")
		}
		objects = append(objects, l.Contents...)
		if !aws.BoolValue(l.IsTruncated) {
			break
		}
		nextToken = l.NextContinuationToken
		input.ContinuationToken = nextToken
	}
	return objects, nil
}
