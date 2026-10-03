package storage

import (
	"encoding/json"
	"os"
	"time"
)

// State — runtime состояние: активное ядро/профиль, pid, время старта.
type State struct {
	Core       string    `json:"core"`
	ProfileID  string    `json:"profileId"`
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"startedAt"`
	ConfigPath string    `json:"configPath"`
}

// LoadState reads state.json; nil means not running.
func (s *Store) LoadState() (*State, error) {
	data, err := readFile(s.StatePath)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	if st.PID == 0 {
		return nil, nil
	}
	return &st, nil
}

// SaveState writes state.json atomically.
func (s *Store) SaveState(st State) error {
	return withLock(s.StatePath, func() error {
		data, err := json.MarshalIndent(st, "", "  ")
		if err != nil {
			return err
		}
		return writeAtomic(s.StatePath, data)
	})
}

// ClearState removes state.json (idempotent).
func (s *Store) ClearState() error {
	err := os.Remove(s.StatePath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
