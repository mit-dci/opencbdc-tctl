package http

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/logging"
)

type packetBucket struct {
	Target            string   `json:"target"`
	Source            string   `json:"source"`
	BandwidthOverTime []*int64 `json:"bandwidthOverTime"`
	Bandwidth         *int64   `json:"bandwidth"`
}

func (h *HttpServer) testRunBandwidthData(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	runID := params["runID"]
	tr, ok := h.tr.GetTestRun(runID)
	if !ok {
		http.Error(w, "Not found", 404)
		return
	}

	bandWidthFile := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s/bandwidth.json", tr.ID),
	)
	result := make([]*packetBucket, 0)

	if _, err := os.Stat(bandWidthFile); os.IsNotExist(err) {

		logsFolder := filepath.Join(
			common.DataDir(),
			fmt.Sprintf("testruns/%s/packetlogs", tr.ID),
		)
		err := os.MkdirAll(logsFolder, 0755)
		if err != nil && !os.IsExist(err) {
			http.Error(w, "Internal Server Error", 500)
			return
		}

		downloads := make([]common.S3Download, 0)
		packetFiles := make([]string, len(tr.ExecutedCommands))
		for i, cmd := range tr.ExecutedCommands {

			packetFiles[i] = filepath.Join(
				logsFolder,
				fmt.Sprintf("packets_%s.bin", cmd.CommandID),
			)
			if _, err := os.Stat(packetFiles[i]); os.IsNotExist(err) {

				downloads = append(downloads, common.S3Download{
					SourceRegion: os.Getenv("AWS_REGION"),
					SourceBucket: os.Getenv("OUTPUTS_S3_BUCKET"),
					SourcePath: fmt.Sprintf(
						"command-outputs/%s/cmd_%s_packets.bin",
						cmd.CommandID[:8],
						cmd.CommandID,
					),
					TargetPath: packetFiles[i],
					Retries:    3,
				})
			}
		}

		if len(downloads) > 0 {
			logging.Infof("Downloading %d files from S3")
			err = h.awsm.DownloadMultipleFromS3(downloads)
			if err != nil && !os.IsExist(err) {
				http.Error(w, "Internal Server Error", 500)
				return
			}
		}

		packetBuckets := sync.Map{}
		packetFileChan := make(chan string, runtime.NumCPU())
		wg := sync.WaitGroup{}
		for i := 0; i < runtime.NumCPU(); i++ {
			wg.Add(1)
			go func() {
				for path := range packetFileChan {
					logging.Infof("Processing packet file %s", path)
					f, err := os.Open(path)
					if err != nil && !os.IsExist(err) {
						http.Error(w, "Internal Server Error", 500)
						return
					}
					br := bufio.NewReaderSize(f, 1024*1024)

					for pkt, err := common.ReadPacketMetadata(br); err == nil; pkt, err = common.ReadPacketMetadata(f) {
						raw, ok := packetBuckets.Load(bucketHash(pkt))
						var buck *packetBucket
						if !ok {
							bw := int64(0)
							buck = &packetBucket{
								Source: formatEndpoint(
									pkt.SourceIP,
									pkt.SourcePort,
								),
								Target: formatEndpoint(
									pkt.TargetIP,
									pkt.TargetPort,
								),
								BandwidthOverTime: make([]*int64, 400),
								Bandwidth:         &bw,
							}

							for j := 0; j < 400; j++ {
								bw2 := int64(0)
								buck.BandwidthOverTime[j] = &bw2
							}
							packetBuckets.Store(bucketHash(pkt), buck)
						} else {
							buck = raw.(*packetBucket)
						}
						atomic.AddInt64(buck.Bandwidth, int64(pkt.Length))
						atomic.AddInt64(
							buck.BandwidthOverTime[pkt.Timestamp],
							int64(pkt.Length),
						)
					}
					f.Close()
				}
				wg.Done()
			}()
		}

		for _, path := range packetFiles {
			packetFileChan <- path
		}
		close(packetFileChan)
		wg.Wait()
		packetBuckets.Range(func(k interface{}, v interface{}) bool {
			result = append(result, v.(*packetBucket))
			return true
		})

		json, err := json.Marshal(result)
		if err != nil {
			logging.Errorf("Error: %v", err)
		}
		err = os.WriteFile(bandWidthFile, json, 0644)
		if err != nil {
			logging.Errorf("Error: %v", err)
		}
	} else {
		b, err := os.ReadFile(bandWidthFile)
		if err != nil {
			logging.Errorf("Error: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		err = json.Unmarshal(b, &result)
		if err != nil {
			logging.Errorf("Error: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
	}
	writeJson(w, result)
}

// formatEndpoint strips off ephemeral ports
func formatEndpoint(ip [4]byte, port uint16) string {
	endpoint := net.IP(ip[:]).String()
	if port < 32000 {
		endpoint = fmt.Sprintf(
			"%s:%d",
			endpoint,
			port,
		)
	}
	return endpoint
}

func bucketHash(pkt *common.PacketMetadata) string {
	return fmt.Sprintf(
		"%x|%x|%d",
		pkt.SourceIP[:],
		pkt.TargetIP[:],
		pkt.TargetPort,
	)
}
