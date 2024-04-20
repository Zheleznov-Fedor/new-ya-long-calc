package utils

import (
	"strconv"
	"sync"
)

type Counter struct {
	value int
	mu    sync.RWMutex
	Cnt   int
}

var Port = Counter{}

type Сount interface {
	GetValue() string
	SetCnt(cnt int)
}

func (c *Counter) GetValue() string {
	c.mu.RLock()
	data := 5000 + c.value
	c.value = c.value + 1
	if c.value == c.Cnt {
		c.value = 0
	}
	c.mu.RUnlock()
	return strconv.Itoa(data)
}

func (c *Counter) SetCnt(cnt int) {
	c.mu.RLock()
	c.Cnt = cnt
	c.mu.RUnlock()
}
