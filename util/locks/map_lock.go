package locks

import (
	"sync"
)

// MappedLock stores locks indexes by id of type T
// and allows to lock individual objects w/o tracking them.
type MappedLock[T comparable] struct {
	mtx   sync.Mutex
	locks map[T]*sync.Mutex
}

// NewMappedLock allocates map of locks
func NewMappedLock[T comparable]() *MappedLock[T] {
	return &MappedLock[T]{
		locks: make(map[T]*sync.Mutex),
	}
}

// Lock the object with given id
func (l *MappedLock[T]) Lock(id T) {
	_ = l.tryLock(id, true)
}

// TryLock the object with given id. Returns false if failed (already locked)
func (l *MappedLock[T]) TryLock(id T) (done bool) {
	return l.tryLock(id, false)
}

func (l *MappedLock[T]) tryLock(id T, wait bool) (done bool) {
restart:
	l.mtx.Lock()
	lk := l.locks[id]
	if lk == nil { // fast-path
		lk = &sync.Mutex{}
		lk.Lock()
		l.locks[id] = lk
		l.mtx.Unlock()
		return true
	}
	l.mtx.Unlock()

	if !wait {
		return false
	}

	// wait for previous lock to be released
	lk.Lock()
	lk.Unlock() // nolint:staticcheck,gocritic
	goto restart
}

// Unlock previously locked object with given id
func (l *MappedLock[T]) Unlock(id T) {
	l.mtx.Lock()
	lk := l.locks[id]
	delete(l.locks, id)
	l.mtx.Unlock()

	lk.Unlock()
}
