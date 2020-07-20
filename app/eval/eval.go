package eval

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/bits"
	"os"
	"strconv"
	"strings"
)

type NodeType int

const (
	Ap NodeType = iota
	Lambda
	Fun
	Num
	Cons
	Closure
	Ref
)

type Node struct {
	fun       *Node
	nodes     []*Node
	nodeType  NodeType
	funName   string
	num       int64
	bound     string // Lambda-bound reference.
	modulated string // 0s and 1s
}

func (n *Node) Clone() *Node {
	if n == nil {
		return nil
	}
	clone := &Node{fun: n.fun.Clone(), nodeType: n.nodeType, funName: n.funName, num: n.num, bound: n.bound}
	for _, node := range n.nodes {
		clone.nodes = append(clone.nodes, node.Clone())
	}
	return clone
}

// Instantiate returns a selective copy where the lambda variable has been replaced.
func (n *Node) Instantiate(ref string, sub *Node) *Node {
	//log.Printf("Instantiate called on %v  with  %v  ->  %v", n, ref, sub)
	if n == nil {
		return nil
	}
	type queuedNode struct {
		path      []int
		pathNodes []*Node
		node      *Node
	}
	queue := []queuedNode{{node: n}}
	var refItem queuedNode
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if item.node.nodeType == Ref && item.node.funName == ref {
			refItem = item
			break
		}
		for pos, child := range item.node.nodes {
			childItem := queuedNode{node: child}
			childItem.path = append(item.path, pos)
			childItem.pathNodes = append(item.pathNodes, item.node)
			queue = append(queue, childItem)
		}
		if item.node.fun != nil {
			childItem := queuedNode{node: item.node.fun}
			childItem.path = append(item.path, -1)
			childItem.pathNodes = append(item.pathNodes, item.node)
			queue = append(queue, childItem)
		}
	}
	if refItem.node == nil {
		//log.Println("WTF?")
		return n
	}
	path := refItem.path
	pathNodes := refItem.pathNodes
	clone := sub
	for len(pathNodes) > 0 {
		parentNode := pathNodes[len(pathNodes)-1]
		childPos := path[len(path)-1]
		parentClone := &Node{
			nodeType: parentNode.nodeType, funName: parentNode.funName, num: parentNode.num, bound: parentNode.bound}
		for pos, child := range parentNode.nodes {
			if childPos == pos {
				parentClone.nodes = append(parentClone.nodes, clone)
			} else {
				parentClone.nodes = append(parentClone.nodes, child)
			}
		}
		if childPos == -1 {
			parentClone.fun = clone
		} else {
			parentClone.fun = parentNode.fun
		}
		clone = parentClone
		path = path[0 : len(path)-1]
		pathNodes = pathNodes[0 : len(pathNodes)-1]
	}
	//log.Printf("Instantiated: %v", clone)
	return clone
}

var visited = make(map[*Node]bool)
var topCall = true
var printAddr = flag.Bool("print_expr_addr", false,
	"Print address of expression nodes.")
var ShowSharing = flag.Bool("show_expr_sharing", false,
	"Print address of expression nodes.")

