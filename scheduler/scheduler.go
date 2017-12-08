package scheduler

import (
	"time"
	"runtime"
)

type Fun func()

type Scheduler struct {
	timeout time.Duration
	jobPool chan Fun
	end     chan bool
}

func (c *Scheduler) Start() {
	go func() {
		for {
			select {
			case <-time.After(c.timeout):
				if len(c.jobPool) <= 0 {
					c.end <- true
					return
				}
				break
			}
		}
	}()
}

func (c *Scheduler) Add(f Fun) {
	c.jobPool <- f
	go func() {
		f()
		<-c.jobPool
	}()
}

func (c *Scheduler) Wait() {
	<-c.end
}

func NewScheduler() *Scheduler {
	runtime.GOMAXPROCS(runtime.NumCPU())
	return &Scheduler{
		jobPool: make(chan Fun, 10),
		timeout: time.Second * 5,
		end:     make(chan bool),
	}
}
