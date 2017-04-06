package containers

import (
	"github.com/g8os/core0/base/pm/core"
	"github.com/patrickmn/go-cache"
	"time"
)

const (
	InternalRoute = core.Route("--internal--")
)

type internalRouter struct {
	cache *cache.Cache
}

func newInternalRouter() *internalRouter {
	i := &internalRouter{
		cache: cache.New(30*time.Second, 5*time.Second),
	}
	i.cache.OnEvicted(i.evict)
	return i
}

func (i *internalRouter) evict(id string, o interface{}) {
	ch := o.(chan *core.JobResult)
	close(ch)
}

func (i *internalRouter) Prepare(id string) {
	i.cache.Set(id, make(chan *core.JobResult, 1), cache.DefaultExpiration)
}

func (i *internalRouter) Get(id string) *core.JobResult {
	o, ok := i.cache.Get(id)
	if !ok {
		return nil
	}
	ch := o.(chan *core.JobResult)
	return <-ch
}

func (i *internalRouter) Route(job *core.JobResult) {
	o, ok := i.cache.Get(job.ID)
	if !ok {
		return
	}

	ch := o.(chan *core.JobResult)
	ch <- job
}
