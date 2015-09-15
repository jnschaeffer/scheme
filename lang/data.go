package lang

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

type objType int

const (
	boolT objType = iota
	numT
	vecT
	charT
	symT
	strT
	bvecT
	identT
	listT
	procT
	primitiveT

	intT
	realT
)

var typeMap = map[objType]string{
	boolT:  "bool",
	numT:   "num",
	vecT:   "vector",
	charT:  "char",
	strT:   "string",
	symT:   "symbol",
	bvecT:  "b-vector",
	identT: "identifier",
	listT:  "list",
	procT:  "procedure",
}

func isTypeGen(t objType) func(o *object) bool {
	return func(o *object) bool {
		return o.t == t
	}
}

var (
	isBool      = isTypeGen(boolT)
	isNum       = isTypeGen(numT)
	isVec       = isTypeGen(vecT)
	isChar      = isTypeGen(charT)
	isString    = isTypeGen(strT)
	isIdent     = isTypeGen(identT)
	isList      = isTypeGen(listT)
	isProc      = isTypeGen(procT)
	isSym       = isTypeGen(symT)
	isPrimitive = isTypeGen(primitiveT)
)

type number struct {
	t        objType
	intVal   int
	floatVal float64
}

type numOp func(n1, n2 number) number

func fromString(s string) number {
	if i, err := strconv.Atoi(s); err == nil {
		return number{
			intVal: i,
		}
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return number{
			floatVal: f,
		}
	}

	return number{}
}

func applyNumOp(f numOp, n1, n2 number) number {
	if n1.t > n2.t {
		n2.t = realT
		n2.floatVal = float64(n2.intVal)
	}

	if n2.t > n1.t {
		n1.t = realT
		n1.floatVal = float64(n1.intVal)
	}

	return f(n1, n2)
}

func add(n1, n2 number) number {
	switch n1.t {
	case intT:
		return number{
			t:      intT,
			intVal: n1.intVal + n2.intVal,
		}
	case realT:
		return number{
			t:        realT,
			floatVal: n1.floatVal + n2.floatVal,
		}
	}

	panic("unknown number type")
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

/* LIST */

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
	for i := l - 1; i >= 0; i-- {
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

/* OBJECT */

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
		return fmt.Sprintf("%f", o.v)
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
		return fmt.Sprintf("\"%s\"", o.v.(string))
	case procT:
		return fmt.Sprintf("#<proc>")
	default:
		return fmt.Sprintf("%s: %#v", typeMap[o.t], o.v)
	}
}

/* PROCEDURE */

type compoundProc struct {
	params []string
	body   []*object
}

/* PRIMITIVES */

type primitiveProc func(...*object) *object

func procGen(p primitiveProc) *object {
	return &object{
		t: primitiveT,
		v: p,
	}
}

func cons(args ...*object) *object {
	o1, o2 := args[0], args[1]

	return &object{
		t: listT,
		v: list{
			car: o1,
			cdr: o2,
		},
	}
}

func car(args ...*object) *object {
	o := args[0]

	return o.v.(list).car
}

func cdr(args ...*object) *object {
	o := args[0]

	return o.v.(list).cdr
}

func empty(args ...*object) *object {
	o := args[0]

	result := isList(o) && o.v == nil

	return &object{
		t: boolT,
		v: result,
	}
}

type binaryOp func(*object, *object) *object

func foldGen(f binaryOp) primitiveProc {
	return func(o ...*object) *object {
		initial := o[0]
		rest := o[1:]

		for _, r := range rest {
			initial = f(initial, r)
			if initial == nil {
				return nil
			}
		}

		return initial
	}
}

/* ANALYSIS */
func isSelfEvaluating(o *object) bool {
	types := []objType{boolT, numT, vecT, charT, strT, bvecT}

	res := false
	for _, t := range types {
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
	isQuoted     = isTaggedListGen("quote")
	isAssignment = isTaggedListGen("set!")
	isDefinition = isTaggedListGen("define")
	isLambda     = isTaggedListGen("lambda")
	isIf         = isTaggedListGen("if")
)

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

func isApplication(o *object) bool {
	return isList(o) && isProc(car(o))
}

/* EVALUATION */

func extendEnv(params []string, vals []*object, e *env) *env {
	m := make(map[string]*object, len(params))

	for i := range params {
		m[params[i]] = vals[i]
	}

	return &env{
		m:     m,
		outer: e,
	}
}

func evalQuote(o *object, e *env) *object {
	args := cdr(o)
	arg := car(args)

	ret := arg

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

func evalPrimitive(p primitiveProc, args []*object, e *env) *object {
	for i, a := range args {
		args[i] = eval(a, e)
	}

	r := p(args...)

	return r
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
		if isTrue(eval(pred, e)) {
			o = conseq
		} else {
			o = alt
		}

		goto Tailcall
	case isLambda(o):
		paramObjs := listToVec(car(cdr(o)))
		paramStrs := make([]string, len(paramObjs))
		for i, p := range paramObjs {
			if !isIdent(p) {
				log.Printf("invalid parameter value %s", p)
				return nil
			}

			paramStrs[i] = p.v.(string)
		}

		body := listToVec(cdr(cdr(o)))

		proc := compoundProc{
			params: paramStrs,
			body:   body,
		}

		return &object{
			t: procT,
			v: proc,
		}
	case isList(o):
		args := listToVec(cdr(o))
		op := eval(car(o), e)

		for i, a := range args {
			args[i] = eval(a, e)
		}

		if isPrimitive(op) {
			return op.v.(primitiveProc)(args...)
		}

		if !isProc(op) {
			log.Printf("unknown procedure")
			return nil
		}

		proc := op.v.(compoundProc)

		if len(proc.params) != len(args) {
			log.Printf("argument length mismatch: %d != %d", len(proc.params), len(args))
			return nil
		}

		e = extendEnv(proc.params, args, e)

		body := proc.body
		for i := 0; i < len(body)-1; i++ {
			eval(body[i], e)
		}

		o = body[len(body)-1]

		goto Tailcall
	}

	log.Printf("ERROR: unknown statement %s", o.String())

	return nil
}

func write(o *object) {
	fmt.Printf("%s\n", o)
}

var globalEnvMap = map[string]*object{
	"cons":   procGen(cons),
	"car":    procGen(car),
	"cdr":    procGen(cdr),
	"empty?": procGen(empty),
}

func REPL() {
	input := bufio.NewReader(os.Stdin)
	e := &env{
		m:     globalEnvMap,
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
