package http

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/coordinator"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
} // use default options

type websocketConn struct {
	conn                               *websocket.Conn
	outgoing                           chan []byte
	subscribedToTestRunLogForTestRunID string
}

var testRunUpdate = sync.Map{}
var connectedAgentsUpdate *time.Timer
var websocketsLock sync.Mutex = sync.Mutex{}
var websockets []*websocketConn = []*websocketConn{}

type websocketMessage struct {
	Type string                 `json:"t"`
	Msg  map[string]interface{} `json:"m"`
}

func (c *websocketConn) sendLoop() {
	for msg := range c.outgoing {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			logging.Errorf("Error writing to websocket: %v", err)
		}
	}
}

func (srv *HttpServer) publishToWebsocketsLoop() {
	for ev := range srv.events {
		if ev.Type == coordinator.EventTypeSystemStateChange {
			ev = srv.GetSystemStateEvent()
		}
		b, err := json.Marshal(ev)
		if err != nil {
			log.Printf("encode: %v\n", err)
			continue
		}

		// Convert testruns to their more compact frontend representation
		if ev.Type == coordinator.EventTypeTestRunCreated {
			existingPayload, ok := ev.Payload.(*coordinator.TestRunCreatedPayload)
			if ok {
				tr, ok := existingPayload.Data.(*common.TestRun)
				if ok {
					ev.Payload = &coordinator.TestRunCreatedPayload{
						Data: srv.makeFrontendRun(tr),
					}
				}
			}
		}

		// Throttle updates
		if ev.Type == coordinator.EventTypeTestRunStatusChanged {
			pl, ok := ev.Payload.(coordinator.TestRunStatusChangePayload)
			if ok && !pl.Debounced {
				pl.Debounced = true
				debounced := coordinator.Event{
					Type:    ev.Type,
					Payload: pl,
				}

				last, ok := testRunUpdate.Load(pl.TestRunID)
				if ok {
					lastTimer, ok := last.(*time.Timer)
					if ok {
						lastTimer.Stop()
					}

				}
				testRunUpdate.Store(
					pl.TestRunID,
					time.AfterFunc(time.Second*1, func() {
						srv.events <- debounced
						testRunUpdate.Delete(pl.TestRunID)
					}),
				)
				continue
			}
		} else if ev.Type == coordinator.EventTypeConnectedAgentCountChanged {
			pl, ok := ev.Payload.(coordinator.ConnectedAgentCountChangedPayload)
			if ok && !pl.Debounced {
				pl.Debounced = true
				debounced := coordinator.Event{
					Type:    ev.Type,
					Payload: pl,
				}
				if connectedAgentsUpdate != nil {
					connectedAgentsUpdate.Stop()
				}
				connectedAgentsUpdate = time.AfterFunc(time.Second*1, func() {
					srv.events <- debounced
				})
				continue
			}
		}

		for _, c := range websockets {
			if c != nil {
				write := true

				switch ev.Type {
				case coordinator.EventTypeTestRunLogAppended:
					write = c.subscribedToTestRunLogForTestRunID == ev.Payload.(coordinator.TestRunLogAppendedPayload).TestRunID
				}

				if write {
					// Make this non-blocking
					select {
					case c.outgoing <- b:
					default:
					}
				}
			}
		}
	}
}

func (srv *HttpServer) wsTokenPayload(
	r *http.Request,
) (map[string]interface{}, error) {
	token := make([]byte, 8)
	_, err := rand.Read(token)
	if err != nil {
		return map[string]interface{}{}, err
	}
	srv.wsTokens.Store(fmt.Sprintf("%x", token), time.Now().Add(30*time.Second))
	return map[string]interface{}{
		"target": fmt.Sprintf(
			"%s/ws/%x",
			srv.GetHttpsEndpoint("wss", srv.httpsWithoutClientCertPort, r),
			token,
		),
	}, nil
}

func (srv *HttpServer) tokenCleanupLoop() {
	for {
		srv.wsTokens.Range(func(key, value interface{}) bool {
			if value.(time.Time).Before(time.Now()) {
				srv.wsTokens.Delete(key)
			}
			return true
		})
		time.Sleep(time.Second * 30)
	}
}

func (srv *HttpServer) GetSystemStateEvent() coordinator.Event {
	state := "running"
	if !srv.tr.TestRunsLoaded() {
		state = "loading"
	}
	return coordinator.Event{
		Type: coordinator.EventTypeSystemStateChange,
		Payload: coordinator.SystemStateChangePayload{
			State: state,
		},
	}
}
