package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
)

var plotLock = sync.Mutex{}
var reportLock = sync.Mutex{}

type pythonPlotRequest struct {
	Request map[string]interface{} `json:"request"`
	Data    []*common.MatrixResult `json:"data"`
}

// error bars: stddev of samples divided by square root of sample count (std
// error)

type savedSweepPlot struct {
	ID      string                 `json:"id"`
	Title   string                 `json:"title"`
	Date    time.Time              `json:"date"`
	Request map[string]interface{} `json:"request"`
}

func (h *HttpServer) generateSweepPlot(r io.Reader) ([]byte, error) {
	// Ensure only one plot generation is running at the same time to prevent
	// server overload
	plotLock.Lock()
	defer plotLock.Unlock()
	req := map[string]interface{}{}
	err := json.NewDecoder(r).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("Error parsing request: %s", err.Error())
	}

	savePlot := false
	saveRaw, ok := req["save"]
	if ok {
		save, ok := saveRaw.(bool)
		if ok && save {
			savePlot = true
		}
	}

	sweepIDRaw, ok := req["sweepID"]
	if !ok {
		return nil, fmt.Errorf("Request is missing sweepID")
	}

	sweepID, ok := sweepIDRaw.(string)
	if !ok {
		return nil, fmt.Errorf("Request has non-string sweepID")
	}

	sweepIDs := []string{sweepID}
	if strings.Contains(sweepID, "|") {
		sweepIDs = strings.Split(sweepID, "|")
	}

	mtrx, _, _ := h.tr.GenerateSweepMatrix(sweepIDs)
	for i := range mtrx {
		mtrx[i].ResultDetails = mtrx[i].Results
	}

	pythonReq := pythonPlotRequest{
		Request: req,
		Data:    mtrx,
	}
	pythonRequestBytes, err := json.Marshal(pythonReq)
	if err != nil {
		return nil, fmt.Errorf(
			"Error marshalling python request: %s",
			err.Error(),
		)
	}
	exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, fmt.Errorf("Error determining exeDir: %v", err)
	}
	randomID, err := common.RandomID(12)
	if err != nil {
		return nil, fmt.Errorf("Error getting randomness: %v", err)
	}
	randName := fmt.Sprintf("plot-%s", randomID)
	plotInput := ""
	plotOutput := ""
	if savePlot {
		sweepPlotsDir := filepath.Join(
			common.DataDir(),
			"testruns",
			"sweep-plots",
			sweepID,
		)
		err := os.MkdirAll(sweepPlotsDir, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		plotInput = filepath.Join(
			sweepPlotsDir,
			fmt.Sprintf("%s.json", randName),
		)
		plotOutput = filepath.Join(
			sweepPlotsDir,
			fmt.Sprintf("%s.png", randName),
		)
	} else {
		plotInput = fmt.Sprintf("/tmp/%s.json", randName)
		plotOutput = fmt.Sprintf("/tmp/%s.png", randName)
		// If not saving, delete the temp files when done
		defer os.Remove(plotInput)
		defer os.Remove(plotOutput)
	}

	err = ioutil.WriteFile(plotInput, pythonRequestBytes, 0644)
	if err != nil {
		return nil, err
	}
	calcScript := filepath.Join(exeDir, "generate_sweep_plot.py")
	cmd := exec.Command("python3", calcScript, plotInput, plotOutput)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return ioutil.ReadFile(plotOutput)
}

type SweepPlotConfig struct {
	// The types of plots users can generate
	Types  []SweepPlotType  `json:"types"`
	Fields []SweepPlotField `json:"fields"`
	Axes   []SweepPlotAxis  `json:"axes"`
}

type SweepPlotType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SweepPlotField struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Eval      string `json:"eval"`
	ShortHand string `json:"shortHand"`
}

