package internal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const defaultPathFormat = "2006/01/02"

func UploadFile(ctx context.Context, server S3Server, filePath string) (string, string, error) {
	key, err := buildObjectKey(server, filePath, time.Now())
	if err != nil {
		return "", "", err
	}

	cfgOptions := []func(*config.LoadOptions) error{
		config.WithRegion(server.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			server.AccessKeyID,
			server.SecretAccessKey,
			server.SessionToken,
		)),
	}
	if server.Endpoint != "" {
		resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				return aws.Endpoint{URL: server.Endpoint, HostnameImmutable: true}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		cfgOptions = append(cfgOptions, config.WithEndpointResolverWithOptions(resolver))
	}
	cfg, err := config.LoadDefaultConfig(ctx, cfgOptions...)
	if err != nil {
		return "", "", err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = server.ForcePathStyle
	})

	file, err := osOpen(filePath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	contentType, err := sniffContentType(file)
	if err != nil {
		return "", "", err
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(server.Bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	}
	if server.ACL != "" {
		input.ACL = types.ObjectCannedACL(server.ACL)
	}

	if _, err := client.PutObject(ctx, input); err != nil {
		return "", "", err
	}

	return key, buildCDNURL(server.CDNBaseURL, key), nil
}

func buildObjectKey(server S3Server, filePath string, now time.Time) (string, error) {
	name, err := randomName(16)
	if err != nil {
		return "", err
	}
	format := strings.TrimSpace(server.PathFormat)
	if format == "" {
		format = defaultPathFormat
	}
	prefix := strings.TrimSpace(server.BaseDir)
	parts := make([]string, 0, 3)
	if prefix != "" {
		parts = append(parts, strings.Trim(prefix, "/"))
	}
	if format != "" {
		parts = append(parts, now.Format(format))
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	parts = append(parts, name+ext)
	return path.Join(parts...), nil
}

func randomName(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func buildCDNURL(base, key string) string {
	base = strings.TrimRight(base, "/")
	return fmt.Sprintf("%s/%s", base, key)
}

func sniffContentType(file io.ReadSeeker) (string, error) {
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}

type readSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

// osOpen is a small shim for testing or future extension.
var osOpen = func(path string) (readSeekCloser, error) {
	return os.Open(path)
}
