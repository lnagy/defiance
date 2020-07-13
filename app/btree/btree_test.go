package btree

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type opType int

const (
	opSet opType = iota
	opErase
	opFill
)

type op struct {
	opType opType
	key    int
	value  string
}

func TestOperations(t *testing.T) {
	tests := []struct {
		ops      []op
		expected map[int]string
	}{
		// Test 0
		{[]op{{opSet, 10, "hello"}}, map[int]string{10: "hello"}},
		// Test 1
		{[]op{{opSet, 10, "hello"}, {opSet, 10, "world"}},
			map[int]string{10: "world"}},
		// Test 2
		{[]op{{opSet, 10, "hello"}, {opSet, 8, "world"}},
			map[int]string{8: "world", 10: "hello"}},
		// Test 3
		{[]op{
			{opSet, 10, "hello"},
			{opSet, 8, "world"},
			{opSet, 12, "nice"},
		},
			map[int]string{8: "world", 10: "hello", 12: "nice"}},
		// Test 4
		{expected: map[int]string{2: "hello", 4: "nice", 6: "world", 7: "good", 9: "morning", 11: "Budapest"}},
		// Test 5
		{ops: []op{{opType: opFill, key: 20}}},
	}
	for testId, test := range tests {
		bt := New(5)
		for _, op := range test.ops {
			switch op.opType {
			case opSet:
				{
					bt.Set(op.key, op.value)
				}
			case opFill:
				{
					test.expected = make(map[int]string)
					keys := make([]int, op.key)
					for k := 0; k < op.key; k += 1 {
						keys[k] = k
					}
					rand.Seed(time.Now().UnixNano())
					rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
					for _, k := range keys {
						bt.Set(k, fmt.Sprint("v", k))
						test.expected[k] = fmt.Sprint("v", k)
						t.Logf("Test %v: Tree:\n%v", testId, bt)
					}
				}
			case opErase:
				{
					// Do nothing.
				}
			}
		}
		if len(test.ops) == 0 {
			for k, v := range test.expected {
				bt.Set(k, v)
			}
		}
		for k, v := range test.expected {
			got, err := bt.Get(k)
			if err != nil || got != v {
				t.Errorf("Test %v: Get(%v). expected: %#v, got: %#v, error: %v", testId, k, v, got, err)
				t.Errorf("Tree:\n%v", bt)
				t.FailNow()
			}
		}
	}
}