func (n *Node) String() string {
	if *ShowSharing {
		if topCall {
			visited = make(map[*Node]bool)
			topCall = false
			defer func() {
				topCall = true
			}()
		}
		if visited[n] {
			return fmt.Sprintf("{%p}", n)
		}
		visited[n] = true
	}
	if n == nil {
		return "<nil>"
	}
	if n.modulated != "" {
		return n.modulated
	}
	switch n.nodeType {
	case Ref:
		return fmt.Sprintf("%v", n.funName)
	case Num:
		return fmt.Sprintf("%v", n.num)
	case Fun:
		return n.funName
	case Lambda:
		//log.Printf("Printing lambda with bound '%v'", n.bound)
		if *printAddr {
			return fmt.Sprintf("%p|(%v.%v)", n, n.bound, n.fun)
		} else {
			return fmt.Sprintf("(%v.%v)", n.bound, n.fun)
		}
	case Cons:
		{
			if len(n.nodes) != 2 {
				return fmt.Sprintf("<Corrupted CONS: %v node(s)>", len(n.nodes))
			} else {
				return fmt.Sprintf("[ %v :: %v ]", n.nodes[0], n.nodes[1])
			}
		}
	case Closure:
		{
			//log.Printf("Printing closure with function '%v'", n.funName)
			var args []string
			for _, node := range n.nodes {
				args = append(args, fmt.Sprint(node))
			}
			if *printAddr {
				return fmt.Sprintf("%p|%v(%v)", n, n.funName, strings.Join(args, ", "))
			} else {
				return fmt.Sprintf("%v(%v)", n.funName, strings.Join(args, ", "))
			}
		}
	case Ap:
		{
			if len(n.nodes) != 1 {
				return fmt.Sprintf("<Corrupted AP: %v node(s)>", len(n.nodes))
			} else {
				if *printAddr {
					return fmt.Sprintf("%p|(%v %v)", n, n.fun, n.nodes[0])
				} else {
					return fmt.Sprintf("(%v %v)", n.fun, n.nodes[0])
				}
			}
		}
	default:
		return fmt.Sprintf("<Unknown NodeType: %v>", n.nodeType)
	}
}

// NodeCount() returns the size of the subtree rooted at n.
func (n *Node) NodeCount() int {
	visited := make(map[*Node]bool)
	queue := []*Node{n}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if visited[node] {
			continue
		}
		visited[node] = true
		if node.fun != nil && !visited[node.fun] {
			queue = append(queue, node.fun)
		}
		for _, child := range node.nodes {
			if !visited[child] {
				queue = append(queue, child)
			}
		}
	}
	return len(visited)
}

type Parser struct {
	Vars           map[string]*Node
	parsingVar     string
	NodeCount      int
	RecursiveCount int // Number of recursive definitions.
}

func (p *Parser) ParseAp(tokens []string, pos int) (*Node, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, errors.New(fmt.Sprintf("out of tokens at %v", pos))
	}
	node := &Node{}
	if fun, rem1, err := p.ParseExp(tokens, pos); err != nil {
		return nil, rem1, err
	} else {
		node.fun = fun
		if arg, rem2, err := p.ParseExp(rem1, pos+(len(tokens)-len(rem1))); err != nil {
			return nil, rem2, err
		} else {
			node.nodes = append(node.nodes, arg)
			return node, rem2, nil
		}
	}
}

func (p *Parser) ParseExp(tokens []string, pos int) (*Node, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, errors.New(fmt.Sprintf("out of tokens at %v", pos))
	}
	if tokens[0] == "ap" {
		if node, rem, err := p.ParseAp(tokens[1:], pos+1); err != nil {
			return nil, rem, err
		} else {
			return node, rem, nil
		}
	}
	p.NodeCount += 1
	if []rune(tokens[0])[0] == ':' {
		if p.parsingVar == tokens[0] {
			p.RecursiveCount += 1
			p.parsingVar = ""
		}
		return &Node{nodeType: Ref, funName: tokens[0]}, tokens[1:], nil
	}
	if num, err := strconv.ParseInt(tokens[0], 10, 64); err == nil {
		return &Node{nodeType: Num, num: num}, tokens[1:], nil
	}
	// Otherwise it must be a function name.
	return &Node{nodeType: Fun, funName: tokens[0]}, tokens[1:], nil
}

