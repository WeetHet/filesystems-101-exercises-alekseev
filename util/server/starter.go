package server

import (
	"context"
	"sync"

	"tests-runner/util/workgroup"
)

// Starter can start and stop multiple Startable components
type Starter struct {
	mtx     sync.RWMutex
	all     []SubServer
	started []SubServer
}

// NewStarter creates new Starter which is responsible for
// starting and stopping specified components
func NewStarter(ss []SubServer) *Starter {
	return &Starter{
		all:     ss,
		started: make([]SubServer, 0, len(ss)),
	}
}

// Start tries to start all of the components
// In case error is returned user must still call Stop() and Wait()
// to ensure, that components which were started does not consume
// unexpected resources.
func (s *Starter) Start(ctx context.Context) error {
	wg := workgroup.New(workgroup.Config{
		Concurrency: len(s.all),
		StopOnError: true,
	})
	for i := range s.all {
		c := s.all[i]
		wg.GoCtx(ctx, func(ctx context.Context) (err error) {
			if err = c.Start(ctx); err != nil {
				return err
			}
			s.mtx.Lock()
			s.started = append(s.started, c)
			s.mtx.Unlock()
			return
		})
	}
	wg.Wait()
	return wg.Error()
}

// Stop stops all started components
func (s *Starter) Stop() {
	if s == nil {
		return
	}

	s.mtx.RLock()
	for _, c := range s.started {
		c.Stop()
	}
	s.mtx.RUnlock()
}

// Wait waits for all started components to stop
// User must call Stop() earlier.
func (s *Starter) Wait() {
	if s == nil {
		return
	}

	s.mtx.RLock()
	for _, c := range s.started {
		c.Wait()
	}
	s.mtx.RUnlock()
}

// AllStarted reports if all components were started
func (s *Starter) AllStarted() bool {
	s.mtx.RLock()
	res := len(s.all) == len(s.started)
	s.mtx.RUnlock()
	return res
}

// IsHealthy reports if all components are healthy
func (s *Starter) IsHealthy(ctx context.Context) (err error) {
	if !s.AllStarted() {
		return nil
	}
	for _, c := range s.all {
		if err = c.IsHealthy(ctx); err != nil {
			return err
		}
	}
	return nil
}

// IsReady reports if all components are ready
func (s *Starter) IsReady(ctx context.Context) bool {
	if !s.AllStarted() {
		return false
	}
	for _, c := range s.all {
		if !c.IsReady(ctx) {
			return false
		}
	}
	return true
}
