package http

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

func (h *HttpServer) matrixToCsv(
	mtrx []*common.MatrixResult,
	w http.ResponseWriter,
	r *http.Request,
	fileName string,
) {
	cfgBytes, _ := json.Marshal(mtrx[0].Config)
	cfgRaw := map[string]interface{}{}
	err := json.Unmarshal(cfgBytes, &cfgRaw)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	fields := []string{}
	for k := range cfgRaw {
		fields = append(fields, k)
	}

	resBytes, _ := json.Marshal(mtrx[0].ResultAvg)
	resRaw := map[string]interface{}{}
	err = json.Unmarshal(resBytes, &resRaw)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resultFields := []string{}
	for k := range resRaw {
		if !strings.Contains(k, "Percentiles") {
			resultFields = append(resultFields, k)
		}
	}

	latPercentiles := []float64{}
	percentiles := []string{}
	for _, p := range mtrx[0].ResultAvg.LatencyPercentiles {
		percentiles = append(
			percentiles,
			strings.ReplaceAll(fmt.Sprintf("latency%f", p.Bucket), ".", ""),
		)
		latPercentiles = append(latPercentiles, p.Bucket)
	}

	tpPercentiles := []float64{}
	for _, p := range mtrx[0].ResultAvg.ThroughputPercentiles {
		percentiles = append(
			percentiles,
			strings.ReplaceAll(fmt.Sprintf("throughput%f", p.Bucket), ".", ""),
		)
		tpPercentiles = append(tpPercentiles, p.Bucket)
	}

	header := append(append(fields, resultFields...), percentiles...)

	records := [][]string{
		header,
	}

	for _, mrow := range mtrx {
		values := []string{}
		cfgBytes, _ = json.Marshal(mrow.Config)
		cfgRaw = map[string]interface{}{}
		err = json.Unmarshal(cfgBytes, &cfgRaw)
		if err != nil {
			http.Error(
				w,
				"Internal Server Error",
				http.StatusInternalServerError,
			)
			return
		}

		resBytes, _ = json.Marshal(mrow.ResultAvg)
		resRaw = map[string]interface{}{}
		err = json.Unmarshal(resBytes, &resRaw)
		if err != nil {
			http.Error(
				w,
				"Internal Server Error",
				http.StatusInternalServerError,
			)
			return
		}

		for _, k := range fields {
			values = append(values, fmt.Sprintf("%v", cfgRaw[k]))
		}

		for _, k := range resultFields {
			values = append(values, fmt.Sprintf("%v", resRaw[k]))
		}
		for _, pct := range latPercentiles {
			for _, p := range mrow.ResultAvg.LatencyPercentiles {
				if p.Bucket == pct {
					values = append(values, fmt.Sprintf("%v", p.Value))
				}
			}
		}

		for _, pct := range tpPercentiles {
			for _, p := range mrow.ResultAvg.ThroughputPercentiles {
				if p.Bucket == pct {
					values = append(values, fmt.Sprintf("%v", p.Value))
				}
			}
		}

		records = append(records, values)
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().
		Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))

	cw := csv.NewWriter(w)
	for _, record := range records {
		if err := cw.Write(record); err != nil {
			logging.Errorf("error writing record to csv: %v", err)
		}
	}
	cw.Flush()
}
