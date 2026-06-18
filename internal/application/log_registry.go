package application

import "sync"

type LogRegistry struct {
	m sync.Map
}

