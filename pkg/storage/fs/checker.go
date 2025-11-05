package fs

import (
	"context"
	"os"
	"time"

	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
	"github.com/spf13/afero"
	"github.com/gomods/athens/pkg/log"
)

func (s *storageImpl) Exists(ctx context.Context, module, version string) (bool, error) {
	const op errors.Op = "fs.Exists"
	_, span := observ.StartSpan(ctx, op.String())
	defer span.End()

	if version == "" || version == "expired?" {
		archivePath := s.versionLocation(module, "")
		exists, err := afero.Exists(s.filesystem, archivePath)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, errors.E(op, errors.M(module), err)
		}
		if exists && version == "expired?" {
			info, err := s.filesystem.Stat(archivePath)
			if err != nil {
				log.EntryFromContext(ctx).Warnf("got nil Stat for archive '%s'", module)
				exists = false
			} else {
				modTime := info.ModTime()
				age := time.Since(modTime)
				secs := int(age.Round(time.Second).Seconds())
				log.EntryFromContext(ctx).Debugf(
					"archive '%s' has '%s' ModTime field and %d seconds age",
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
		return exists, nil
	}

	versionedPath := s.versionLocation(module, version)
	files, err := afero.ReadDir(s.filesystem, versionedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.E(op, errors.M(module), errors.V(version), err)
	}

	return len(files) == 3, nil
}
