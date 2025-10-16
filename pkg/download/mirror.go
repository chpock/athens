package download

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gomods/athens/pkg/download/mode"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/log"
)

// PathMirror URL.
const PathMirror = "/mirror/{archive:.+}"

// MirrorHandler implements GET /mirror/.
func MirrorHandler(dp Protocol, lggr log.Entry, df *mode.DownloadFile) http.Handler {
	const op errors.Op = "download.MirrorHandler"
	f := func(w http.ResponseWriter, r *http.Request) {
		archive, err := getMirrorParams(r, op)
		// mod, ver, err := getModuleParams(r, op)
		// if err != nil {
		// 	lggr.SystemErr(err)
		// 	w.WriteHeader(errors.Kind(err))
		// 	return
		// }
		// zip, err := dp.Zip(r.Context(), mod, ver)
		a, err := dp.Archive(r.Context(), archive)
		if err != nil {
			severityLevel := errors.Expect(err, errors.KindNotFound, errors.KindRedirect)
			err = errors.E(op, err, severityLevel)
			lggr.SystemErr(err)
			// if errors.Kind(err) == errors.KindRedirect {
			// 	url, err := getRedirectURL(df.URL(mod), r.URL.Path)
			// 	if err != nil {
			// 		lggr.SystemErr(err)
			// 		w.WriteHeader(errors.Kind(err))
			// 		return
			// 	}
			// 	http.Redirect(w, r, url, errors.KindRedirect)
			// 	return
			// }
			w.WriteHeader(errors.Kind(err))
			return
		}
		defer func() { _ = a.Close() }()

		w.Header().Set("Content-Type", "application/octet-stream")
		size := a.Size()
		if size > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}
		if r.Method == http.MethodHead {
			return
		}
		_, err = io.Copy(w, a)
		if err != nil {
			lggr.SystemErr(errors.E(op, errors.M(archive), err))
		}
	}
	return http.HandlerFunc(f)
}
