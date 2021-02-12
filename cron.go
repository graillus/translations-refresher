package main

import "time"

// Cron represents a sceduled cron task
type Cron struct {
	period time.Duration
}

// NewCron creates a new Cron instance
func NewCron(p time.Duration) *Cron {
	return &Cron{p}
}

// Run starts the Cron
func (c Cron) Run(f func()) {
	ticker := time.NewTicker(c.period)

	for {
		select {
		case <-ticker.C:
			f()
		}
	}
}
