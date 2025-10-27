package minioutil

import (
	"context"
	"dspn-regogenerator/internal/config"
	_ "embed"
	"strings"
	"text/template"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

//go:embed anonymousPolicy.json
var anonymousReadOnlyPolicyTemplate string

func NewFromConfig() (*miniosdk.Client, error) {
	return miniosdk.New(
		config.MinioEndpoint,
		&miniosdk.Options{
			Creds: credentials.NewStaticV4(config.MinioAccessKey, config.MinioSecretKey, ""),
		},
	)
}

// Check if the bucket with the provided name exists and it is anonymously accessible.
// If the bucket does not exist, it is created.
// If the bucket exists but it is not anonymously accessible, an error is returned.
func EnsureBucket(ctx context.Context, client *miniosdk.Client, bucketName string) error {
	builder := &strings.Builder{}
	template.Must(template.New("policy").Parse(anonymousReadOnlyPolicyTemplate)).Execute(builder, struct {
		BucketName string
	}{
		BucketName: bucketName,
	})
	anonymousReadOnlyPolicy := builder.String()

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if !exists {
		err = client.MakeBucket(ctx, bucketName, miniosdk.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}
	if policy, err := client.GetBucketPolicy(ctx, bucketName); err != nil {
		return err
	} else if policy == "" || policy != anonymousReadOnlyPolicy {
		if err := client.SetBucketPolicy(ctx, bucketName, anonymousReadOnlyPolicy); err != nil {
			return err
		}
	}

	return nil
}
