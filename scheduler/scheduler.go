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
	JobPool []Fun
	end     chan int
}

func (c *Scheduler) Start() {
	go func() {
		for {
			select {
			case j := <-c.jc:
				go func() {
					c.mt.Lock()
					index := len(c.JobPool)
					c.JobPool = append(c.JobPool, j)
					j()
					c.JobPool = append(c.JobPool[:index], c.JobPool[index+1:]...)
					c.mt.Unlock()
				}()
				break
			case <-time.After(c.timeout):
				if len(c.JobPool) <= 0 {
					c.end <- 0
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
		JobPool: make([]Fun, 0),
		timeout: time.Second * 5,
		end:     make(chan int),
	}
}
