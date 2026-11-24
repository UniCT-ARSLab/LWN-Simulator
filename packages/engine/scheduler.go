package engine

import schedulerruntime "github.com/Giuseppe-Compagnone/lwn-engine/scheduler"

type Scheduler = schedulerruntime.Scheduler

func NewScheduler() *Scheduler {
	return schedulerruntime.New()
}
