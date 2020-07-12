// Package qsort uses a B-Tree to sort integers (as a proof of concept
// for using modules).
package qsort

import "github.com/google/btree"

func Sort(xs []int) int {
	tree := btree.New(2)
	return tree.Len() + len(xs)
}
