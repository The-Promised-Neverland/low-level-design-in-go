package main

type requestHeap []Request

func (h requestHeap) Len() int      { return len(h) }
func (h requestHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *requestHeap) Push(x any)   { *h = append(*h, x.(Request)) }
func (h *requestHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

type MinHeap struct{ requestHeap }

func (h MinHeap) Less(i, j int) bool { return h.requestHeap[i].Floor < h.requestHeap[j].Floor }

type MaxHeap struct{ requestHeap }

func (h MaxHeap) Less(i, j int) bool { return h.requestHeap[i].Floor > h.requestHeap[j].Floor }
