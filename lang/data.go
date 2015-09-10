package lang

import (
	"fmt"
)

type objType int

const (
	boolT objType = iota
	numT
	vecT
	charT
	strT
	bvecT
	identT
	listT
	procT
)

var typeMap = map[objType]string{
	boolT: "bool",
	numT:  "num",
	vecT:  "vector",
	charT: "char",
	strT:  "string",
	bvecT: "b-vector",
	identT:  "identifier",
	listT: "list",
	procT: "procedure",
}

func isTypeGen(t objType) func(o *object) bool {
	return func(o *object) bool {
		return o.t == t
	}
}

var (
	isBool = isTypeGen(boolT)
	isNum = isTypeGen(numT)
	isVec = isTypeGen(vecT)
	isChar = isTypeGen(charT)
	isStr = isTypeGen(strT)
	isIdent = isTypeGen(identT)
	isList = isTypeGen(listT)
	isProv = isTypeGen(procT)
)

var trueObj = &object{
	t: boolT,
	v: true,
}

type env struct {
	m     map[string]*object
	outer *env
}

func (e *env) find(k string) *object {
	m := e.m
	for m != nil {
		if o, ok := m[k]; ok {
			return o
		}
		m = e.outer.m
	}

	return nil
}

type list struct {
	car *object
	cdr *object
}

func (l list) String() string {
	str := fmt.Sprintf("(%s", l.car.String())
	x := l.cdr
	for {
		if x.t == listT {
			if x.v == nil {
				str += fmt.Sprintf(")")
				break
			} else {
				o := x.v.(list)
				str += fmt.Sprintf(" %s", o.car.String())
				x = o.cdr
			}
		} else {
			str += fmt.Sprintf(" . %s)", x.String())
			break
		}
	}

	return str
}
	
type object struct {
	t objType
	v interface{}
}

func (o *object) String() string {
	switch o.t {
	case boolT:
		if o.v.(bool) {
			return "#t"
		} else {
			return "#f"
		}
	case numT:
		return fmt.Sprintf("%s", o.v)
	case listT:
		if o.v == nil {
			return "()"
		} else {
			lst := o.v.(list)
			return lst.String()
		}
	case identT:
		return o.v.(string)
	default:
		return fmt.Sprintf("%s: %#v", typeMap[o.t], o.v)
	}
}

func cons(o1, o2 *object) *object {
	return &object{
		t: listT,
		v: list{
			car: o1,
			cdr: o2,
		},
	}
}

func eval(s string) *object {
	return parse(s)
}

func write(o *object) {
	fmt.Printf("%s\n", o)
}

func Run(s string) {
	write(eval(s))
}
