package workers

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
)

const (
	RepeatInterval = 30 * time.Second
)

type Reconciler struct {
	wakeup chan *sync.WaitGroup
}

// Wakeup causes the worker reconcile to be performed as soon as possible.  If wait is true, the this
// function blocks until the reconcile is completed, otherwise this function does not block.
func (r *Reconciler) Wakeup(wait bool) {
	if wait {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		r.wakeup <- wg
		wg.Wait()
	} else {
		select {
		case r.wakeup <- nil:
			// wakeup channel accepted the message
		default:
			// wakeup channel was full..
		}
	}
}

func (r *Reconciler) Start(worker Worker) {
	r.wakeup = make(chan *sync.WaitGroup, 1)
	*worker.GetStopChan() = make(chan struct{})
	worker.GetSyncGroup().Add(1)
	worker.SetIsRunning(true)

	glog.V(1).Infoln(fmt.Sprintf("Starting reconciliation loop for %T [%s]", worker, worker.GetID()))
	//starts reconcile immediately and then on every repeat interval
	worker.reconcile()
	ticker := time.NewTicker(RepeatInterval)
	go func() {
		for {
			select {
			case wg := <-r.wakeup: //we were asked to wake up...
				glog.V(1).Infoln(fmt.Sprintf("Starting reconciliation loop for %T [%s]", worker, worker.GetID()))
				worker.reconcile()
				if wg != nil {
					wg.Done()
				}
			case <-ticker.C: //time out
				glog.V(1).Infoln(fmt.Sprintf("Starting reconciliation loop for %T [%s]", worker, worker.GetID()))
				worker.reconcile()
			case <-*worker.GetStopChan():
				ticker.Stop()
				defer worker.GetSyncGroup().Done()
				glog.V(1).Infoln(fmt.Sprintf("Stopping reconciliation loop for %T [%s]", worker, worker.GetID()))
				return
			}
		}
	}()
}

func (r *Reconciler) Stop(worker Worker) {
	defer worker.SetIsRunning(false)
	select {
	case <-*worker.GetStopChan(): //already closed
		return
	default:
		close(*worker.GetStopChan()) //explicit close
		worker.GetSyncGroup().Wait() //wait for in-flight job to finish
	}
}
