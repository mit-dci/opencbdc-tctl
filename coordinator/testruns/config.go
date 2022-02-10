package testruns

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/coordinator"
)

// TestManagerConfig is the main type in which parameters for the controller can
// be persisted
type TestManagerConfig struct {
	MaxAgents int `json:"maxAgents"`
}

// SetMaxAgents changes the maximum number of parallel running agents which is
// used by the scheduler. The scheduler will not run testruns from the queue
// that would exceed this number of agents active
func (t *TestRunManager) SetMaxAgents(max int) error {
	t.config.MaxAgents = max
	return t.PersistConfig()
}

// Config returns the entire config of the TestRunManager
func (t *TestRunManager) Config() TestManagerConfig {
	return *t.config
}

// LoadConfig loads the configuration variables from persistence (file). It also
// sends a real-time update for the frontend to know what the current value is
func (t *TestRunManager) LoadConfig() error {
	path := filepath.Join(common.DataDir(), "testruns", "manager.config.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		f, err := os.OpenFile(
			filepath.Join(common.DataDir(), "testruns", "manager.config.json"),
			os.O_RDONLY,
			0644,
		)
		if err != nil {
			return err
		}
		defer f.Close()
		return json.NewDecoder(f).Decode(t.config)
	}
	t.ev <- coordinator.Event{
		Type: coordinator.EventTypeTestRunManagerConfigUpdated,
		Payload: coordinator.TestRunManagerConfigUpdatedPayload{
			Config: t.config,
		},
	}
	return nil
}

// PersistConfig saves the configuration variables to persistence (file).It also
// sends a real-time update for the frontend to know what the current value is
func (t *TestRunManager) PersistConfig() error {
	f, err := os.OpenFile(
		filepath.Join(common.DataDir(), "testruns", "manager.config.json"),
		os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(t.config)
	if err != nil {
		return err
	}
	t.ev <- coordinator.Event{
		Type: coordinator.EventTypeTestRunManagerConfigUpdated,
		Payload: coordinator.TestRunManagerConfigUpdatedPayload{
			Config: t.config,
		},
	}
	return nil
}
