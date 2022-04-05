package http

import (
	"encoding/json"
	"net/http"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

func (h *HttpServer) estimateChargeForTestRunHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	defer r.Body.Close()
	var tr common.TestRun

	err := json.NewDecoder(r.Body).Decode(&tr)
	if err != nil {
		logging.Errorf("Error parsing request: %s", err.Error())
		http.Error(w, "Request format incorrect", 500)
		return
	}

	runs := common.ExpandSweepRun(&tr, "")

	// Assume each instance will run for ~15 minutes and then calculate the
	// total hours for each instance size
	instanceDuration := map[string]float64{}
	for i := range runs {
		for j := range runs[i].Roles {
			tmpl, err := h.awsm.GetLaunchTemplate(
				runs[i].Roles[j].AwsLaunchTemplateID,
			)
			if err == nil {
				d, ok := instanceDuration[tmpl.InstanceType]
				if !ok {
					instanceDuration[tmpl.InstanceType] = 0.25
				} else {
					instanceDuration[tmpl.InstanceType] = d + 0.25
				}
			} else {
				http.Error(w, "Unknown template used", 500)
				return
			}

		}
	}

	writeJson(w, map[string]interface{}{
		"testruns":      len(runs),
		"instanceHours": instanceDuration,
	})
}
