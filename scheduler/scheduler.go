package scheduler

import (
	"time"
	"sync"
)

type Fun func()

type Scheduler struct {
	jc      chan Fun
	mt      sync.Mutex
	timeout time.Duration
	jobPool []Fun
	end     chan bool
}

func (c *Scheduler) Start() {
	go func() {
		for {
			select {
			case j := <-c.jc:
				go func() {
					c.mt.Lock()
					index := len(c.jobPool)
					c.jobPool = append(c.jobPool, j)
					j()
					c.jobPool = append(c.jobPool[:index], c.jobPool[index+1:]...)
					c.mt.Unlock()
				}()
				break
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
	c.jc <- f
}

func (c *Scheduler) Wait() {
	<-c.end
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		jc:      make(chan Fun, 10),
		jobPool: make([]Fun, 0),
		timeout: time.Second * 5,
		end:     make(chan bool),
	}
}
