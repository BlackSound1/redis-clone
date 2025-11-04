package main

import "time"

// Track various context variables useful across the whole app
type AppState struct {
	conf          *Config
	aof           *AOF
	bgSaveRunning bool
	dbCopy        map[string]*Item
	transaction   *Transaction
	monitors      []*Client
}

// NewAppState creates a new AppState type with the given Config settings
// If the Config type specifies that AOF should be enabled, it will create a new AOF type
// and, if necessary, a new goroutine to flush the writer every second
func NewAppState(conf *Config) *AppState {
	state := AppState{
		conf: conf,
	}

	if conf.aofEnabled {
		state.aof = NewAOF(conf)

		// If aofSync mode is everysec, set up a new goroutine
		// that, every second, flushes the writers buffer
		if conf.aofFsync == EverySec {
			go func() {
				t := time.NewTicker(time.Second)
				defer t.Stop()

				for range t.C {
					state.aof.w.Flush()
				}
			}()
		}
	}

	return &state
}
