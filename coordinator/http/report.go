package http

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

type reportDefinition struct {
	Title      string `json:"title"`
	Definition string `json:"definition"`
}

type configTableDefinition struct {
	SweepID string `json:"sweepID"`
}

func (h *HttpServer) generateConfigTable(input string) string {
	var tableDef configTableDefinition

	r := bytes.NewReader([]byte(input))
	err := json.NewDecoder(r).Decode(&tableDef)
	if err != nil {
		return "<b>Error generating config table</b>"
	}

	sweepIDs := []string{tableDef.SweepID}
	if strings.Contains(tableDef.SweepID, "|") {
		sweepIDs = strings.Split(tableDef.SweepID, "|")
	}

	mtrx, minDate, maxDate := h.tr.GenerateSweepMatrix(sweepIDs)
	for i := range mtrx {
		mtrx[i].ResultDetails = mtrx[i].Results
	}

	var raw map[string]interface{}
	propValCounts := map[string]int{}
	for i := range mtrx {
		b, err := json.Marshal(mtrx[i].Config)
		if err == nil {
			raw = map[string]interface{}{}
			err = json.Unmarshal(b, &raw)
			if err == nil {
				for k, v := range raw {
					key := fmt.Sprintf("%v|%v", k, v)
					cnt := propValCounts[key]
					propValCounts[key] = cnt + 1
				}
			}
		}
	}

	var buf bytes.Buffer

	// Inject link to sweep matrix
	propValCounts[fmt.Sprintf(`sweepMatrix|<a target="_blank" href="https://test-controller.hmltn.io/testruns/sweepMatrix/%s">Click here</a>`, strings.ReplaceAll(tableDef.SweepID, "|", "%7C"))] = len(
		mtrx,
	)

	commonPropVals := map[string]bool{}
	for k, v := range propValCounts {
		if v == len(mtrx) {
			commonPropVals[k] = true
		}
	}

	commonPropVals[fmt.Sprintf("minDate|%s", minDate.Format("01/02/2006 15:04"))] = true
	commonPropVals[fmt.Sprintf("maxDate|%s", maxDate.Format("01/02/2006 15:04"))] = true

	_, err = io.WriteString(&buf, "<table class=\"config\"><tbody>")
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	_, err = io.WriteString(
		&buf,
		"<tr ><td align=\"center\" colspan=\"4\" class=\"titleRow\"><b>System composition</b></td></tr>",
	)
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	_, err = io.WriteString(
		&buf,
		"<tr><td width=\"50%\"><b><u>Role</u></b></td><td align=\"center\" width=\"16%\">Count</td><td align=\"center\" width=\"17%\">CPUs</td><td align=\"center\" width=\"17%\">RAM</td></tr>",
	)
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	roles := []string{
		"atomizer",
		"coordinator",
		"shard",
		"sentinel",
		"watchtower",
		"client",
	}

	for _, r := range roles {
		count := -1
		ram := 0
		cpu := 0
		for k := range commonPropVals {
			prop := strings.Split(k, "|")
			if prop[0] == fmt.Sprintf("%ss", r) {
				count, _ = strconv.Atoi(prop[1])
			} else if prop[0] == fmt.Sprintf("%sCPU", r) {
				cpu, _ = strconv.Atoi(prop[1])
			} else if prop[0] == fmt.Sprintf("%sRAM", r) {
				ram, _ = strconv.Atoi(prop[1])
			}
		}
		if ram > 0 {
			countText := "Varying"
			if count > 0 {
				countText = fmt.Sprintf("%d", count)
			}
			_, err = io.WriteString(
				&buf,
				fmt.Sprintf(
					"<tr><td width=\"50%%\">%s%ss</td><td align=\"center\" width=\"16%%\">%s</td><td align=\"center\" width=\"17%%\">%d vCPU</td><td align=\"center\" width=\"17%%\">%d MB</td></tr>",
					strings.ToUpper(r[0:1]),
					r[1:],
					countText,
					cpu,
					ram,
				),
			)
			if err != nil {
				return "<b>Error generating config table</b>"
			}
		}
	}

	for k := range commonPropVals {
		prop := strings.Split(k, "|")
		if prop[0] == "multiRegion" {
			_, err = io.WriteString(
				&buf,
				fmt.Sprintf(
					"<tr><td width=\"50%%\"><b>Multi-region:</b></td><td colspan=\"3\">%s</td></tr>",
					prop[1],
				),
			)
			if err != nil {
				return "<b>Error generating config table</b>"
			}
		}
	}

	_, err = io.WriteString(&buf, "</tbody></table>")
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	_, err = io.WriteString(&buf, "<table class=\"config\"><tbody>")
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	_, err = io.WriteString(
		&buf,
		"<tr ><td align=\"center\" colspan=\"6\" class=\"titleRow\"><b>RAFT Parameters</b></td></tr>",
	)
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	raftParams := map[string]string{
		"raftMaxBatch":         "Max Batch",
		"snapshotDistance":     "Snapshot distance",
		"electionTimeoutUpper": "Upper Election Timeout (ms)",
		"electionTimeoutLower": "Lower Election Timeout (ms)",
		"heartbeat":            "Heart Beat (ms)",
	}
	props := -1
	for key, propName := range raftParams {
		for k := range commonPropVals {
			prop := strings.Split(k, "|")
			if key == prop[0] {
				props++
				if props%3 == 0 {
					if props > 0 {
						_, err = io.WriteString(&buf, "</tr>")
						if err != nil {
							return "<b>Error generating config table</b>"
						}
					}
					_, err = io.WriteString(&buf, "<tr>")
					if err != nil {
						return "<b>Error generating config table</b>"
					}
				}
				_, err = io.WriteString(
					&buf,
					fmt.Sprintf(
						"<td width=\"20%%\"><b>%s:</b></td><td width=\"13%%\">%s</td>",
						propName,
						prop[1],
					),
				)
				if err != nil {
					return "<b>Error generating config table</b>"
				}
				break
			}
		}
	}
	_, err = io.WriteString(&buf, "</tbody></table>")
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	_, err = io.WriteString(&buf, "<table class=\"config\"><tbody>")
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	_, err = io.WriteString(
		&buf,
		"<tr><td align=\"center\" colspan=\"6\" class=\"titleRow\"><b>Load generation</b></td></tr>",
	)
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	clientParams := map[string]string{
		"loadGenInputCount":  "Fixed Transactions Input Count",
		"loadGenOutputCount": "Fixed Transactions Output Count",
		"fixedTxRate":        "Fixed Transaction Rate",
		"invalidTxRate":      "Invalid Transaction Rate",
		"windowSize":         "Window Size",
	}
	props = -1
	for key, propName := range clientParams {
		for k := range commonPropVals {
			prop := strings.Split(k, "|")
			if key == prop[0] {
				props++
				if props%3 == 0 {
					if props > 0 {
						_, err = io.WriteString(&buf, "</tr>")
						if err != nil {
							return "<b>Error generating config table</b>"
						}
					}
					_, err = io.WriteString(&buf, "<tr>")
					if err != nil {
						return "<b>Error generating config table</b>"
					}
				}
				if strings.HasSuffix(prop[0], "Rate") {
					val, _ := strconv.ParseFloat(prop[1], 64)
					prop[1] = fmt.Sprintf("%0.2f %%", float64(val*100))
				}
				_, err = io.WriteString(
					&buf,
					fmt.Sprintf(
						"<td width=\"20%%\"><b>%s:</b></td><td width=\"13%%\">%s</td>",
						propName,
						prop[1],
					),
				)
				if err != nil {
					return "<b>Error generating config table</b>"
				}
			}
		}
	}
	_, err = io.WriteString(&buf, "</tbody></table>")
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	_, err = io.WriteString(&buf, "<table class=\"config\"><tbody>")
	if err != nil {
		return "<b>Error generating config table</b>"
	}
	_, err = io.WriteString(
		&buf,
		"<tr><td align=\"center\" colspan=\"6\" class=\"titleRow\"><b>Other Parameters</b></td></tr>",
	)
	if err != nil {
		return "<b>Error generating config table</b>"
	}

	otherParams := map[string]string{
		"preseedShards":          "Use pre-seeded UTXO set",
		"preseedCount":           "Size of pre-seeded UTXO set",
		"architectureID":         "Architecture",
		"targetBlockInterval":    "Block interval",
		"shardReplicationFactor": "Shard Replication Factor",
		"batchSize":              "Batch Size",
		"batchDelay":             "Batch Delay",
		"commitHash":             "Code commit (System)",
		"controllerCommitHash":   "Code commit (Test Controller)",
		"sweepMatrix":            "Sweep Matrix",
		"stxoCacheDepth":         "Spent TXO Cache Depth",
		"minDate":                "First test run completed",
		"maxDate":                "Last test run completed",
	}

	props = -1
	for key, propName := range otherParams {
		for k := range commonPropVals {
			prop := strings.Split(k, "|")
			if prop[0] == "commitHash" {
				if len(prop[1]) > 7 {
					prop[1] = fmt.Sprintf(
						`<a target="_blank" href="https://github.com/mit-dci/cbdc-universe0/tree/%s">%s</a>`,
						prop[1],
						prop[1][:7],
					)
				}
			}
			if prop[0] == "controllerCommitHash" {
				if len(prop[1]) > 7 {
					prop[1] = fmt.Sprintf(
						`<a target="_blank" href="https://github.com/mit-dci/cbdc-test-controller/tree/%s">%s</a>`,
						prop[1],
						prop[1][:7],
					)
				}
			}
			if key == prop[0] {
				if strings.HasSuffix(prop[0], "Rate") {
					val, _ := strconv.ParseFloat(prop[1], 64)
					prop[1] = fmt.Sprintf("%0.2f %%", float64(val*100))
				}
				prop[0] = propName
				props++
				if props%3 == 0 {
					if props > 0 {
						_, err = io.WriteString(&buf, "</tr>")
						if err != nil {
							return "<b>Error generating config table</b>"
						}
					}
					_, err = io.WriteString(&buf, "<tr>")
					if err != nil {
						return "<b>Error generating config table</b>"
					}
				}
				_, err = io.WriteString(
					&buf,
					fmt.Sprintf(
						"<td width=\"20%%\"><b>%s:</b></td><td width=\"13%%\">%s</td>",
						prop[0],
						prop[1],
					),
				)
				if err != nil {
					return "<b>Error generating config table</b>"
				}
			}
		}
	}

	for k := range commonPropVals {
		prop := strings.Split(k, "|")

		_, ok := clientParams[prop[0]]
		if ok {
			continue
		}

		_, ok = raftParams[prop[0]]
		if ok {
			continue
		}

		_, ok = otherParams[prop[0]]
		if ok {
			continue
		}

		if prop[0] == "multiRegion" {
			continue
		}

		roleParam := false
		for _, r := range roles {
			if strings.HasPrefix(prop[0], r) {
				roleParam = true
				break
			}
		}
		if roleParam {
			continue
		}

		props++
		if props%3 == 0 {
			if props > 0 {
				_, err = io.WriteString(&buf, "</tr>")
				if err != nil {
					return "<b>Error generating config table</b>"
				}
			}
			_, err = io.WriteString(&buf, "<tr>")
			if err != nil {
				return "<b>Error generating config table</b>"
			}
		}
		_, err = io.WriteString(
			&buf,
			fmt.Sprintf(
				"<td width=\"20%%\"><b>%s:</b></td><td width=\"13%%\">%s</td>",
				prop[0],
				prop[1],
			),
		)
		if err != nil {
			return "<b>Error generating config table</b>"
		}

	}

	_, err = io.WriteString(&buf, "</tbody></table>")
	if err != nil {
		return "<b>Error generating config table</b>"
	}

	return buf.String()
}

