package eval

import (
	"fmt"
	"strings"
	"testing"
)

func TestInstantiate(t *testing.T) {
	tests := []struct {
		node     *Node
		ref      string
		sub      *Node
		expected string
	}{
		// Test 0
		{&Node{nodeType: Ref, funName: "X0"}, "X0", &Node{nodeType: Num, num: 7}, "7"},
		// Test 1
		{&Node{nodeType: Ref, funName: "X0"}, "X1", &Node{nodeType: Num, num: 7}, "X0"},
		// Test 2
		{&Node{nodeType: Ap, fun: &Node{nodeType: Fun, funName: "neg"},
			Nodes: []*Node{{nodeType: Ref, funName: "X0"}}},
			"X0", &Node{nodeType: Num, num: 7}, "(neg 7)"},
		// Test 3
		{&Node{nodeType: Closure, funName: "add",
			Nodes: []*Node{{nodeType: Num, num: 8}, {nodeType: Ref, funName: "X0"}}},
			"X0", &Node{nodeType: Num, num: 7}, "add(8, 7)"},
		// Test 4
		{&Node{nodeType: Ap, fun: &Node{nodeType: Ref, funName: "X0"},
			Nodes: []*Node{{nodeType: Num, num: 8}}},
			"X0", &Node{nodeType: Fun, funName: "inc"}, "(inc 8)"},
	}
	for testId, test := range tests {
		//if testId != 7 {
		//	continue
		//}
		clone := test.node.Instantiate(test.ref, test.sub)
		if testId == 3 || testId == 4 {
			if test.node == clone {
				t.Fail()
				t.Errorf("Test %v:\n%v", testId, test.node)
				t.Errorf("Test %v: Failed to clone top node.", testId)
			}
			if test.node.Nodes[0] != clone.Nodes[0] {
				t.Fail()
				t.Errorf("Test %v:\n%v", testId, test.node)
				t.Errorf("Test %v: Failed to retain unaffected branch.", testId)
			}
		}
		if clone == nil {
			t.Fail()
			t.Errorf("Test %v:\n%v", testId, test.node)
			t.Errorf("Test %v: Failed to instantiate.", testId)
		} else {
			if got := fmt.Sprint(clone); got != test.expected {
				t.Errorf("Test %v:\n%v", testId, test.node)
				t.Errorf("Test %v: Expected instantiation: %v, got: %v", testId, test.expected, got)
			}
		}
	}
}

func TestClone(t *testing.T) {
	var parser Parser
	node, err := parser.Parse(":1 = ap ap cons 7 ap ap cons 123229502148636 nil")
	if err != nil {
		t.Fail()
	} else {
		clone := node.Clone()
		if fmt.Sprint(node) != fmt.Sprint(clone) {
			t.Errorf("Clone() failed. expected: %v, got: %v", node, clone)
		}
		if clone == node {
			t.Errorf("Clone() failed. pointer unchanged")
		}
	}
}