func (p *Parser) Parse(exp string) (*Node, error) {
	p.Vars = make(map[string]*Node)
	lines := strings.Split(exp, "\n")
	var lastNode *Node
	for row, line := range lines {
		if line == "" {
			continue
		}
		tokens := strings.Split(line, " ")
		p.parsingVar = tokens[0]
		if len(tokens) < 3 {
			return nil, errors.New(fmt.Sprintf("line %v: not enough tokens: %v", row+1, line))
		}
		node, rem, err := p.ParseExp(tokens[2:], 2)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("line %v: %v", row+1, err))
		}
		if len(rem) > 0 {
			return nil, errors.New(fmt.Sprintf("line %v: unparsed leftover %v", row+1, rem))
		}
		p.Vars[tokens[0]] = node
		lastNode = node
	}
	return lastNode, nil
}

type Reducer struct {
	Root         *Node
	steps        []string
	stepCount    int
	MaxStepCount int
	keepSteps    bool
	lambdas      int
	vars         map[string]*Node
	PrintSteps   bool
	clones       int
	prevStep     string
}

func common(prev, next string) (pfx, changed, sfx string) {
	prevBytes := []byte(prev)
	nextBytes := []byte(next)
	pfxLength := 0
	for pos := 0; pos < len(prevBytes) && pos < len(nextBytes); pos += 1 {
		if prevBytes[pos] == nextBytes[pos] {
			pfxLength += 1
		} else {
			break
		}
	}
	sfxLength := 0
	for pos := 1; pos < len(prevBytes) && pos < len(nextBytes); pos += 1 {
		if prevBytes[len(prevBytes)-pos] == nextBytes[len(nextBytes)-pos] {
			sfxLength += 1
		} else {
			break
		}
	}
	if pfxLength > 6 {
		pfx = fmt.Sprintf("{%v}==  ", pfxLength)
	} else {
		pfxLength = 0
	}
	if sfxLength > 6 {
		sfx = fmt.Sprintf("  =={%v}", sfxLength)
	} else {
		sfxLength = 0
	}
	changed = string(nextBytes[pfxLength : len(nextBytes)-sfxLength])
	return
}

func (r *Reducer) RecordStep() {
	if r.keepSteps {
		r.steps = append(r.steps, fmt.Sprint(r.Root))
	}
	if r.PrintSteps {
		visual := fmt.Sprint(r.Root)
		pfx, changed, sfx := common(r.prevStep, visual)
		r.prevStep = visual
		_, err := fmt.Fprintf(os.Stderr, "#%v  -->  %v%v%v\n\n", r.stepCount, pfx, changed, sfx)
		if err != nil {
			// Do nothing.
		}
	}
}

func (p *Parser) NewReducer(node *Node, keepSteps bool) *Reducer {
	reducer := &Reducer{Root: node, keepSteps: keepSteps}
	reducer.RecordStep()
	reducer.vars = p.Vars
	return reducer
}

func (r *Reducer) newVarName() string {
	varName := fmt.Sprint("X", r.lambdas)
	r.lambdas += 1
	return varName
}

func modulate(num int64) string {
	var bytes []byte
	if num == 0 {
		return "010"
	}
	if num > 0 {
		bytes = append(bytes, []byte("01")...)
	} else {
		bytes = append(bytes, []byte("10")...)
		num = -num
	}
	bitsNeeded := (64 - bits.LeadingZeros64(uint64(num)) + 3) / 4
	bytes = append(bytes, []byte(strings.Repeat("1", bitsNeeded)+"0")...)
	bytes = append(bytes, []byte(fmt.Sprintf("%0[1]*[2]b", bitsNeeded*4, num))...)
	return string(bytes)
}

