package main

import (
	"sync"
)

type Cache interface {
	Put(key string, value string) string
	Get(key string) string
	Show()
}

type LRUNode struct {
	key       string
	data      string
	nextBlock *LRUNode
	prevBlock *LRUNode
}

type LRUCache struct {
	nodeCache map[string]*LRUNode
	MFUNode   *LRUNode
	LRUNode   *LRUNode
	capacity  int // Total Capacity this cache can hold
	mu        sync.Mutex
}

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		nodeCache: make(map[string]*LRUNode),
		capacity:  capacity,
		MFUNode:   nil,
		LRUNode:   nil,
	}
}

func (c *LRUCache) Put(key string, value string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Check if this key exists already. If not make a node
	if _, ok := c.nodeCache[key]; ok {
		c.nodeCache[key].data = value
		c.re_rank(key)
		// fmt.Println("Already cached, re-ranked")
		return value
	}
	// Check if capacity if full
	if len(c.nodeCache) == c.capacity {
		// evict the LRUNode
		lruNode := c.LRUNode
		// fmt.Println("EVICTED -> ", lruNode.data)
		if lruNode.prevBlock != nil {
			c.LRUNode = lruNode.prevBlock
			lruNode.prevBlock.nextBlock = nil
			lruNode.prevBlock = nil
		} else {
			c.LRUNode = nil
			c.MFUNode = nil
		}
		delete(c.nodeCache, lruNode.key)
	}
	c.nodeCache[key] = &LRUNode{
		key:       key,
		data:      value,
		nextBlock: nil,
		prevBlock: nil,
	}
	nodeBlock := c.nodeCache[key]
	if c.MFUNode == nil {
		c.MFUNode = nodeBlock
		c.LRUNode = nodeBlock
	} else {
		nodeBlock.nextBlock = c.MFUNode
		c.MFUNode.prevBlock = nodeBlock
		c.MFUNode = nodeBlock
	}
	// fmt.Println("Successfully cached")
	return value
}

func (c *LRUCache) re_rank(key string) {
	nodeBlock := c.nodeCache[key]
	// Check if nodeBlock is already the MFUNode
	if c.MFUNode == nodeBlock {
		return
	}
	// Check if nodeBlock is the LRUNode
	if c.LRUNode == nodeBlock {
		lruNode := c.LRUNode
		c.LRUNode = lruNode.prevBlock
		lruNode.prevBlock.nextBlock = nil
		lruNode.prevBlock = nil
		nodeBlock.nextBlock = c.MFUNode
	} else {
		// Node is somewhere in the middle of the list
		nodeBlock.prevBlock.nextBlock = nodeBlock.nextBlock
		nodeBlock.nextBlock.prevBlock = nodeBlock.prevBlock
		nodeBlock.prevBlock = nil
		nodeBlock.nextBlock = c.MFUNode
	}
	c.MFUNode.prevBlock = nodeBlock
	c.MFUNode = nodeBlock
}

func (c *LRUCache) Get(key string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.nodeCache[key]; !ok {
		return "NOT FOUND"
	}
	c.re_rank(key)
	return c.nodeCache[key].data
}

func (c *LRUCache) Show() {
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.MFUNode
	count := 1
	for current != nil {
		// fmt.Printf("Rank %d -  %s\n", count, current.key)
		count++
		current = current.nextBlock
	}
}

func main() {

}