func TestNodeCount(t *testing.T) {
	tests := []struct {
		expressions string
		expected    string
		nodes       int
	}{
		// Test 0
		{":1 = 7", "7", 1},
		// Test 1
		{":1 = ap ap cons 7 ap ap cons 123229502148636 nil",
			"[ 7 :: [ 123229502148636 :: nil ] ]", 9},
		// Test 2
		{":1 = ap ap cons 7 nil", "[ 7 :: nil ]", 5},
		// Test 3
		{":1 = ap ap cons 7 :2\n:2 = ap ap cons 8 :1", "[ 7 :: nil ]", 5},
	}
	for testId, test := range tests {
		//if testId != 7 {
		//	continue
		//}
		var parser Parser
		node, err := parser.Parse(test.expressions)
		if err != nil {
			t.Fail()
			t.Errorf("Test %v:\n%v\n====\n%v", testId, test.expressions, node)
			t.Errorf("Test %v: Failed to parse: %v", testId, err)
		} else {
			if got := node.NodeCount(); got != test.nodes {
				t.Errorf("Test %v:\n%v\n====\n%v", testId, test.expressions, node)
				t.Errorf("Test %v: Expected node count: %v, got: %v", testId, test.nodes, got)
			}
		}
	}
}

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
		// Test 17
		{":1 = ap car ap ap cons 2 ap ap cons 5 nil", true, "2"},
		// Test 18
		{":1 = ap cdr ap ap cons 2 ap ap cons 5 nil", true, "[ 5 :: nil ]"},
		// Test 19
		{":1 = ap ap ap s add inc 1", true, "3"},
		// Test 20
		{":1 = ap ap ap s mul ap add 1 6", true, "42"},
		// Test 21
		{":1 = ap ap ap c add 1 2", true, "3"},
		// Test 22
		{":1 = ap ap ap b inc dec 7", true, "7"},
		// Test 23
		{":1 = ap ap add 7 :2\n:2 = -3\n:3 = :1", true, "4"},
		// Test 24
		{":1 = ap ap ap if0 0 3 7", true, "3"},
		// Test 25
		{":1 = ap ap ap if0 1 3 7", true, "7"},
		// Test 26
		{":1 = ap ap ap if0 ap dec 1 3 ap dec t", true, "3"},
		// Test 27
		{":1141 = ap ap c b ap ap s ap ap b c ap ap b ap b b ap eq 0 ap ap b ap c :1141 ap add -1\n:1 = :1141", true,
			"(X1.c(b, ((s ((b c) ((b (b b)) (eq 0)))) ((b (c :1141)) (add -1))), X1))"},
		// Test 28
		{":1 = ap ap ap cons 2 5 add", true, "7"},
		// Test 29
		{":1 = ap ap eq 3 ap i 7", true, "f"},
		// Test 30
		{":1 = ap ap eq 3 ap i 7", true, "f"},
		// Test 31
		{":1 = ap dec 7\n:2 = ap ap add ap inc :1 ap dec :1", true, "12"},
		// Test 32
		{":1 = ap ap double ap add 1 2", true, "4"},
		// Test 33
		{":1 = ap mod 0", true, "010"},
		// Test 34
		{":1 = ap mod 1", true, "01100001"},
		// Test 35
		{":1 = ap mod -1", true, "10100001"},
		// Test 36
		{":1 = ap mod -15", true, "10101111"},
		// Test 37
		{":1 = ap mod 16", true, "0111000010000"},
		// Test 38
		{":1 = ap mod -255", true, "1011011111111"},
		// Test 39
		{":1 = ap mod 256", true, "011110000100000000"},
		// Test 40
		{":1 = ap modlist nil", true, "00"},
		// Test 41
		{":1 = ap modlist ap ap cons nil nil", true, "110000"},
		// Test 42
		{":1 = ap modlist ap ap cons 0 nil", true, "1101000"},
		// Test 43
		{":1 = ap modlist ap ap cons 1 2", true, "110110000101100010"},
		// Test 44
		{":1 = ap modlist ap ap cons 1 ap ap cons 2 nil", true,
			"1101100001110110001000"},
		// Test 45
		{":1 = ap ap cons 1 ap ap cons 2 nil\n:2 = ap modlist ap ap cons 1 ap ap cons :1 ap ap cons 4 nil",
			true,
			"1101100001111101100001110110001000110110010000"},
		// Test 46
		{":1 = ap dem ap mod 0", true, "0"},
		// Test 47
		{":1 = ap dem ap mod 1", true, "1"},
		// Test 48
		{":1 = ap dem ap mod -1", true, "-1"},
		// Test 49
		{":1 = ap dem ap mod -15", true, "-15"},
		// Test 50
		{":1 = ap dem ap mod 16", true, "16"},
		// Test 51
		{":1 = ap dem ap mod -255", true, "-255"},
		// Test 52
		{":1 = ap dem ap mod 256", true, "256"},
	}
	for testId, test := range tests {
		//if testId != 32 {
		//	continue
		//}
		//log.Printf("==== Running Test %v", testId)
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
			clone := node.Clone()
			reducer := parser.NewReducer(node, true)
			reducer.MaxStepCount = 100
			//reducer.PrintSteps = true
			result, reduceErr := reducer.ReduceRoot()
			if reduceErr != nil {
				t.Logf("Test %v:\n%v\n====\n%v", testId, test.expressions, clone)
				t.Logf("Test %v: Reduction Steps (calls: %v):\n%v", testId, reducer.stepCount, strings.Join(reducer.steps, "\n"))
				t.Errorf("Test %v: Failed to reduce: %v", testId, reduceErr)
				failed = true
			} else if fmt.Sprint(result) != test.expected {
				t.Logf("Test %v:\n%v\n====\n%v", testId, test.expressions, clone)
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
