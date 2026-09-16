package main

import (
	"container/heap"
	"time"
)

type ExpiryItem struct {
	Key       string
	ExpiresAt time.Time
	Position  int
}

type ExpiryHeap struct {
	store []*ExpiryItem
}

func NewMinPriorityQueue() *ExpiryHeap {
	h := &ExpiryHeap{
		store: make([]*ExpiryItem, 0),
	}
	heap.Init(h)
	return h
}

func (h *ExpiryHeap) Len() int {
	return len(h.store)
}

func (h *ExpiryHeap) Less(i int, j int) bool {
	return h.store[i].ExpiresAt.Before(h.store[j].ExpiresAt)
}

func (h *ExpiryHeap) Swap(i int, j int) {
	h.store[i], h.store[j] = h.store[j], h.store[i]
	h.store[i].Position = i
	h.store[j].Position = j
}

func (h *ExpiryHeap) Push(x any) {
	item := x.(*ExpiryItem)
	item.Position = len(h.store)
	h.store = append(h.store, item)
}

func (h *ExpiryHeap) Pop() any {
	n := len(h.store)
	item := h.store[n-1]
	h.store[n-1] = nil
	h.store = h.store[:n-1]
	item.Position = -1
	return item
}

func (h *ExpiryHeap) Peek() *ExpiryItem {
	if len(h.store) == 0 {
		return nil
	}
	return h.store[0]
}
