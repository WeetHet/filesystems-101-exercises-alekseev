package set

import (
	"encoding/json"
	"maps"
	"slices"
)

// Set represents an unordered set of objects
type Set[T comparable] map[T]struct{}

type StrSet = Set[string]

func NewStrSet(ss ...string) StrSet { return New(ss...) }

func (s Set[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.AsSlice())
}

func (s *Set[T]) UnmarshalJSON(bs []byte) error {
	var items []T
	err := json.Unmarshal(bs, &items)
	if err == nil {
		*s = New(items...)
		return nil
	}

	// set might have been marshalled before MarshalJSON was added
	// and is represented as `{"key1": {}, "key2": {}, "key3": {}}`.
	var m map[T]struct{}
	err = json.Unmarshal(bs, &m)
	if err != nil {
		return err
	}

	*s = m
	return nil
}

// Add adds an element to the set
func (s Set[T]) Add(es ...T) Set[T] {
	for _, e := range es {
		s[e] = struct{}{}
	}
	return s
}

// Remove removes an element from the set
func (s Set[T]) Remove(es ...T) Set[T] {
	for _, e := range es {
		delete(s, e)
	}
	return s
}

// Clear removes all elements in the set
func (s Set[T]) Clear() {
	clear(s)
}

// Contains returns whether the given item is in the set
func (s Set[T]) Contains(elem T) bool {
	_, contains := s[elem]
	return contains
}

// ContainsAny returns whether any of the given items is in the set
func (s Set[T]) ContainsAny(es ...T) bool {
	return slices.ContainsFunc(es, s.Contains)
}

// ContainsAll returns whether all of the given items are in the set
func (s Set[T]) ContainsAll(es ...T) bool {
	for _, elem := range es {
		if _, contains := s[elem]; !contains {
			return false
		}
	}
	return true
}

// Size returns the number of itmes in the set
func (s Set[T]) Size() int {
	return len(s)
}

// IsEmpty returns true iff the set contains no keys
func (s Set[T]) IsEmpty() bool {
	return len(s) == 0
}

// New creates a set
func New[T comparable](keys ...T) Set[T] {
	s := make(Set[T], len(keys))
	return s.Add(keys...)
}

// AsSlice converts Set to slice of its keys
func (s Set[T]) AsSlice() []T {
	result := make([]T, 0, len(s))
	for key := range s {
		result = append(result, key)
	}
	return result
}

// Difference calculates difference `s \ otherS` -- set
// of elements which are in the `s` but no in the `otherS`
func (s Set[T]) Difference(otherS Set[T]) Set[T] {
	result := New[T]()
	for e := range s {
		if !otherS.Contains(e) {
			result.Add(e)
		}
	}
	return result
}

// Intersection calculates intersection `s \cap otherS` -- set
// of elements which are in the `s` and in the `otherS`
func (s Set[T]) Intersection(otherS Set[T]) Set[T] {
	result := New[T]()
	if len(s) < len(otherS) {
		for e := range s {
			if otherS.Contains(e) {
				result.Add(e)
			}
		}
	} else {
		for e := range otherS {
			if s.Contains(e) {
				result.Add(e)
			}
		}
	}
	return result
}

// Equal returns true iff s1 and s2 are both nil or contains the same elements
func (s Set[T]) Equal(s2 Set[T]) bool {
	if len(s) != len(s2) {
		return false
	}

	for elem := range s {
		if !s2.Contains(elem) {
			return false
		}
	}

	return true
}

func (s Set[T]) Clone() Set[T] {
	return maps.Clone(s)
}
