package internal

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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

func UploadFile(ctx context.Context, server S3Server, filePath string, unique bool) (string, string, error) {
	debugf("upload start file=%s", filePath)
	debugf("endpoint=%s force_path_style=%t base_dir=%s path_format=%s acl=%s cdn_base_url=%s",
		server.Endpoint,
		server.ForcePathStyle,
		server.BaseDir,
		server.PathFormat,
		server.ACL,
		server.CDNBaseURL,
	)

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

	name, err := objectNameFromFile(file, filePath, unique)
	if err != nil {
		return "", "", err
	}

	key, err := buildObjectKey(server, name, time.Now())
	if err != nil {
		return "", "", err
	}
	debugf("object key=%s bucket=%s region=%s unique=%t", key, server.Bucket, server.Region, unique)

	if statter, ok := file.(interface{ Stat() (os.FileInfo, error) }); ok {
		if info, err := statter.Stat(); err == nil {
			debugf("file size=%d bytes", info.Size())
		}
	}

	contentType, err := sniffContentType(file)
	if err != nil {
		return "", "", err
	}
	debugf("content-type=%s", contentType)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(server.Bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	}
	if server.ACL != "" {
		input.ACL = types.ObjectCannedACL(server.ACL)
	}

	output, err := client.PutObject(ctx, input)
	if err != nil {
		return "", "", err
	}

	if output != nil {
		debugf("put-object etag=%s version_id=%s", aws.ToString(output.ETag), aws.ToString(output.VersionId))
	}
	cdnURL := buildCDNURL(server.CDNBaseURL, key)
	debugf("cdn_url=%s", cdnURL)

	return key, cdnURL, nil
}

func buildObjectKey(server S3Server, name string, now time.Time) (string, error) {
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
	parts = append(parts, name)
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

func objectNameFromFile(file io.ReadSeeker, filePath string, unique bool) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if unique {
		name, err := randomName(16)
		if err != nil {
			return "", err
		}
		return name + ext, nil
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)) + ext, nil
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