type SweepPlotAxis struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *HttpServer) getSweepPlotConfig() SweepPlotConfig {
	buckets := []float64{
		0.001,
		0.01,
		0.1,
		1,
		25,
		50,
		75,
		99,
		99.9,
		99.99,
		99.999,
	}
	bucketFields := make([]SweepPlotField, len(buckets)*2)
	for i, buck := range buckets {
		bucketFields[i] = SweepPlotField{
			ID: fmt.Sprintf("latencyPerc%f", buck),
			Eval: fmt.Sprintf(
				"next(item for item in r['result']['latencyPercentiles'] if item['bucket'] == %f)['value']*1000",
				buck,
			),
			Name:      fmt.Sprintf("Latency %f percentile (ms)", buck),
			ShortHand: fmt.Sprintf("Latency %f%%", buck),
		}
		bucketFields[i+len(buckets)] = SweepPlotField{
			ID: fmt.Sprintf("throughputPerc%f", buck),
			Eval: fmt.Sprintf(
				"next(item for item in r['result']['throughputPercentiles'] if item['bucket'] == %f)['value']",
				buck,
			),
			Name:      fmt.Sprintf("Throughput %f percentile", buck),
			ShortHand: fmt.Sprintf("Throughput %f%%", buck),
		}
	}

	return SweepPlotConfig{
		Types: []SweepPlotType{
			{ID: "line", Name: "Line plot"},
			{ID: "line-err", Name: "Line Plot With Error Bars"},
		},
		Fields: append([]SweepPlotField{
			{
				ID:   "clients",
				Eval: "r['config']['clients']",
				Name: "Number of clients",
			},
			{
				ID:   "atomizers",
				Eval: "r['config']['atomizers']",
				Name: "Number of atomizers",
			},
			{
				ID:   "watchtowers",
				Eval: "r['config']['watchtowers']",
				Name: "Number of watchtowers",
			},
			{
				ID:   "sentinels",
				Eval: "r['config']['sentinels']",
				Name: "Number of sentinels",
			},
			{
				ID:   "shards",
				Eval: "r['config']['shards']",
				Name: "Number of shards",
			},
			{
				ID:   "coordinators",
				Eval: "r['config']['coordinators']",
				Name: "Number of coordinators",
			},
			{
				ID:        "logicalShards",
				Eval:      "int(r['config']['shards'] / r['config']['shardReplicationFactor'])",
				Name:      "Number of logical shards",
				ShortHand: "Logical shards",
			},
			{
				ID:        "logicalCoordinators",
				Eval:      "int(r['config']['coordinators'] / r['config']['shardReplicationFactor'])",
				Name:      "Number of logical coordinators",
				ShortHand: "Logical coordinators",
			},
			{
				ID:        "faultTolerance",
				Eval:      "(int((r['config']['shardReplicationFactor']-1) / 2) if r['config']['architectureID'] == '2pc' else int((r['config']['atomizers']-1) / 2))",
				Name:      "Number of failures the system can tolerate",
				ShortHand: "Failure tolerance",
			},
			{
				ID:        "utxoSetSize",
				Eval:      "int(r['config']['preseedCount'])",
				Name:      "Number of seeded UTXOs",
				ShortHand: "Number of UTXOs",
			},
			{
				ID:   "hourOfDay",
				Eval: "int(r['config']['hourUTC'])",
				Name: "Hour of day (UTC)",
			},
			{
				ID:   "dayOfMonth",
				Eval: "int(r['config']['dayUTC'])",
				Name: "Day of month (UTC)",
			},
			{
				ID:   "arch",
				Eval: "{'2pc':'2PC','default':'Atomizer'}[r['config']['architectureID']]",
				Name: "Architecture",
			},
			{
				ID:        "inputs",
				Eval:      "r['config']['loadGenInputCount']",
				Name:      "Number of inputs per transaction",
				ShortHand: "Inputs",
			},
			{
				ID:        "outputs",
				Eval:      "r['config']['loadGenOutputCount']",
				Name:      "Number of outputs per transaction",
				ShortHand: "Outputs",
			},
			{
				ID:        "loadGenThreads",
				Eval:      "r['config']['loadGenThreads']",
				Name:      "Number of threads per load generator",
				ShortHand: "Load Generator Threads",
			},
			{
				ID:        "invalidTxRate",
				Eval:      "r['config']['invalidTxRate']",
				Name:      "Invalid transaction rate",
				ShortHand: "Invalid TX Rate",
			},
			{
				ID:        "fixedTxRate",
				Eval:      "r['config']['fixedTxRate']",
				Name:      "Fixed transaction rate",
				ShortHand: "Fixed TX Rate",
			},
			{
				ID:   "windowSize",
				Eval: "r['config']['windowSize']",
				Name: "Window size",
			},
			{
				ID:   "all-cli-txsec",
				Eval: "(r['config']['windowSize']/(r['config']['stxoCacheDepth']*r['config']['targetBlockInterval']))*r['config']['clients']",
				Name: "Client Traffic (TX/s)",
			},
			{
				ID:        "throughputAvg",
				Eval:      "r['result']['throughputAvg']",
				Name:      "Observed throughput average (TX/s)",
				ShortHand: "Throughput avg",
			},
			{
				ID:        "throughputAvg2",
				Eval:      "r['result']['throughputAvg2'] if 'throughputAvg2' in r['result'] else 0",
				Name:      "Observed throughput average (2) (TX/s)",
				ShortHand: "Throughput avg 2",
			},
			{
				ID:        "latencyAvg",
				Eval:      "r['result']['latencyAvg']*1000",
				Name:      "Observed latency average (ms)",
				ShortHand: "Latency avg",
			},
			{
				ID:   "throughputMax",
				Eval: "r['result']['throughputMax']",
				Name: "Observed throughput maximum (TX/s)",
			},
			{
				ID:   "latencyMax",
				Eval: "r['result']['latencyMax']*1000",
				Name: "Observed latency maximum (ms)",
			},
			{
				ID:   "throughputMin",
				Eval: "r['result']['throughputMin']",
				Name: "Observed throughput minimum (TX/s)",
			},
			{
				ID:   "latencyMin",
				Eval: "r['result']['latencyMin']*1000",
				Name: "Observed latency minimum (ms)",
			},
		}, bucketFields...),
		Axes: []SweepPlotAxis{
			{ID: "x", Name: "'X-Axis"},
			{ID: "y", Name: "Y-Axis"},
			{ID: "y2", Name: "Y-Axis 2"},
		},
	}
}