func (r *Reducer) ReduceFunction(n *Node) (*Node, error) {
	if n.fun.nodeType != Fun {
		return nil, errors.New(fmt.Sprintf("expected function node: %v", n))
	}
	if len(n.nodes) != 1 {
		return nil, errors.New(fmt.Sprintf("function node expects exactly one arg: %v", n))
	}
	switch n.fun.funName {
	case "f": // First argument ignored.
		n.nodes[0] = &Node{nodeType: Fun, funName: "_"}
	case "if0", "mod", "neg", "inc", "dec", "isnil", "car", "cdr", "double":
		// Functions strict in first argument.
		for {
			if arg, err := r.Reduce(n.nodes[0]); err != nil {
				return nil, err
			} else {
				if n.nodes[0] != arg {
					n.nodes[0] = arg
					r.RecordStep()
				} else {
					break
				}
			}
		}
	}
	switch n.fun.funName {
	case "nil":
		return &Node{nodeType: Fun, funName: "t"}, nil
	case "neg", "inc", "dec", "mod":
		if n.nodes[0].nodeType != Num {
			return nil, errors.New(fmt.Sprintf("expected single numeric argument: %v", n))
		} else {
			switch n.fun.funName {
			case "neg":
				return &Node{nodeType: Num, num: -n.nodes[0].num}, nil
			case "inc":
				return &Node{nodeType: Num, num: n.nodes[0].num + 1}, nil
			case "dec":
				return &Node{nodeType: Num, num: n.nodes[0].num - 1}, nil
			case "mod":
				return &Node{nodeType: Num, num: n.nodes[0].num, modulated: modulate(n.nodes[0].num)}, nil
			}
		}
	case "isnil":
		if n.nodes[0].funName == "nil" {
			return &Node{nodeType: Fun, funName: "t"}, nil
		} else {
			return &Node{nodeType: Fun, funName: "f"}, nil
		}
	case "car":
		if n.nodes[0].nodeType != Cons {
			return nil, errors.New(fmt.Sprintf("'car' expects CONS: %v", n))
		} else {
			return n.nodes[0].nodes[0], nil
		}
	case "cdr":
		if n.nodes[0].nodeType != Cons {
			return nil, errors.New(fmt.Sprintf("'cdr' expects CONS: %v", n))
		} else {
			return n.nodes[0].nodes[1], nil
		}
	case "cons", "mul", "div", "add", "eq", "lt", "t", "f":
		{
			var node *Node
			if n.fun.funName == "cons" {
				node = &Node{nodeType: Cons}
			} else {
				node = &Node{nodeType: Closure, funName: n.fun.funName}
			}
			node.nodes = append(node.nodes, n.nodes[0])
			var varName string
			if n.fun.funName == "t" {
				// Second argument is ignored.
				varName = "_"
				node.nodes = append(node.nodes, &Node{nodeType: Ref, funName: "_"})
			} else {
				varName = r.newVarName()
				node.nodes = append(node.nodes, &Node{nodeType: Ref, funName: varName})
			}
			return &Node{nodeType: Lambda, fun: node, bound: varName}, nil
		}

	case "double":
		{
			varName := r.newVarName()
			node := &Node{nodeType: Ap, fun: n.nodes[0], nodes: []*Node{{nodeType: Ap, fun: n.nodes[0]}}}
			node.nodes[0].nodes = append(node.nodes[0].nodes, &Node{nodeType: Ref, funName: varName})
			return &Node{nodeType: Lambda, fun: node, bound: varName}, nil
		}
	case "s", "c", "b", "if0":
		{
			closure := &Node{nodeType: Closure, funName: n.fun.funName}
			closure.nodes = append(closure.nodes, n.nodes[0])
			firstArg := r.newVarName()
			secondArg := r.newVarName()
			if n.fun.funName == "if0" {
				if n.nodes[0].nodeType != Num {
					return nil, errors.New(fmt.Sprintf("'if0' expects numeric first argument: %v", n))
				}
				if n.nodes[0].num == 0 {
					secondArg = "_"
				} else {
					firstArg = "_"
				}
			}
			closure.nodes = append(closure.nodes, &Node{nodeType: Ref, funName: firstArg})
			closure.nodes = append(closure.nodes, &Node{nodeType: Ref, funName: secondArg})
			return &Node{nodeType: Lambda, fun: &Node{nodeType: Lambda, fun: closure, bound: secondArg},
				bound: firstArg}, nil
		}
	case "i":
		return n.nodes[0], nil
	default:
		return nil, errors.New(fmt.Sprintf("unimplemented function: %v", n))
	}
	return nil, errors.New(fmt.Sprintf("unreachable: %v", n))
}

