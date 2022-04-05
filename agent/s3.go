package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/logging"
	"github.com/mit-dci/opencbdc-tctl/wire"
)

// handleDeployFileFromS3 handles a DeployFileFromS3RequestMsg. This instructs
// the agent to download a file from S3 and write it to the agent's file system
func (a *Agent) handleDeployFileFromS3(
	msg *wire.DeployFileFromS3RequestMsg,
) (wire.Msg, error) {
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(msg.SourceRegion),
		config.WithEndpointResolver(common.S3customResolver()),
	)
	if err != nil {
		return nil, err
	}

	ret := &wire.DeployFileFromS3ResponseMsg{Success: true}
	logging.Debugf(
		"Received request to deploy from S3 bucket %s - file %s at %s",
		msg.SourceBucket,
		msg.SourcePath,
		msg.TargetPath,
	)
	// Compose the full target path from the environment directory and the
	// path specified in the request message
	targetFile := filepath.Join(
		environmentDir(msg.EnvironmentID),
		msg.TargetPath,
	)

	// Ensure the directory where the file must go exists
	err = os.MkdirAll(filepath.Dir(targetFile), 0755)
	if err != nil {
		return nil, err
	}

	// Open the target file for writing
	f, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Create the S3 client
	client := s3.NewFromConfig(
		cfg,
		func(opt *s3.Options) { opt.Region = msg.SourceRegion },
	)

	// Create a downloader
	downloader := manager.NewDownloader(client)
	downloader.PartSize = 5000000
	downloader.Concurrency = 30
	downloader.PartBodyMaxRetries = 500

	// Download the object
	_, err = downloader.Download(context.TODO(), f,
		&s3.GetObjectInput{
			Bucket: aws.String(msg.SourceBucket),
			Key:    aws.String(msg.SourcePath),
		})
	if err != nil {
		return nil, err
	}

	// Unpack the file if this has been requested
	if msg.Unpack {
		logging.Infof("Unpacking file from S3")
		err = common.TarExtractFlat(targetFile, msg.FlatUnpack, msg.UnpackNoDir)
		if err != nil {
			return nil, fmt.Errorf(
				"error extracting file %s: %v",
				msg.TargetPath,
				err,
			)
		}
		os.Remove(targetFile)
	}
	logging.Infof("Done")
	return ret, nil
}

// handleUploadFileToS3 handles the UploadFileToS3RequestMsg. This is the
// coordinator
// instructing the agent to upload a file from its file system to an S3 bucket.
// This
// is mostly used for uploading test run outputs and performance data at the end
// of
// a test run, as well as command outputs once commands have executed
// succesfully
func (a *Agent) handleUploadFileToS3(
	msg *wire.UploadFileToS3RequestMsg,
) (wire.Msg, error) {
	ret := &wire.UploadFileToS3ResponseMsg{Success: true}
	sourceFile := filepath.Join(
		environmentDir(msg.EnvironmentID),
		msg.SourcePath,
	)
	logging.Debugf(
		"Received request to upload to S3 bucket %s - file %s from %s",
		msg.TargetBucket,
		msg.TargetPath,
		msg.SourcePath,
	)
	err := a.uploadFileToS3(
		sourceFile,
		msg.TargetRegion,
		msg.TargetBucket,
		msg.TargetPath,
	)
	if err != nil {
		logging.Errorf("Error uploading file to S3: %v", err)
		return nil, err
	}
	logging.Infof("Done")
	return ret, nil
}

// uploadFileToS3 handles the actual uploading of the files to the given S3
// bucket
// this is used by the message handler as well as the command execution logic
func (a *Agent) uploadFileToS3(
	src string,
	targetRegion string,
	targetBucket string,
	targetFileName string,
) error {
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(targetRegion),
		config.WithEndpointResolver(common.S3customResolver()),
	)
	if err != nil {
		return err
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	bucket := aws.String(targetBucket)
	key := aws.String(targetFileName)
	client := s3.NewFromConfig(
		cfg,
		func(opt *s3.Options) { opt.Region = targetRegion },
	)

	uploader := manager.NewUploader(client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: bucket,
		Key:    key,
		Body:   f,
	})
	if err != nil {
		return err
	}
	logging.Infof("Uploaded %s to %s/%s", src, targetBucket, targetFileName)
	return nil
}
