package application

import "sync"

type LogRegistry struct {
	m sync.Map
}


func NewLogRegistry() *LogRegistry {
	return &LogRegistry{}
}

func (r *LogRegistry) Register(projectID string, ch chan LogLine) {
	r.m.Store(projectID, ch)
}
 
func (r *LogRegistry) Get(projectID string) (chan LogLine, bool) {
	v, ok := r.m.Load(projectID)
	if !ok {
		return nil, false
	}
	return v.(chan LogLine), true
}
 
func (r *LogRegistry) Delete(projectID string) {
	r.m.Delete(projectID)
}
 