func isTerminal(nt NodeType) bool {
	switch nt {
	case Num, Lambda, Fun:
		return true
	default:
		return false
	}
}

func (r *Reducer) ReduceRoot() (*Node, error) {
	for !isTerminal(r.Root.nodeType) {
		node, err := r.Reduce(r.Root)
		if err != nil {
			return nil, err
		}
		if r.Root != node {
			r.Root = node
			r.RecordStep()
		} else {
			break
		}
	}
	// Make lists strict.
	maxSteps := 200000
	steps := 0
	//r.PrintSteps = true
	if r.Root.nodeType == Cons {
		queue := []**Node{&r.Root.nodes[0], &r.Root.nodes[1]}
		//log.Printf("\n====Adding to queue (head): %v\n", r.Root.nodes[0])
		//log.Printf("\n====Adding to queue (tail): %v\n", r.Root.nodes[1])
		for len(queue) > 0 {
			steps += 1
			if steps > maxSteps {
				break
			}
			node := queue[0]
			queue = queue[1:]
			//log.Printf("============ Reducing to terminal state: %v", *node)
			if len(queue) == 7 && (*node).funName == "b" {
				//r.PrintSteps = true
			}
			for !isTerminal((*node).nodeType) {
				reduced, err := r.Reduce(*node)
				if err != nil {
					return nil, err
				}
				if *node != reduced {
					*node = reduced
					r.RecordStep()
				} else {
					break
				}
			}
			if (*node).nodeType == Cons {
				queue = append(queue, &(*node).nodes[0])
				queue = append(queue, &(*node).nodes[1])
				//log.Printf("\n====Adding to queue (head): %v\n", (*node).nodes[0])
				//log.Printf("\n====Adding to queue (tail): %v\n", (*node).nodes[1])
				//log.Printf("\n====Queue length: %v\n", len(queue))
			}
		}
	}
	return r.Root, nil
}

