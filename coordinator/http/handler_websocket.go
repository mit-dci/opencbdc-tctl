package http

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/mit-dci/opencbdc-tct/coordinator"
	"github.com/mit-dci/opencbdc-tct/logging"
)

func (srv *HttpServer) wsWithTokenHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	vars := mux.Vars(r)

	token, ok := srv.wsTokens.LoadAndDelete(vars["token"])
	if !ok {
		logging.Warnf(
			"Websocket connection tried with non-existent token %s",
			vars["token"],
		)
		http.Error(w, "Forbidden", 401)
		return
	}
	if token.(time.Time).Before(time.Now()) {
		logging.Warnf(
			"Websocket connection tried with expired token %s",
			vars["token"],
		)
		http.Error(w, "Forbidden", 401)
		return
	}

	logging.Warnf("Websocket connection initiated with token %s", vars["token"])
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade: %v", err)
		return
	}
	conn := &websocketConn{
		conn:                               c,
		outgoing:                           make(chan []byte, 100),
		subscribedToTestRunLogForTestRunID: "",
	}
	websocketsLock.Lock()
	websockets = append(websockets, conn)
	srv.events <- coordinator.Event{
		Type: coordinator.EventTypeConnectedUsersChanged,
		Payload: coordinator.ConnectedUsersChangedPayload{
			Count: len(websockets),
		},
	}
	websocketsLock.Unlock()

	go conn.sendLoop()

	msg, err := json.Marshal(coordinator.Event{
		Type: coordinator.EventTypeMaintenanceModeChanged,
		Payload: coordinator.MaintenanceModeChangedPayload{
			MaintenanceMode: srv.coord.GetMaintenance(),
		},
	})
	if err == nil {
		conn.outgoing <- msg
	}

	msg, err = json.Marshal(srv.GetSystemStateEvent())
	if err == nil {
		conn.outgoing <- msg
	} else {
		logging.Errorf("Error marshalling system state: %v", err)
	}

	defer c.Close()
	for {
		mt, msg, err := c.ReadMessage()
		if err != nil {
			logging.Warnf("read error: %v", err)
			break
		}

		if mt == websocket.TextMessage {
			var m websocketMessage
			err = json.Unmarshal(msg, &m)
			if err != nil {
				logging.Warnf("unmarhal error: %v", err)
				break
			}

			switch m.Type {
			case "unsubscribeTestRunLog":
				conn.subscribedToTestRunLogForTestRunID = ""
			case "subscribeTestRunLog":
				conn.subscribedToTestRunLogForTestRunID = m.Msg["id"].(string)
				tr, ok := srv.tr.GetTestRun(
					conn.subscribedToTestRunLogForTestRunID,
				)
				if ok {
					log := tr.LogTail()
					ev := coordinator.Event{
						Type: coordinator.EventTypeTestRunLogAppended,
						Payload: coordinator.TestRunLogAppendedPayload{
							TestRunID: tr.ID,
							Log:       log,
						},
					}
					b, err := json.Marshal(ev)
					if err == nil {
						conn.outgoing <- b
					}
				}
			}
		}

	}

	websocketsLock.Lock()
	idx := -1
	for i, ws := range websockets {
		if ws == conn {
			idx = i
		}
	}
	if idx != -1 {
		websockets[len(websockets)-1], websockets[idx] = websockets[idx], websockets[len(websockets)-1]
		websockets = websockets[:len(websockets)-1]
	}
	srv.events <- coordinator.Event{
		Type: coordinator.EventTypeConnectedUsersChanged,
		Payload: coordinator.ConnectedUsersChangedPayload{
			Count: len(websockets),
		},
	}
	websocketsLock.Unlock()
}
