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
	macroT

	intT
	realT
)

var typeMap = map[objType]string{
	boolT:      "bool",
	numT:       "num",
	vecT:       "vector",
	charT:      "char",
	strT:       "string",
	symT:       "symbol",
	bvecT:      "b-vector",
	identT:     "identifier",
	listT:      "list",
	procT:      "procedure",
	primitiveT: "primitive",
	macroT:     "macro",
	errorT:     "error",
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
	isMacro     = isTypeGen(macroT)
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

func (l *list) String() string {

	str := fmt.Sprintf("(%s", l.car.String())
	x := l.cdr
	for {
		if x.t == listT {
			if x.v == nil {
				str += fmt.Sprintf(")")
				break
			} else {
				o := x.v.(*list)
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

func vecToImproperList(objs []*object) *object {
	l := len(objs)
	if l == 0 {
		return emptyList
	}

	o := objs[l-1]
	for i := l-2; i >= 0; i-- {
		o = cons(objs[i], o)
	}

	return o
}

func listToVec(o *object) []*object {
	var objs []*object

	for o.v != nil {
		if isList(o) {
			l := o.v.(*list)
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
			lst := o.v.(*list)
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
	e      *env
	hasTail bool
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
		v: &list{
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

	return o.v.(*list).car, nil
}

func cdr(args ...*object) (*object, error) {
	o := args[0]
	if !isList(o) {
		return nil, typeMismatch(listT, o.t)
	}

	if o.v == nil {
		return nil, fmt.Errorf("reached empty list")
	}

	return o.v.(*list).cdr, nil
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
	if isList(o) && !isEmptyList(o) {
		l := o.v.(*list)
		if isIdent(l.car) {
			return l.car.v.(string) == tag
		}
	}

	return false
}

func isEmptyList(o *object) bool {
	return isList(o) && o.v == nil
}

func isTaggedListGen(tag string) func(o *object) bool {
	return func(o *object) bool {
		return isTaggedList(o, tag)
	}
}

var (
	isQuasiquoted      = isTaggedListGen("quasiquote")
	isQuoted           = isTaggedListGen("quote")
	isAssignment       = isTaggedListGen("set!")
	isDefinition       = isTaggedListGen("define")
	isLambda           = isTaggedListGen("lambda")
	isIf               = isTaggedListGen("if")
	isUnquoted         = isTaggedListGen("unquote")
	isSplicingUnquoted = isTaggedListGen("unquote-splicing")
	isSyntaxDefinition = isTaggedListGen("define-syntax")
)

func isTrue(o *object) bool {
	return !(o.t == boolT && o.v.(bool) == false)
}

func ifExprs(o *object) (*object, *object, *object) {
	var pred, conseq, alt *object

	args := o.v.(*list).cdr
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

/* EXPANSION */

func applyMacro(m *object, argv []*object, e *env) (*object, error) {
	log.Printf("applying %s", m.v)
	p := m.v.(compoundProc)
	expr := p.body[0]

	f, err := extendEnv(p.params, argv, p.hasTail, e)
	if err != nil {
		return nil, err
	}

	return eval(expr, f)
}

func expand(o *object, e *env) (*object, error) {
	if !(isList(o) && !isEmptyList(o)) {
		return o, nil
	}

	// don't expand quotes before evaluation
	if (isQuoted(o) || isQuasiquoted(o)) {
		return o, nil
	}

	p := o
	head, _ := car(o)
	tail, _ := cdr(o)
	done := false

	if isIdent(head) {
		m := e.lookup(head.v.(string))

		if m != nil && isMacro(m) {
			log.Printf("found macro %s", head.String())
			argv := listToVec(tail)
			log.Printf("expanding %s", o.String())

			r, err := applyMacro(m, argv, e)

			if err != nil {
				log.Printf("MACRO ERROR")
				return nil, err
			}

			log.Printf("expanded to %s", r.String())

			return expand(r, e)
		}
	}

	for !done {
		if isList(head) {
			log.Printf("expanding list")
			r, err := expand(head, e)
			if err != nil {
				return nil, err
			}

			p.v.(*list).car = r
		}

		switch {
		case !isList(tail):
			head = tail
			tail = emptyList
		case isEmptyList(tail):
			done = true
		default:
			p = tail
			head, _ = car(tail)
			tail, _ = cdr(tail)
		}
	}

	return o, nil
}

/* EVALUATION */

func extendEnv(params []string, vals []*object, hasTail bool, e *env) (*env, error) {

	var tail []*object
	var boundVals []*object
	for i, v := range vals {
		switch {
		case i < len(params) - 1:
			boundVals = append(boundVals, v)
		case i == len(params) - 1:
			if hasTail {
				tail = append(tail, v)
			} else {
				boundVals = append(boundVals, v)
			}
		case i > len(params) - 1:
			if !hasTail {
				return nil, fmt.Errorf("too many arguments")
			}

			tail = append(tail, v)
		}
	}

	if hasTail {
		boundVals = append(boundVals, vecToList(tail))
	}

	if len(boundVals) < len(params) {
		return nil, fmt.Errorf("not enough arguments")
	}
				
	m := make(map[string]*object, len(params))

	for i := range params {
		m[params[i]] = boundVals[i]
	}

	ret := &env{
		m:     m,
		outer: e,
	}

	return ret, nil
}

func evalQuote(o *object, e *env) (*object, error) {
	args, _ := cdr(o)

	ret, _ := car(args)

	return ret, nil
}

func evalDefine(o *object, e *env) (*object, error) {
	first, _ := cadr(o)
	body, _ := cddr(o)

	var id *object

	// working with lambda - rewrite and evaluate
	if isList(first) {
		id, _ = car(first)
		params, _ := cdr(first)
		log.Printf("splitting %s into %s and %s", first, id, params)

		body = cons(symbolObj("lambda"),
			cons(params, body))
		log.Printf("rewritten as %s", body)
	} else {
		id = first
		body, _ = car(body)
	}

	idStr := id.v.(string)
	evaled, err := eval(body, e)
	if err != nil {
		return nil, err
	}

	e.m[idStr] = evaled

	return evaled, nil
}

func evalAssignment(o *object, e *env) (*object, error) {
	args := o.v.(*list).cdr
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

func evalSyntaxDefinition(o *object, e *env) (*object, error) {
	args, _ := cdr(o)
	argv := listToVec(args)
	if len(argv) != 2 {
		return nil, fmt.Errorf("argument length mismatch: %d != %d", 2, len(argv))
	}

	id, p := argv[0], argv[1]

	p, err := eval(p, e)
	if err != nil {
		return nil, err
	}

	if !isIdent(id) {
		return nil, typeMismatch(identT, id.t)
	}

	if !isProc(p) {
		return nil, typeMismatch(procT, p.t)
	}

	m := &object{
		t: macroT,
		v: p.v,
	}

	e.m[id.v.(string)] = m

	return m, nil
}

func evalUnquote(o *object, e *env, level int) (*object, error) {
	switch {
	case level < 0:
		return nil, fmt.Errorf("illegal unquote")
	case level == 0:
		d, _ := cadr(o)
		r, err := eval(d, e)
		if err != nil {
			return nil, err
		}

		log.Printf("unquote evaluated to %s", r)
		return r, nil
	default:
		log.Printf("evaluating unquoted object %s", o)
		d, _ := cadr(o)
		d, err := evalQuasiquote(d, e, level)
		if err != nil {
			return nil, err
		}

		head, _ := car(o)
		result := cons(head, cons(d, emptyList))

		return result, nil
	}
}

func evalSplicingUnquote(o *object, e *env, level int) (*object, error) {
	body, _ := cadr(o)
	evaled, err := evalUnquote(o, e, level)
	if err != nil {
		return nil, err
	}

	log.Printf("splice result is %s", evaled)

	if !isList(evaled) {
		return nil, typeMismatch(listT, body.t)
	}

	return evaled, nil
}

func evalQuasiquote(o *object, e *env, level int) (*object, error) {
	log.Printf("evaluating quasiquote object %s at %d", o, level)

	q := o

	if level == 0 {
		q, _ = cadr(o)
		return evalQuasiquote(q, e, 1)
	}

	switch {
	case isEmptyList(q):
		log.Printf("empty list. returning")
		return o, nil
	case isSelfEvaluating(q) || isIdent(q):
		log.Printf("self-evaluating. returning")
		return o, nil
	case isQuasiquoted(q):
		log.Printf("increasing to level %d", level+1)
		inner, _ := cadr(q)
		p, err := evalQuasiquote(inner, e, level+1)
		if err != nil {
			return nil, err
		}
		result := cons(symbolObj("quasiquote"), cons(p, emptyList))

		log.Printf("returning %s", q.String())
		return result, nil

	case isUnquoted(q):
		log.Printf("decreasing to level %d", level-1)
		p, err := evalUnquote(q, e, level-1)
		if err != nil {
			return nil, err
		}
		return p, nil

	case isList(q):
		log.Printf("evaluating list")
		vec := listToVec(q)
		var result []*object
		for _, v := range vec {
			// special case for unquote-splicing
			if isSplicingUnquoted(v) {
				log.Printf("evaluating splicing unquote %s at level %d", v, level-1)
				p, err := evalSplicingUnquote(v, e, level-1)
				if err != nil {
					return nil, err
				}
				log.Printf("splice result: %s", p)
				r := listToVec(p)

				result = append(result, r...)
			} else {
				p, err := evalQuasiquote(v, e, level)
				if err != nil {
					return nil, err
				}

				result = append(result, p)
			}
		}

		return vecToList(result), nil
	}

	return nil, fmt.Errorf("how did we get here?")
}

func evalVector(objs []*object, e *env) ([]*object, error) {
	ret := make([]*object, len(objs))
	for i, a := range objs {
		r, err := eval(a, e)
		if err != nil {
			return nil, err
		}
		ret[i] = r
	}

	return ret, nil
}

func evalLambdaParams(params *object) ([]string, bool, error) {
	switch {
	case !isList(params):
		if !isIdent(params) {
			return nil, false, typeMismatch(identT, params.t)
		}
		return []string{params.v.(string)}, true, nil
	case isEmptyList(params):
		return nil, false, nil
	}

	var paramObjs []*object

	hasTail := false
	log.Printf("params are %s", params)

	done := false
	head, _ := car(params)
	tail, _ := cdr(params)
	for !done {
		paramObjs = append(paramObjs, head)
		switch {
		case !isList(tail):
			head = tail
			tail = emptyList
			hasTail = true
		case isEmptyList(tail):
			done = true
		default:
			head, _ = car(tail)
			tail, _ = cdr(tail)
		}
	}

	log.Printf("params are now %s", paramObjs)

	paramStrs := make([]string, len(paramObjs))
	for i, p := range paramObjs {
		if !isIdent(p) {
			return nil, false, fmt.Errorf("invalid parameter value %s", p)
		}

		paramStrs[i] = p.v.(string)
	}

	return paramStrs, hasTail, nil
}

func evalLambda(o *object, e *env) (*object, error) {
	params, err := cadr(o)
	if err != nil {
		return nil, err
	}

	paramStrs, hasTail, err := evalLambdaParams(params)
	if err != nil {
		return nil, err
	}

	bodyList, err := cddr(o)
	if err != nil {
		return nil, err
	}
	body := listToVec(bodyList)

	nArgs := len(paramStrs)
	if hasTail {
		nArgs--
	}

	proc := compoundProc{
		params: paramStrs,
		body:   body,
		nArgs:  nArgs,
		e:      e,
		hasTail: hasTail,
	}

	ret := &object{
		t: procT,
		v: proc,
	}

	return ret, nil
}

func eval(o *object, e *env) (*object, error) {

	//log.Printf("evaluating %s", o.String())
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
	case isQuasiquoted(o):
		return evalQuasiquote(o, e, 0)
	case isQuoted(o):
		return evalQuote(o, e)
	case isDefinition(o):
		return evalDefine(o, e)
	case isSyntaxDefinition(o):
		return evalSyntaxDefinition(o, e)
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
		return evalLambda(o, e)

	case isList(o):
		args, _ := cdr(o)
		argv := listToVec(args)
		op, _ := car(o)
		op, err := eval(op, e)
		if err != nil {
			return nil, err
		}

		if isPrimitive(op) {
			argv, err = evalVector(argv, e)
			if err != nil {
				return nil, err
			}

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

		argv, err = evalVector(argv, e)
		if err != nil {
			return nil, err
		}

		proc := op.v.(compoundProc)

		e, err = extendEnv(proc.params, argv, proc.hasTail, proc.e)
		if err != nil {
			return nil, err
		}

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
			prompt = "  " + strings.Repeat("  ", leftCnt-rightCnt)
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

		p, err := parse(line)
		if err != nil {
			fmt.Printf("PARSE: %s\n", err)
			continue
		}

		p, err = expand(p, e)
		if err != nil {
			fmt.Printf("EXPAND: %s\n", err)
			continue
		}

		o, err := eval(p, e)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
		} else {
			fmt.Printf("%s\n", o)
		}
	}
}
