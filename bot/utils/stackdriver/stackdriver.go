package stackdriver

import (
	"context"
	"fmt"

	"cloud.google.com/go/logging"
	"github.com/sirupsen/logrus"
)

type Hook struct {
	client  *logging.Client
	logger  *logging.Logger
	syncCtx context.Context
	sync    bool
}

func initHook(sync bool, client *logging.Client, logID string, opts ...logging.LoggerOption) *Hook {
	h := &Hook{client: client, sync: sync, syncCtx: context.Background()}
	h.logger = h.client.Logger(logID, opts...)
	return h
}

// New returns a logrus hook for the given client and
// relays logs to the Stackdriver API asynchronously. It is the client's
// responsibility to call client.Close() so that buffered logs get
// written before the end of the program!
func New(client *logging.Client, logID string, opts ...logging.LoggerOption) *Hook {
	return initHook(false, client, logID, opts...)
}

// NewSync returns a logrus hook for the given client and
// relays logs to the Stackdriver API synchronously. Not recommended for
// typical use (see https://godoc.org/cloud.google.com/go/logging#hdr-Synchronous_Logging)
// In order to use a non-background context for a LogSync entry, call SetSyncContext on the
// returned hook.
func NewSync(client *logging.Client, logID string, opts ...logging.LoggerOption) *Hook {
	return initHook(true, client, logID, opts...)
}

func (h *Hook) SetSyncContext(ctx context.Context) {
	h.syncCtx = ctx
}

func mapLogrusToStackdriverLevel(l logrus.Level) logging.Severity {
	switch l {
	case logrus.DebugLevel:
		return logging.Debug
	case logrus.InfoLevel:
		return logging.Info
	case logrus.WarnLevel:
		return logging.Warning
	case logrus.ErrorLevel:
		return logging.Error
	case logrus.FatalLevel:
		return logging.Critical
	case logrus.PanicLevel:
		return logging.Alert
	default:
		return logging.Debug // Should never happen
	}
}

// Levels returns the logrus levels that this hook is applied to.
// TODO: Allow configuration.
func (h *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire sends the log entry to Stackdriver either synchrounously or asynchronously, depending
// on how the hook was instantiated. Levels from Logrus are mapped to the Stackdriver API levels
// (https://godoc.org/cloud.google.com/go/logging#pkg-constants) as follows:
// [logrus Level] -> [Stackdriver Level]
// Debug, Info, Warning, Error -> (same)
// Fatal -> Critical
// Panic -> Alert
func (h *Hook) Fire(e *logrus.Entry) error {
	entry := logging.Entry{
		Timestamp: e.Time,
		Severity:  mapLogrusToStackdriverLevel(e.Level),
		Payload:   e.Message,
	}
	entry.Labels = make(map[string]string)
	for k, v := range e.Data {
		switch t := v.(type) {
		case string:
			entry.Labels[k] = t
		default:
			entry.Labels[k] = fmt.Sprintf("%v", t)
		}
	}
	if h.sync {
		return h.logger.LogSync(h.syncCtx, entry)
	}
	h.logger.Log(entry)
	return nil
}
