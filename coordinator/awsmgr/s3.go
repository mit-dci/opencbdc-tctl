package awsmgr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

// getS3Default returns an S3 client for use in the default region.
func (am *AwsManager) getS3Default() (*s3.Client, error) {
	defaultRegion := os.Getenv("AWS_DEFAULT_REGION")
	if defaultRegion == "" {
		logging.Warnf(
			"Environment AWS_DEFAULT_REGION is not set, falling back to us-east-1",
		)
		defaultRegion = "us-east-1"
	}
	return am.getS3(defaultRegion)
}

// getS3 returns an S3 client for the given region. Since s3 clients are thread
// safe, we cache these clients per region and reuse them if they were
// previously created.
func (am *AwsManager) getS3(region string) (*s3.Client, error) {
	if !am.Enabled {
		return nil, errors.New("AWS not enabled")
	}

	am.s3ClientsLock.Lock()
	defer am.s3ClientsLock.Unlock()

	cli, ok := am.s3Clients[region]
	if !ok {
		opts := []func(*config.LoadOptions) error{
			config.WithRegion(region),
			config.WithEndpointResolver(common.S3customResolver()),
			defaultRetrier(),
		}
		cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
		if err != nil {
			return nil, err
		}
		cli = s3.NewFromConfig(
			cfg,
			func(opt *s3.Options) { opt.Region = region },
		)
		am.s3Clients[region] = cli
	}

	return cli, nil
}

// ReadFromS3 fetches an object as byte slice from an S3 bucket. If tail is -1
// it iwll read the entire object - if it is higher than -1 it will only read
// the last tail bytes of the object
func (am *AwsManager) ReadFromS3(
	d common.S3Download,
	tail int,
) ([]byte, error) {
	client, err := am.getS3(d.SourceRegion)
	if err != nil {
		return nil, err
	}
	byteRange := aws.String(fmt.Sprintf("-%d", tail))
	if *byteRange == "--1" {
		byteRange = nil
	}
	res, err := client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(d.SourceBucket),
		Key:    aws.String(d.SourcePath),
		Range:  byteRange,
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

// DownloadFromS3 downloads an object from an S3 bucket
func (am *AwsManager) DownloadFromS3(d common.S3Download) error {
	client, err := am.getS3(d.SourceRegion)
	if err != nil {
		return err
	}
	return am.downloadFromS3UsingClient(client, d)
}

// downloadFromS3UsingClient uses a specific s3 client instance to perform a
// download from S3
func (am *AwsManager) downloadFromS3UsingClient(
	client *s3.Client,
	d common.S3Download,
) error {
	// Ensure the directory where we should place the file exists (by trying to
	// create it and ignoring Already Exists errors)
	targetFile := d.TargetPath
	err := os.MkdirAll(filepath.Dir(targetFile), 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	// Open the output file for writing, creating it if it does not yet exist
	f, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	// Make sure to close the file whenever we return from this method
	defer f.Close()

	// Create a new downloader and download the given bucket and key to the file
	// stream we just created for the output file
	bucket := aws.String(d.SourceBucket)
	key := aws.String(d.SourcePath)
	downloader := manager.NewDownloader(client)
	downloader.PartSize = 5000000
	downloader.Concurrency = 10
	downloader.PartBodyMaxRetries = 500
	_, err = downloader.Download(context.Background(), f,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		})
	if err != nil {
		logging.Warnf(
			"Error downloading %s/%s to %s: %v",
			d.SourceBucket,
			d.SourcePath,
			d.TargetPath,
		)
	}
	return err
}

// FileExistsOnS3 checks if a file in the given region, bucket and path exists
// in S3. The return value indicates existence (true/false). In case an error
// occurs during checking the file's existence, it will be returned
func (am *AwsManager) FileExistsOnS3(
	region, bucket, path string,
) (bool, error) {
	client, err := am.getS3(region)
	if err != nil {
		return false, err
	}
	bkt := aws.String(bucket)
	key := aws.String(path)
	_, err = client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: bkt,
		Key:    key,
	})
	if err != nil {
		var responseError *awshttp.ResponseError
		if errors.As(err, &responseError) &&
			responseError.ResponseError.HTTPStatusCode() == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// UploadToS3IfNotExists will check a file's existence and skip uploading if the
// file already exists
func (am *AwsManager) UploadToS3IfNotExists(d common.S3Upload) error {
	exists, err := am.FileExistsOnS3(
		d.TargetRegion,
		d.TargetBucket,
		d.TargetPath,
	)
	if err != nil {
		return err
	}
	if !exists {
		return am.UploadToS3(d)
	}
	return nil
}

// UploadToS3 will upload a local file to a bucket and key in S3
func (am *AwsManager) UploadToS3(d common.S3Upload) error {
	// Create an S3 client in the correct region
	client, err := am.getS3(d.TargetRegion)
	if err != nil {
		return err
	}
	// Open the file we need to upload
	f, err := os.Open(d.SourcePath)
	if err != nil {
		return err
	}
	// Ensure to close the file when we exit this function
	defer f.Close()

	// Create a new uploader and have it read from the filestream we just opened
	// to push the bytes to the S3 object
	bucket := aws.String(d.TargetBucket)
	key := aws.String(d.TargetPath)
	uploader := manager.NewUploader(client)
	_, err = uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: bucket,
		Key:    key,
		Body:   f,
	})
	if err != nil {
		return err
	}
	logging.Infof(
		"Uploaded %s to %s/%s",
		d.SourcePath,
		d.TargetBucket,
		d.TargetPath,
	)
	return nil
}