func (h *HttpServer) generatePlot(input string) string {
	r := bytes.NewReader([]byte(input))
	plot, err := h.generateSweepPlot(r)
	if err != nil {
		return "<b>Error generating plot</b>"
	}
	return fmt.Sprintf(
		`data:image/png;base64, %s`,
		base64.StdEncoding.EncodeToString(plot),
	)
}

type runPlotQuery struct {
	RunID    string `json:"runID"`
	PlotType string `json:"plotType"`
}

func (h *HttpServer) generateRunPlot(input string) string {
	var req runPlotQuery
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		logging.Errorf("Error unmarshaling runplot JSON: %v", err)
		return ""
	}

	path := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s/plots/%s.png", req.RunID, req.PlotType),
	)
	plot, err := ioutil.ReadFile(path)
	if err != nil {
		return "<b>Error reading plot</b>"
	}
	return fmt.Sprintf(
		`data:image/png;base64, %s`,
		base64.StdEncoding.EncodeToString(plot),
	)
}

//go:embed report-template.html
var reportTemplate string

func (h *HttpServer) generateReport(def reportDefinition) string {
	result := ""
	reportLock.Lock()
	defer reportLock.Unlock()
	b, _ := json.Marshal(def)
	sha := sha256.Sum256(b)
	err := os.MkdirAll(
		filepath.Join(common.DataDir(), "testruns/reports"),
		0755,
	)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return ""
	}

	outputFile := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/reports/%x.html", sha),
	)
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		body := def.Definition
		body = strings.ReplaceAll(body, "%TITLE%", def.Title)

		mrk := strings.Index(body, "[cfg]")
		for mrk > -1 {
			endMrk := strings.Index(body[mrk:], "[/cfg]") + mrk
			if endMrk-mrk == -1 {
				break
			}
			body = fmt.Sprintf(
				"%s%s%s",
				body[:mrk],
				h.generateConfigTable(body[mrk+5:endMrk]),
				body[endMrk+6:],
			)
			mrk = strings.Index(body, "[cfg]")
		}

		mrk = strings.Index(body, "[plot]")
		for mrk > -1 {
			endMrk := strings.Index(body[mrk:], "[/plot]") + mrk
			if endMrk-mrk == -1 {
				break
			}
			body = fmt.Sprintf(
				"%s%s%s",
				body[:mrk],
				h.generatePlot(body[mrk+6:endMrk]),
				body[endMrk+7:],
			)
			mrk = strings.Index(body, "[plot]")
		}

		mrk = strings.Index(body, "[runplot]")
		for mrk > -1 {
			endMrk := strings.Index(body[mrk:], "[/runplot]") + mrk
			if endMrk-mrk == -1 {
				break
			}
			body = fmt.Sprintf(
				"%s%s%s",
				body[:mrk],
				h.generateRunPlot(body[mrk+9:endMrk]),
				body[endMrk+10:],
			)
			mrk = strings.Index(body, "[runplot]")
		}

		result = strings.ReplaceAll(reportTemplate, "%TITLE%", def.Title)
		result = strings.ReplaceAll(result, "%BODY%", body)
		err = os.WriteFile(outputFile, []byte(result), 0644)
		if err != nil {
			return ""
		}
	} else {
		resultB, _ := os.ReadFile(outputFile)
		result = string(resultB)
	}
	return result
}
