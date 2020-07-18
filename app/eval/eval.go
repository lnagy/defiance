package eval

import (
	"errors"
	"fmt"
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
)

type Node struct {
	fun      *Node
	nodes    []*Node
	nodeType NodeType
	funName  string
	num      int64
	bound    **Node // Lambda-bound.
}

func (n *Node) String() string {
	if n == nil {
		return "<nil>"
	}
	switch n.nodeType {
	case Num:
		return fmt.Sprintf("%v", n.num)
	case Fun:
		return n.funName
	case Lambda:
		return fmt.Sprintf("(%v.%v)", (*n.bound).funName, n.fun)
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
			var args []string
			for _, node := range n.nodes {
				args = append(args, fmt.Sprint(node))
			}
			return fmt.Sprintf("%v(%v)", n.funName, strings.Join(args, ", "))
		}
	case Ap:
		{
			if len(n.nodes) != 1 {
				return fmt.Sprintf("<Corrupted AP: %v node(s)>", len(n.nodes))
			} else if n.fun.funName == "root" {
				return fmt.Sprintf("%v", n.nodes[0])
			} else {
				return fmt.Sprintf("(%v %v)", n.fun, n.nodes[0])
			}
		}
	default:
		return fmt.Sprintf("<Unknown NodeType: %v>", n.nodeType)
	}
}

type Parser struct {
	vars map[string]*Node
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
	if []rune(tokens[0])[0] == ':' {
		node, ok := p.vars[tokens[0]]
		if !ok {
			return nil, nil, errors.New(fmt.Sprintf("token %v: unknown id: '%v'", pos, tokens[0]))
		} else {
			return node, tokens[1:], nil
		}
	}
	if num, err := strconv.ParseInt(tokens[0], 10, 64); err == nil {
		return &Node{nodeType: Num, num: num}, tokens[1:], nil
	}
	// Otherwise it must be a function name.
	return &Node{nodeType: Fun, funName: tokens[0]}, tokens[1:], nil
}

func (p *Parser) Parse(exp string) (*Node, error) {
	p.vars = make(map[string]*Node)
	lines := strings.Split(exp, "\n")
	var lastNode *Node
	for row, line := range lines {
		tokens := strings.Split(line, " ")
		node, rem, err := p.ParseExp(tokens[2:], 2)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("line %v: %v", row+1, err))
		}
		if len(rem) > 0 {
			return nil, errors.New(fmt.Sprintf("line %v: unparsed leftover %v", row+1, rem))
		}
		p.vars[tokens[0]] = node
		lastNode = node
	}
	return lastNode, nil
}

type Reducer struct {
	root      *Node
	steps     []string
	stepCount int
	showSteps bool
	lambdas   int
}

func (r *Reducer) RecordStep() {
	if r.showSteps {
		r.steps = append(r.steps, fmt.Sprint(r.root))
	}
}

func NewReducer(node *Node, showSteps bool) *Reducer {
	reducer := &Reducer{root: &Node{nodeType: Ap, fun: &Node{nodeType: Fun, funName: "root"},
		nodes: []*Node{node}}, showSteps: showSteps}
	reducer.RecordStep()
	return reducer
}

func (r *Reducer) ReduceFunction(n *Node) (*Node, error) {
	if n.fun.nodeType != Fun {
		return nil, errors.New(fmt.Sprintf("expected function node: %v", n))
	}
	if len(n.nodes) != 1 {
		return nil, errors.New(fmt.Sprintf("function node expects exactly one arg: %v", n))
	}
	// Shortcutting functions.
	switch n.fun.funName {
	case "nil":
		return &Node{nodeType: Fun, funName: "t"}, nil
	case "f": // First argument ignored.
		n.nodes[0] = &Node{nodeType: Fun, funName: "_"}
	default:
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
	case "neg", "inc", "dec":
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
			if n.fun.funName == "t" {
				// Second argument is ignored.
				node.nodes = append(node.nodes, &Node{nodeType: Fun, funName: "_"})
			} else {
				node.nodes = append(node.nodes, &Node{nodeType: Fun, funName: fmt.Sprint("X", r.lambdas)})
			}
			r.lambdas += 1
			return &Node{nodeType: Lambda, fun: node, bound: &node.nodes[1]}, nil
		}
	case "s", "c", "b":
		{
			closure := &Node{nodeType: Closure, funName: n.fun.funName}
			closure.nodes = append(closure.nodes, n.nodes[0])
			closure.nodes = append(closure.nodes, &Node{nodeType: Fun, funName: fmt.Sprint("X", r.lambdas)})
			r.lambdas += 1
			closure.nodes = append(closure.nodes, &Node{nodeType: Fun, funName: fmt.Sprint("X", r.lambdas)})
			r.lambdas += 1
			return &Node{nodeType: Lambda, fun: &Node{nodeType: Lambda, fun: closure, bound: &closure.nodes[2]},
				bound: &closure.nodes[1]}, nil
		}
	case "root", "i":
		return n.nodes[0], nil
	default:
		return nil, errors.New(fmt.Sprintf("unimplemented function: %v", n))
	}
	return nil, errors.New(fmt.Sprintf("unreachable: %v", n))
}

func (r *Reducer) Reduce(n *Node) (*Node, error) {
	//log.Printf("Reducing: %v", n)
	if n == nil {
		return nil, nil
	}
	r.stepCount += 1
	switch n.nodeType {
	case Num, Fun, Cons, Lambda:
		return n, nil
	case Ap:
		switch {
		case n.fun == nil:
			return nil, errors.New(fmt.Sprintf("fun is nil: %v", n))
		case n.fun.nodeType == Ap:
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
				if n.fun.bound == nil {
					return nil, errors.New(fmt.Sprintf("no bound variable: %v", n))
				}
				if len(n.nodes) != 1 {
					return nil, errors.New(fmt.Sprintf("lambda expects exactly one arg: %v", n))
				}
				// Only evaluate the argument if it's not discarded.
				if (*n.fun.bound).funName != "_" {
					if arg, err := r.Reduce(n.nodes[0]); err != nil {
						return nil, err
					} else {
						if n.nodes[0] != arg {
							n.nodes[0] = arg
							r.RecordStep()
						}
						*n.fun.bound = arg
						return r.Reduce(n.fun.fun)
					}
				} else {
					return r.Reduce(n.fun.fun)
				}
			}
		default:
			return r.ReduceFunction(n)
		}
	case Closure:
		{
			//for pos, node := range n.nodes {
			//	if arg, err := r.Reduce(node); err != nil {
			//		return nil, err
			//	} else {
			//		if n.nodes[pos] != arg {
			//			n.nodes[pos] = arg
			//			r.RecordStep()
			//		}
			//	}
			//}
			switch n.funName {
			case "add", "mul", "div", "eq", "lt":
				{
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
