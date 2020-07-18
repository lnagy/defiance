package eval

import (
	"fmt"
	"strings"
	"testing"
)

func TestEval(t *testing.T) {
	tests := []struct {
		expressions string
		correct     bool
		expected    string
	}{
		// Test 0
		{":1 = ap ap cons 7 ap ap cons 123229502148636 nil", true,
			"[ 7 :: [ 123229502148636 :: nil ] ]"},
		// Test 1
		{":1 = ap ap add 7 2", true, "9"},
		// Test 2
		{":1 = ap add 7", true,
			"(X0.add(7, X0))"},
		// Test 3
		{":1 = ap ap mul 7 2", true, "14"},
		// Test 4
		{":1 = ap ap div 7 2", true, "3"},
		// Test 5
		{":1 = ap ap div 7 -2", true, "-3"},
		// Test 6
		{":1 = ap ap add ap ap mul 7 2 6", true, "20"},
		// Test 7
		{":1 = ap ap add 6 ap ap mul 7 2", true, "20"},
		// Test 8
		{":1 = ap ap cons 7 ap ap cons 123229502148636 nil\n:2 = ap isnil :1", true,
			"f"},
		// Test 9
		{":1 = nil\n:2 = ap isnil :1", true, "t"},
		// Test 10
		{":1 = ap ap eq 0 7", true, "f"},
		// Test 11
		{":1 = ap ap eq ap ap add 2 5 7", true, "t"},
		// Test 12
		{":1 = ap ap lt 0 7", true, "t"},
		// Test 13
		{":1 = ap ap lt ap ap add 2 5 7", true, "f"},
		// Test 14
		{":1 = ap neg ap ap add 2 5", true, "-7"},
		// Test 15
		{":1 = ap ap t t ap ap add 2 5", true, "t"},
		// Test 16
		{":1 = ap ap f t ap ap add 2 5", true, "7"},
	}
	for testId, test := range tests {
		var parser Parser
		node, err := parser.Parse(test.expressions)
		correct := err == nil
		failed := false
		if correct != test.correct {
			t.Errorf("Test %v:\n%v\n====\n%v", testId, test.expressions, node)
			t.Errorf("Test %v: Expected correct: %v, got: %v", testId, test.correct, err)
			failed = true
		} else if test.correct && node == nil {
			t.Errorf("Test %v:\n%v\n====\n%v", testId, test.expressions, node)
			t.Errorf("Test %v: Failed to parse", testId)
			failed = true
		}
		if node != nil {
			reducer := NewReducer(node, true)
			result, reduceErr := reducer.Reduce(reducer.root)
			if reduceErr != nil {
				t.Logf("Test %v:\n%v\n====\n%v", testId, test.expressions, node)
				t.Logf("Test %v: Reduction Steps (calls: %v):\n%v", testId, reducer.stepCount, strings.Join(reducer.steps, "\n"))
				t.Errorf("Test %v: Failed to reduce: %v", testId, reduceErr)
				failed = true
			} else if fmt.Sprint(result) != test.expected {
				t.Logf("Test %v:\n%v\n====\n%v", testId, test.expressions, node)
				t.Logf("Test %v: Reduction Steps (calls: %v):\n%v", testId, reducer.stepCount, strings.Join(reducer.steps, "\n"))
				t.Errorf("Test %v: Failed to reduce. expected: %v, got: %v", testId, test.expected, result)
				failed = true
			}
		}
		if failed {
			break
		}
	}
}
