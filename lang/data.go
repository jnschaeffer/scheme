package lang

import (
	"bufio"
	"io"
	"log"
	"os"
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
	builtinT
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
	isString = isTypeGen(strT)
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

func (e *env) lookup(k string) *object {
	for e != nil {
		if o, ok := e.m[k]; ok {
			return o
		}
		e = e.outer
	}

	return nil
}

func (e *env) set(k string, o *object) *object {
	f := e

	for f != nil {
		if _, ok := f.m[k]; ok {
			f.m[k] = o
			return o
		}
		f = f.outer
	}

	e.m[k] = o

	return o
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

func vecToList(objs []*object) *object {
	l := len(objs)
	o := emptyList
	for i := l-1; i >= 0; i-- {
		o = cons(objs[i], o)
	}

	return o
}

func listToVec(o *object) []*object {
	var objs []*object

	for o.v != nil {
		if l, ok := o.v.(list); ok {
			objs = append(objs, l.car)
			o = l.cdr
		} else {
			objs = append(objs, o)
			return objs
		}
	}

	return objs
}
	
type object struct {
	t objType
	v interface{}
}

func symbolObj(s string) *object {
	return &object{
		t: identT,
		v: s,
	}
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
	case strT:
		return o.v.(string)
	case procT:
		return "#<proc>"
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

/* ANALYSIS */
func isSelfEvaluating(o *object) bool {
	types := []objType{boolT, numT, vecT, charT, strT, bvecT}

	res := false
	for _, t := range types{
		if o.t == t {
			res = true
		}
	}

	return res
}

func isTaggedList(o *object, tag string) bool {
	if isList(o) {
		l := o.v.(list)
		if isIdent(l.car) {
			return l.car.v.(string) == tag
		}
	}

	return false
}

func isTaggedListGen(tag string) func(o *object) bool {
	return func(o *object) bool {
		return isTaggedList(o, tag)
	}
}

var (
	isQuoted = isTaggedListGen("quote")
	isAssignment = isTaggedListGen("set!")
	isDefinition = isTaggedListGen("define")
	isLambda = isTaggedListGen("lambda")
	isIf = isTaggedListGen("if")
)

/* EVALUATION */

func evalQuote(o *object, e *env) *object {
	args := o.v.(list).cdr
	arg := args.v.(list).car

	ret := &object{
		t: strT,
		v: arg.String(),
	}

	return ret
}

func evalDefine(o *object, e *env) *object {
	args := o.v.(list).cdr
	argv := listToVec(args)

	id, expr := argv[0], argv[1]

	idStr := id.v.(string)
	evaled := eval(expr, e)

	e.m[idStr] = evaled

	return evaled
}

func evalAssignment(o *object, e *env) *object {
	args := o.v.(list).cdr
	argv := listToVec(args)

	id, expr := argv[0], argv[1]

	idStr := id.v.(string)
	evaled := eval(expr, e)

	e.set(idStr, evaled)

	return evaled
}

func isTrue(o *object) bool {
	return !(o.t == boolT && o.v.(bool) == false)
}

func ifExprs(o *object) (*object, *object, *object) {
	var pred, conseq, alt *object

	args := o.v.(list).cdr
	argv := listToVec(args)

	pred, conseq = argv[0], argv[1]
	if len(argv) == 3 {
		alt = argv[2]
	}

	return pred, conseq, alt
}

func eval(o *object, e *env) *object {

Tailcall:
	switch {
	case o == nil:
		return nil
	case isSelfEvaluating(o):
		return o
	case o.t == identT:
		return e.lookup(o.v.(string))
	case isQuoted(o):
		return evalQuote(o, e)
	case isDefinition(o):
		return evalDefine(o, e)
	case isAssignment(o):
		return evalAssignment(o, e)
	case isIf(o):
		pred, conseq, alt := ifExprs(o)
		if isTrue(pred) {
			o = conseq
		} else {
			o = alt
		}

		goto Tailcall
	}

	log.Printf("ERROR: unknown statement %s", o.String())

	return nil
}

func write(o *object) {
	fmt.Printf("%s\n", o)
}

func REPL() {
	input := bufio.NewReader(os.Stdin)
	e := &env{
		m: map[string]*object{},
		outer: nil,
	}
	for {
		if _, err := os.Stdout.WriteString("> "); err != nil {
			log.Fatalf("WriteString: %s", err)
		}
		line, err := input.ReadBytes('\n')
		if err == io.EOF {
			return
		}

		if err != nil {
			log.Fatalf("ReadBytes: %s", err)
		}

		write(eval(parse(string(line)), e))
	}
}
