package notify

import "time"

type Event struct {
	Summary      string
	Start        time.Time
	End          time.Time
	Description  string
	CalendarName string
	UID          string
}

type NotificationConfig struct {
	Enabled        bool
	CheckInterval  int
	AdvanceNotice  []int
	ReloadInterval int
}

type Daemon struct{}

func NewDaemon(config *NotificationConfig, loader func() ([]Event, error)) *Daemon {
	return &Daemon{}
}

func (d *Daemon) Run() error {
	return nil
}