func (r *Reducer) Reduce(n *Node) (*Node, error) {
	//log.Printf("Reducing: %v", n)
	if n == nil {
		return nil, nil
	}
	r.stepCount += 1
	if r.MaxStepCount > 0 && r.stepCount > r.MaxStepCount {
		return nil, errors.New(fmt.Sprintf("Reached max step count: %v", r.MaxStepCount))
	}
	if r.stepCount%5000 == 0 {
		log.Printf("Step: %v  Node Count: %v", r.stepCount, r.Root.NodeCount())
	}
	switch n.nodeType {
	case Ref:
		node, ok := r.vars[n.funName]
		if !ok {
			return nil, errors.New(fmt.Sprintf("unknown id: %v", n.funName))
		}
		r.clones += 1
		//log.Printf("Cloned: %v", n.funName)
		//if r.clones > 20000 {
		//	log.Fatal("too many clones")
		//}
		return node.Clone(), nil
	case Num, Fun, Lambda:
		return n, nil
	case Cons:
		if head, err := r.Reduce(n.nodes[0]); err != nil {
			return nil, err
		} else {
			if n.nodes[0] != head {
				n.nodes[0] = head
				r.RecordStep()
			}
		}
		if tail, err := r.Reduce(n.nodes[1]); err != nil {
			return nil, err
		} else {
			if n.nodes[1] != tail {
				n.nodes[1] = tail
				r.RecordStep()
			}
		}
		return n, nil
	case Ap:
		switch {
		case n.fun == nil:
			return nil, errors.New(fmt.Sprintf("fun is nil: %v", n))
		case n.fun.nodeType == Cons:
			// log.Printf("applying cons")
			node := &Node{nodeType: Ap, fun: &Node{nodeType: Ap, fun: n.nodes[0], nodes: []*Node{n.fun.nodes[0]}},
				nodes: []*Node{n.fun.nodes[1]}}
			return node, nil
		case n.fun.nodeType == Ap || n.fun.nodeType == Ref || n.fun.nodeType == Closure:
			if fun, err := r.Reduce(n.fun); err != nil {
				return nil, err
			} else {
				if fun == nil {
					return nil, errors.New(fmt.Sprintf("'fun' reduction is nil: %v", n))
				}
				n.fun = fun
				r.RecordStep()
				return r.Reduce(n)
			}
		case n.fun.nodeType == Lambda:
			{
				if n.fun.bound == "" {
					return nil, errors.New(fmt.Sprintf("no bound variable: %v", n))
				}
				if len(n.nodes) != 1 {
					return nil, errors.New(fmt.Sprintf("lambda expects exactly one arg: %v", n))
				}
				instantiated := n.fun.fun
				// Only use the argument if it's not discarded.
				if n.fun.bound != "_" {
					instantiated = n.fun.fun.Instantiate(n.fun.bound, n.nodes[0])
				}
				n.nodes[0] = instantiated
				n.fun = &Node{nodeType: Fun, funName: "i"}
				r.RecordStep()
				return r.Reduce(n)
			}
		default:
			return r.ReduceFunction(n)
		}
	case Closure:
		{
			switch n.funName {
			case "add", "mul", "div", "eq", "lt":
				{
					for pos, node := range n.nodes {
						if node.nodeType == Num {
							continue
						}
						for {
							if arg, err := r.Reduce(node); err != nil {
								return nil, err
							} else {
								if n.nodes[pos] != arg {
									node = arg
									n.nodes[pos] = arg
									r.RecordStep()
								} else {
									break
								}
							}
						}
					}
					if n.nodes[0].nodeType != Num || n.nodes[1].nodeType != Num {
						return nil, errors.New(fmt.Sprintf("expected two numeric arguments: %v", n))
					}
					switch n.funName {
					case "add":
						return &Node{nodeType: Num, num: n.nodes[0].num + n.nodes[1].num}, nil
					case "mul":
						return &Node{nodeType: Num, num: n.nodes[0].num * n.nodes[1].num}, nil
					case "div":
						return &Node{nodeType: Num, num: n.nodes[0].num / n.nodes[1].num}, nil
					case "eq":
						if n.nodes[0].num == n.nodes[1].num {
							return &Node{nodeType: Fun, funName: "t"}, nil
						} else {
							return &Node{nodeType: Fun, funName: "f"}, nil
						}
					case "lt":
						if n.nodes[0].num < n.nodes[1].num {
							return &Node{nodeType: Fun, funName: "t"}, nil
						} else {
							return &Node{nodeType: Fun, funName: "f"}, nil
						}
					}
				}
			case "if0":
				if n.nodes[0].num == 0 {
					return n.nodes[1], nil
				} else {
					return n.nodes[2], nil
				}
			case "t":
				return n.nodes[0], nil
			case "f":
				return n.nodes[1], nil
			case "s":
				return &Node{nodeType: Ap, fun: &Node{nodeType: Ap, fun: n.nodes[0], nodes: []*Node{n.nodes[2]}},
					nodes: []*Node{{nodeType: Ap, fun: n.nodes[1], nodes: []*Node{n.nodes[2]}}}}, nil
			case "c":
				return &Node{nodeType: Ap, fun: &Node{nodeType: Ap, fun: n.nodes[0], nodes: []*Node{n.nodes[2]}},
					nodes: []*Node{n.nodes[1]}}, nil
			case "b":
				return &Node{nodeType: Ap, fun: n.nodes[0],
					nodes: []*Node{{nodeType: Ap, fun: n.nodes[1], nodes: []*Node{n.nodes[2]}}}}, nil
			}
		}
	}
	return nil, errors.New(fmt.Sprintf("unimplemented: %v", n))
}
