// Package btree implements a B-Tree (used as a warm up exercise).
package btree

import (
	"errors"
	"fmt"
	"log"
)

type node struct {
	nodes  []*node
	keys   []int
	values []string
}

type BTree struct {
	maxChildren int
	size        int
	tree        *node
}

func New(maxChildren int) *BTree {
	return &BTree{maxChildren: maxChildren}
}

func (t *BTree) String() string {
	show := func(indent int, n *node) string {
		return fmt.Sprintf("%[1]*[2]v %[3]p: %[4]v\n", indent, "", n, *n)
	}
	indent := 0
	queue := []*node{t.tree}
	rv := fmt.Sprintf("max: %v, size: %v\n", t.maxChildren, t.size)
	level := 1
	nextLevel := 0
	for len(queue) > 0 {
		n := queue[0]
		rv += show(indent, n)
		if len(n.values) == 0 {
			queue = append(queue, n.nodes...)
			nextLevel += len(n.nodes)
		}
		level -= 1
		queue = queue[1:]
		if level == 0 {
			indent += 2
			level = nextLevel
			nextLevel = 0
		}
	}
	return rv
}

func (t *BTree) Set(k int, v string) {
	if t.tree == nil {
		t.size = 1
		t.tree = &node{}
		n := t.tree
		n.keys = make([]int, 1, t.maxChildren)
		n.values = make([]string, 1, cap(n.keys))
		n.keys[0] = k
		n.values[0] = v
		return
	}
	path := find(k, t.tree)
	n := path[len(path)-1]
	pos := 0
	for ; pos < len(n.keys); pos += 1 {
		if n.keys[pos] == k {
			n.values[pos] = v
			return
		}
		if k > n.keys[pos] {
			continue
		} else {
			break
		}
	}
	t.size += 1
	n.keys = append(n.keys, 0)
	copy(n.keys[pos+1:], n.keys[pos:])
	n.keys[pos] = k
	n.values = append(n.values, "")
	copy(n.values[pos+1:], n.values[pos:])
	n.values[pos] = v
	pathPos := len(path) - 1
	if len(n.values) > t.maxChildren {
		split := len(n.values) / 2
		splitKey := n.keys[split]
		right := &node{keys: make([]int, split), values: make([]string, split)}
		copy(right.keys, n.keys[split:])
		copy(right.values, n.values[split:])
		n.keys = n.keys[:split]
		n.values = n.values[:split]
		if pathPos > 0 {
			parent := path[pathPos-1]
			for parentPos, child := range parent.nodes {
				if child == n {
					insertPos := parentPos
					log.Println("insertPos: ", insertPos, "  len(parent.keys): ", len(parent.keys))
					parent.keys = append(parent.keys, 0)
					copy(parent.keys[insertPos+1:], parent.keys[insertPos:])
					parent.keys[insertPos] = splitKey
					insertPos += 1
					parent.nodes = append(parent.nodes, nil)
					copy(parent.nodes[insertPos+1:], parent.nodes[insertPos:])
					parent.nodes[insertPos] = right
					break
				}
			}
		} else {
			t.tree = &node{keys: []int{splitKey}, nodes: []*node{n, right}}
		}
	}

}

// find returns the path to the node where the key k is to be inserted.
func find(k int, n *node) (path []*node) {
	path = append(path, n)
	for {
		if len(n.values) > 0 {
			return
		}
		found := false
		for pos, key := range n.keys {
			if k < key {
				n = n.nodes[pos]
				found = true
				break
			}
		}
		if !found {
			n = n.nodes[len(n.nodes)-1]
		}
		path = append(path, n)
	}
}

func (t *BTree) Get(k int) (string, error) {
	if t.tree == nil {
		return "", errors.New("missing key")
	}
	path := find(k, t.tree)
	n := path[len(path)-1]
	for pos, key := range n.keys {
		if key == k {
			return n.values[pos], nil
		}
	}
	return "", errors.New("missing key")
}
