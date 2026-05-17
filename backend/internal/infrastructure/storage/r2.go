package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "mmo/pkg/config"
)

type R2Client struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewR2(cfg appconfig.R2Config) (*R2Client, error) {
	if cfg.AccountID == "" {
		// Return a no-op client for local dev without R2 credentials
		return &R2Client{publicURL: cfg.PublicURL, bucket: cfg.BucketName}, nil
	}

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID),
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("load r2 config: %w", err)
	}

	return &R2Client{
		client:    s3.NewFromConfig(awsCfg),
		bucket:    cfg.BucketName,
		publicURL: cfg.PublicURL,
	}, nil
}

func (r *R2Client) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	if r.client == nil {
		return nil
	}
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	return err
}

func (r *R2Client) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if r.client == nil {
		return fmt.Sprintf("/local-media/%s", key), nil
	}
	presignClient := s3.NewPresignClient(r.client)
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (r *R2Client) Delete(ctx context.Context, key string) error {
	if r.client == nil {
		return nil
	}
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (r *R2Client) PublicURL(key string) string {
	if r.publicURL == "" {
		return "/local-media/" + key
	}
	return r.publicURL + "/" + key
}
