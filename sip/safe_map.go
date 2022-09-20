package sip

import "sync"

// SafeMap Thread safe map
type SafeMap struct {
	containers map[string]interface{}
	mutex      sync.RWMutex
}

func CreateSafeMap(capacity int) *SafeMap {
	safeMap := &SafeMap{
		containers: make(map[string]interface{}, capacity),
	}

	return safeMap
}

func (t *SafeMap) Find(id string) (interface{}, bool) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	element, ok := t.containers[id]
	return element, ok
}

func (t *SafeMap) Remove(id string) (interface{}, bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	i, b := t.containers[id]
	if b {
		delete(t.containers, id)
	}
	return i, b
}

func (t *SafeMap) Add(id string, transaction interface{}) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.containers[id] = transaction
}

func (t *SafeMap) Iterator(callback func(key string, e interface{})) {
	for s, i := range t.containers {
		callback(s, i)
	}
}

func (t *SafeMap) Size() int {
	return len(t.containers)
}

func (t *SafeMap) Clear() {
	t.containers = make(map[string]interface{})
}
