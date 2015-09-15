package lang

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

type objType int

func (t objType) String() string {
	return typeMap[t]
}

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
	errorT

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
	errorT: "error",
}

func typeMismatch(exp, obs objType) error {
	return fmt.Errorf("type mismatch: expected %s, got %s", exp, obs)
}

func isTypeGen(t objType) func(o *object) bool {
	return func(o *object) bool {
		return o != nil && o.t == t
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

func (n number) String() string {
	switch n.t {
	case intT:
		return fmt.Sprintf("%d", n.intVal)
	case realT:
		return fmt.Sprintf("%f", n.floatVal)
	default:
		return "?"
	}
}

type numOp func(n1, n2 number) number

func parseNum(s string) number {
	if i, err := strconv.Atoi(s); err == nil {
		return number{
			t:      intT,
			intVal: i,
		}
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return number{
			t:        realT,
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

func boolObj(b bool) *object {
	return &object{
		t: boolT,
		v: b,
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
		return o.v.(number).String()
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
	nArgs  int
}

/* PRIMITIVES */

type primitiveFunc func(...*object) (*object, error)

type primitiveProc struct {
	f     primitiveFunc
	nArgs int
}

func procGen(f primitiveFunc, nArgs int) *object {
	p := primitiveProc{
		f:     f,
		nArgs: nArgs,
	}

	return &object{
		t: primitiveT,
		v: p,
	}
}

func cons(o1, o2 *object) *object {

	r := &object{
		t: listT,
		v: list{
			car: o1,
			cdr: o2,
		},
	}

	return r
}

func consPrimitive(args ...*object) (*object, error) {
	return cons(args[0], args[1]), nil
}

func car(args ...*object) (*object, error) {
	o := args[0]
	if !isList(o) {
		return nil, typeMismatch(listT, o.t)
	}

	if o.v == nil {
		return nil, fmt.Errorf("empty list")
	}

	return o.v.(list).car, nil
}

func cdr(args ...*object) (*object, error) {
	o := args[0]
	if !isList(o) {
		return nil, typeMismatch(listT, o.t)
	}

	if o.v == nil {
		return nil, fmt.Errorf("reached empty list")
	}

	return o.v.(list).cdr, nil
}

func cdxr(depth int) primitiveFunc {
	return func(args ...*object) (*object, error) {
		o := args[0]
		var err error
		for i := 0; i < depth; i++ {
			o, err = cdr(o)
			if err != nil {
				return nil, err
			}
		}

		return o, nil
	}
}

var (
	cddr  = cdxr(2)
	cdddr = cdxr(3)
)

func cadxr(depth int) primitiveFunc {
	f := cdxr(depth)
	return func(args ...*object) (*object, error) {
		o := args[0]
		o, err := f(o)
		if err != nil {
			return nil, err
		}

		return car(o)
	}
}

var (
	caddr  = cadxr(2)
	cadddr = cadxr(3)
)

func cadr(args ...*object) (*object, error) {
	o := args[0]
	o, err := cdr(o)
	if err != nil {
		return nil, err
	}

	return car(o)
}

func eq(args ...*object) (*object, error) {
	o1, o2 := args[0], args[1]

	if o1.t != o2.t || o1.v != o2.v {
		return boolObj(false), nil
	}

	return boolObj(true), nil
}

func quit(args ...*object) (*object, error) {
	os.Exit(0)

	return nil, nil
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
	p, _ := car(o)
	return isList(o) && isProc(p)
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

func evalQuote(o *object, e *env) (*object, error) {
	args, _ := cdr(o)

	ret, _ := car(args)

	return ret, nil
}

func evalDefine(o *object, e *env) (*object, error) {
	args, _ := cdr(o)
	argv := listToVec(args)

	id, expr := argv[0], argv[1]

	idStr := id.v.(string)
	evaled, err := eval(expr, e)
	if err != nil {
		return nil, err
	}

	e.m[idStr] = evaled

	return evaled, nil
}

func evalAssignment(o *object, e *env) (*object, error) {
	args := o.v.(list).cdr
	argv := listToVec(args)

	id, expr := argv[0], argv[1]

	idStr := id.v.(string)
	evaled, err := eval(expr, e)
	if err != nil {
		return nil, err
	}

	e.set(idStr, evaled)

	return evaled, nil
}

func evalPrimitive(p primitiveProc, args []*object, e *env) (*object, error) {
	if p.nArgs != len(args) {
		err := fmt.Errorf("argument length mismatch: %d != %d", p.nArgs, len(args))
		return nil, err
	}

	r, err := p.f(args...)

	return r, err
}

func eval(o *object, e *env) (*object, error) {

Tailcall:
	switch {
	case o == nil:
		return nil, nil
	case isSelfEvaluating(o):
		return o, nil
	case o.t == identT:
		ret := e.lookup(o.v.(string))
		if ret == nil {
			return nil, fmt.Errorf("unknown identifier %s", o)
		}
		return ret, nil
	case isQuoted(o):
		return evalQuote(o, e)
	case isDefinition(o):
		return evalDefine(o, e)
	case isAssignment(o):
		return evalAssignment(o, e)
	case isIf(o):
		pred, conseq, alt := ifExprs(o)
		evaledPred, err := eval(pred, e)
		if err != nil {
			return nil, err
		}
		if isTrue(evaledPred) {
			o = conseq
		} else {
			o = alt
		}

		goto Tailcall
	case isLambda(o):
		params, err := cadr(o)
		if err != nil {
			return nil, err
		}
		paramObjs := listToVec(params)
		paramStrs := make([]string, len(paramObjs))
		for i, p := range paramObjs {
			if !isIdent(p) {
				return nil, fmt.Errorf("invalid parameter value %s", p)
			}

			paramStrs[i] = p.v.(string)
		}

		bodyList, err := cddr(o)
		if err != nil {
			return nil, err
		}

		body := listToVec(bodyList)

		proc := compoundProc{
			params: paramStrs,
			body:   body,
			nArgs:  len(paramStrs),
		}

		ret := &object{
			t: procT,
			v: proc,
		}

		return ret, nil

	case isList(o):
		args, _ := cdr(o)
		argv := listToVec(args)
		op, _ := car(o)
		op, err := eval(op, e)
		if err != nil {
			return nil, err
		}

		for i, a := range argv {
			argv[i], err = eval(a, e)
			if err != nil {
				return nil, err
			}
		}

		if isPrimitive(op) {
			p := op.v.(primitiveProc)
			r, err := evalPrimitive(p, argv, e)
			if err != nil {
				return nil, err
			}

			return r, nil
		}

		if !isProc(op) {
			return nil, typeMismatch(procT, op.t)
		}

		proc := op.v.(compoundProc)

		if proc.nArgs != len(argv) {
			err = fmt.Errorf("argument length mismatch: %d != %d", len(proc.params), len(argv))
			return nil, err
		}

		e = extendEnv(proc.params, argv, e)

		body := proc.body
		for i := 0; i < len(body)-1; i++ {
			_, err = eval(body[i], e)
			if err != nil {
				return nil, err
			}
		}

		o = body[len(body)-1]

		goto Tailcall
	}

	return nil, fmt.Errorf("unknown statement %s", o)
}

var globalEnvMap = map[string]*object{
	"cons":   procGen(consPrimitive, 2),
	"car":    procGen(car, 1),
	"cdr":    procGen(cdr, 1),
	"cddr":   procGen(cddr, 1),
	"cdddr":  procGen(cdddr, 1),
	"cadr":   procGen(cadr, 1),
	"caddr":  procGen(caddr, 1),
	"cadddr": procGen(cadddr, 1),
	"eq?":    procGen(eq, 2),
	"quit":   procGen(quit, 0),
}

func collectInput(r *bufio.Reader) (string, error) {
	var stmt []byte

	leftCnt := 0
	rightCnt := 0

	for {
		prompt := "> "
		if leftCnt > 0 {
			prompt = "  " + strings.Repeat("  ", leftCnt - rightCnt)
		}

		if _, err := os.Stdout.WriteString(prompt); err != nil {
			log.Fatal(err)
		}
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			return "", err
		}

		for _, b := range line {
			if b == '(' {
				leftCnt++
			}

			if b == ')' {
				rightCnt++
			}
		}

		if leftCnt < rightCnt {
			return "", fmt.Errorf("mismatched parentheses")
		}

		stmt = append(stmt, line...)

		if leftCnt == rightCnt {
			return string(stmt), nil
		}
	}
}

func REPL() {
	input := bufio.NewReader(os.Stdin)
	e := &env{
		m:     globalEnvMap,
		outer: nil,
	}
	for {
		line, err := collectInput(input)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		o, err := eval(parse(line), e)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
		} else {
			fmt.Printf("%s\n", o)
		}
	}
}
