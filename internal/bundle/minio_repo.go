package bundle

import (
	"context"
	"dspn-regogenerator/internal/config"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	opabundle "github.com/open-policy-agent/opa/v1/bundle"
)

// MinioRepository implements the [Repository] interface using Minio as the backend. It provides additional feataures like creating the associated bucket if it does not exist, setting the bucket policy, and checking if a bundle exists in the bucket.
type MinioRepository struct {
	client *minio.Client
	bucket string
}

// Read implements [Repository].
func (m *MinioRepository) Get(path string) (*Bundle, error) {
	reader, err := m.client.GetObject(context.Background(), m.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	if bundle, err := NewFromTarball(reader); err != nil {
		return nil, err
	} else {
		return bundle, nil
	}
}

// Write implements [Repository].
func (m *MinioRepository) Save(path string, bundle Bundle) error {
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()
		if err := opabundle.NewWriter(writer).Write(*bundle.bundle); err != nil {
			panic(err)
		}
	}()

	if _, err := m.client.PutObject(context.Background(), m.bucket, path, reader, -1, minio.PutObjectOptions{}); err != nil {
		return err
	} else {
		return nil
	}
}

// Create a bundle repository that uses Minio as the backend.
// The Minio client is created using the provided endpoint, access key, secret key, and secure flag.
func NewMinioRepository(endpoint, accessKey, secretKey string, secure bool, bucketName string) (*MinioRepository, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			accessKey,
			secretKey,
			"",
		),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}

	return &MinioRepository{
		client: client,
		bucket: bucketName,
	}, nil
}

// Create a bundle repository that uses Minio as the backend.
// The Minio client is created using the package configuration.
func NewMinioRepositoryFromConfig() (*MinioRepository, error) {
	return NewMinioRepository(
		config.MinioEndpoint,
		config.MinioAccessKey,
		config.MinioSecretKey,
		false,
		config.MinioBucket,
	)
}

var _ Repository = &MinioRepository{}

// Create the associated bucket if it does not exist (idempotent).
// The bucket is created with a policy that allows anonymous access to the bundle.
func (m *MinioRepository) CreateBucket(ctx context.Context) error {
	// Check if the bucket exists
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// Create the bucket
	err = m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
	if err != nil {
		return err
	}
	err = m.client.SetBucketPolicy(ctx, config.MinioBucket, fmt.Sprintf(anonymousPolicy, config.MinioBucket, config.MinioBucket))
	if err != nil {
		return err
	}
	return nil
}

const anonymousPolicy string = `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": [
                    "*"
                ]
            },
            "Action": [
                "s3:GetBucketLocation",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::%s"
            ]
        },
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": [
                    "*"
                ]
            },
            "Action": [
                "s3:GetObject"
            ],
            "Resource": [
                "arn:aws:s3:::%s/*"
            ]
        }
    ]
}`

// Check if the bundle with the provided name exists in the bucket.
// Returns true if the bundle exists, false otherwise.
// If the bucket does not exist or any unexpected error occurs, it returns an error.
// If the bucket exists but the bundle does not, it returns false and no error.
func (m *MinioRepository) BundleExists(ctx context.Context, bundleName string) (bool, error) {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, fmt.Errorf("bucket %s does not exist", m.bucket)
	}

	// Check if the bundle exists in the bucket
	objectCh := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{
		Prefix:    bundleName,
		Recursive: true,
	})
	for obj := range objectCh {
		if obj.Err != nil {
			return false, obj.Err
		}
		if obj.Key == bundleName {
			return true, nil
		}
	}
	return false, nil
}

func (m *MinioRepository) CopyBundle(ctx context.Context, srcBundleName, destBundleName string) error {
	src := minio.CopySrcOptions{
		Bucket: m.bucket,
		Object: srcBundleName,
	}
	dest := minio.CopyDestOptions{
		Bucket: m.bucket,
		Object: destBundleName,
	}
	if _, err := m.client.CopyObject(ctx, dest, src); err != nil {
		return fmt.Errorf("error copying bundle %s to %s: %v", srcBundleName, destBundleName, err)
	}
	return nil
}
