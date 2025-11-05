package s3

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
	"github.com/gomods/athens/pkg/log"
)

// Exists implements the (./pkg/storage).Checker interface
// returning true if the module at version exists in storage.
func (s *Storage) Exists(ctx context.Context, module, version string) (bool, error) {
	const op errors.Op = "s3.Exists"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()

	if version == "" || version == "expired?" {
		o, err := s.s3API.HeadObject(
			ctx,
			&s3.HeadObjectInput{
				Bucket: aws.String(s.bucket),
				Key:    aws.String(module),
			})
		exists := false
		if err == nil {
			exists = true
		} else {
			var aerr smithy.APIError
			if errors.AsErr(err, &aerr) && aerr.ErrorCode() == "NotFound" {
				err = nil
				exists = false
			}
		}
		if exists && version == "expired?" {
			if o.LastModified == nil {
				log.EntryFromContext(ctx).Warnf("got nil LastModified for archive '%s'", module)
				exists = false
			} else {
				modTime := *o.LastModified
				age := time.Since(modTime)
				secs := int(age.Round(time.Second).Seconds())
				log.EntryFromContext(ctx).Debugf(
					"archive '%s' has '%s' LastModified field and %d seconds age",
					module, modTime.Format(time.RFC3339), secs,
				)
				// 3600 seconds - 1 hour
				// if secs > 3600 {
				if secs > 3600 {
					log.EntryFromContext(ctx).Infof(
						"archive '%s' has %d seconds age and must be updated",
						module, secs,
					)
					// exists is already true here
				} else {
					exists = false
				}
			}
		}
		return exists, err
	}

	files := []string{"info", "mod", "zip"}
	errChan := make(chan error, len(files))
	defer close(errChan)
	cancelingCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			_, err := s.s3API.HeadObject(
				cancelingCtx,
				&s3.HeadObjectInput{
					Bucket: aws.String(s.bucket),
					Key:    aws.String(config.PackageVersionedName(module, version, file)),
				})
			errChan <- err
		}(file)
	}
	exists := true
	var err error
	for range files {
		err = <-errChan
		if err == nil {
			continue
		}
		var aerr smithy.APIError
		if errors.AsErr(err, &aerr) && aerr.ErrorCode() == "NotFound" {
			err = nil
			exists = false
		}
		break
	}
	cancel()
	wg.Wait()
	return exists, err
}
