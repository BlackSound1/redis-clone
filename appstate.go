package main

import "time"

type RDB_Stats struct {
	rdb_last_save_ts int64
	rdb_saves        int
}

type AOF_Stats struct {
	aof_rewrites int
}

type GeneralStats struct {
	total_connections_received int
	total_commands_processed   int
	expired_keys               int
	evicted_keys               int
}

// Track various context variables useful across the whole app
type AppState struct {
	conf              *Config
	aof               *AOF
	bgSaveRunning     bool
	aofRewriteRunning bool
	dbCopy            map[string]*Item
	transaction       *Transaction
	monitors          []*Client
	serverStart       time.Time
	clientCount       int
	peakMem           int64
	info              *Info
	rdbStats          RDB_Stats
	aofStats          AOF_Stats
	generalStats      GeneralStats
}

// NewAppState creates a new AppState type with the given Config settings
// If the Config type specifies that AOF should be enabled, it will create a new AOF type
// and, if necessary, a new goroutine to flush the writer every second
func NewAppState(conf *Config) *AppState {
	state := AppState{
		conf:         conf,
		serverStart:  time.Now(),
		info:         NewInfo(),
		rdbStats:     RDB_Stats{},
		aofStats:     AOF_Stats{},
		generalStats: GeneralStats{},
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
