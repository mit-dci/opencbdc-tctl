package http

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/logging"
)

func (h *HttpServer) commandOutputHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	stream := params["stream"]
	cmdID := params["cmdID"]
	full := (params["full"] == "full")

	if full {
		w.Header().
			Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s_%s.txt", cmdID, stream))
	}
	w.Header().Add("Content-Type", "text/plain")

	tail := 1024
	if full {
		tail = -1
	}
	b, err := h.awsm.ReadFromS3(common.S3Download{
		SourceRegion: os.Getenv("AWS_REGION"),
		SourceBucket: os.Getenv("OUTPUTS_S3_BUCKET"),
		SourcePath: fmt.Sprintf(
			"command-outputs/%s/cmd_%s_std%s.txt",
			cmdID[:8],
			cmdID,
			stream,
		),
	}, tail)
	res := ""
	if err != nil {
		res = fmt.Sprintf("Error reading command stream: %v", err)
		logging.Errorf("Error reading output: %v", err)
	} else {
		res = string(b)
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(res))
	if err != nil {
		logging.Errorf("Error writing output: %v", err)
	}
}