// DownloadMultipleFromS3 will run multiple downloads in parallel to speed up
// the process when downloading many files. The number of CPUs in the system is
// used as a parallelism limit. If an error occurred with one of the downloads,
// the function will return the first error that occurred.
func (am *AwsManager) DownloadMultipleFromS3(
	downloads []common.S3Download,
) error {
	errChan := make(chan error, 10)
	wg := sync.WaitGroup{}
	wg.Add(len(downloads))
	dlChan := make(chan common.S3Download, 100)

	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for dl := range dlChan {
				err := am.DownloadFromS3(dl)
				if err != nil {
					if dl.Retries == 0 {
						logging.Errorf(
							"Failed to download from S3, no more retries: %v %v",
							dl,
							err,
						)
						errChan <- err
					} else {
						dl.Retries--
						logging.Warnf("Failed to download from S3 - retrying: %v %v", dl, err)
						dlChan <- dl
						continue // Prevent wg.Done() which leads to negative waitgroup counter
					}
				}
				wg.Done()
			}
		}()
	}

	var dlErr error
	for _, d := range downloads {
		dlChan <- d
		select {
		case dlErr = <-errChan:
		default:
		}
		if dlErr != nil {
			break
		}
	}
	wg.Wait()
	close(dlChan)
	return dlErr
}

// ListObjectsInS3 will scan a bucket with a particular prefix and return all
// the objects that match the prefix
func (am *AwsManager) ListObjectsInS3(
	region, bucket, prefix string,
) ([]string, error) {
	client, err := am.getS3(region)
	if err != nil {
		return nil, err
	}
	bkt := aws.String(bucket)
	pfx := aws.String(prefix)

	var continuationToken *string
	continuationToken = nil

	response := make([]string, 0)

	for {
		res, err := client.ListObjectsV2(
			context.Background(),
			&s3.ListObjectsV2Input{
				Bucket:            bkt,
				Prefix:            pfx,
				MaxKeys:           1000,
				ContinuationToken: continuationToken,
			},
		)
		if err != nil {
			return nil, err
		}

		for _, o := range res.Contents {
			response = append(response, *o.Key)
		}

		continuationToken = res.NextContinuationToken
		if continuationToken == nil {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
	return response, nil
}
