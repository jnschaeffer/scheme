package lang

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
	symbolT
	listT
	procT
	primitiveT
	errorT
	macroT
	environmentT
	portT
	eofT

	qualifiedSymT

	intT
	realT
)

var typeMap = map[objType]string{
	boolT:        "bool",
	numT:         "num",
	vecT:         "vector",
	charT:        "char",
	strT:         "string",
	symT:         "symbol",
	bvecT:        "b-vector",
	symbolT:      "identifier",
	listT:        "list",
	procT:        "procedure",
	primitiveT:   "primitive",
	macroT:       "macro",
	errorT:       "error",
	environmentT: "environment",
	portT:        "port",
	eofT:         "eof",
}

func typeMismatch(exp, obs objType) error {
	return fmt.Errorf("type mismatch: expected %s, got %s", exp, obs)
}

func isTypeGen(t objType) func(o *object) bool {
	return func(o *object) bool {
		return o != nil && o.t == t
	}
}

func isTypeProcGen(f func(o *object) bool) primitiveFunc {
	return func(o ...*object) (*object, error) {
		b := f(o[0])

		ret := &object{
			t: boolT,
			v: b,
		}

		return ret, nil
	}
}

var (
	isBool        = isTypeGen(boolT)
	isNum         = isTypeGen(numT)
	isVec         = isTypeGen(vecT)
	isChar        = isTypeGen(charT)
	isString      = isTypeGen(strT)
	isSymbol      = isTypeGen(symbolT)
	isList        = isTypeGen(listT)
	isProc        = isTypeGen(procT)
	isSym         = isTypeGen(symT)
	isPrimitive   = isTypeGen(primitiveT)
	isMacro       = isTypeGen(macroT)
	isEnvironment = isTypeGen(environmentT)
	isPort        = isTypeGen(portT)
	isEOF         = isTypeGen(eofT)
)

type env struct {
	m     map[string]*object
	outer *env
	depth int
}

func (e *env) extend(ids []string, vals []*object) *env {
	depth := e.depth + 1
	m := make(map[string]*object, len(ids))

	for i, id := range ids {
		m[id] = vals[i]
	}

	fmt.Printf("extending env to depth %d with %v\n", depth, ids)

	return &env{
		m: m,
		outer: e,
		depth: depth,
	}
}

func (e *env) lookup(k string) (*object, bool) {
	for e != nil {
		if o, ok := e.m[k]; ok {
			return o, true
		}
		e = e.outer
	}

	return nil, false
}

func (e *env) set(k string, o *object) {
	f := e

	for f != nil {
		if _, ok := f.m[k]; ok {
			f.m[k] = o
		}
		f = f.outer
	}

	e.m[k] = o
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
	for i := l - 2; i >= 0; i-- {
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
		t: symbolT,
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
	if o == nil {
		return ""
	}

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
	case symbolT:
		return o.v.(string)
	case strT:
		return fmt.Sprintf("\"%s\"", o.v.(string))
	case procT:
		return fmt.Sprintf("#<proc>")
	case macroT:
		return fmt.Sprintf("#<macro>")
	case primitiveT:
		return fmt.Sprintf("#<primitive>")
	default:
		return fmt.Sprintf("#<%s>", typeMap[o.t])
	}
}

/* PROCEDURE */

type compoundProc_ struct {
	params  []string
	body    []*object
	nArgs   int
	e       *env
	hasTail bool
}

var globalEnvMap = map[string]*object{
	"cons":            procGen(consPrimitive, 2, false),
	"car":             procGen(car, 1, false),
	"cdr":             procGen(cdr, 1, false),
	"eq?":             procGen(eq, 2, false),
	"quit":            procGen(quit, 0, false),
	"exit":            procGen(quit, 0, false),
	"+":               procGen(binaryOpGen(add, parseNum("0"), false), 0, true),
	"-":               procGen(binaryOpGen(sub, parseNum("0"), true), 0, true),
	"*":               procGen(binaryOpGen(mul, parseNum("1"), false), 0, true),
	"/":               procGen(binaryOpGen(div, parseNum("1.0"), true), 0, true),
	"read":            procGen(read, 0, true),
	"write":           procGen(write, 1, false),
	"symbol?":         procGen(isTypeProcGen(isSymbol), 1, false),
	"pair?":           procGen(isTypeProcGen(isList), 1, false),
	"string?":         procGen(isTypeProcGen(isList), 1, false),
	"symbol->string":  procGen(symbolToString, 1, false),
	"open-input-file": procGen(openInputFile, 1, false),
	"close-port":      procGen(closePort, 1, false),
	"eof-object":      procGen(eofObject, 0, false),
	"eof-object?":     procGen(isTypeProcGen(isEOF), 1, false),
	"cps":             procGen(cpsTransformOp, 2, false),
}

func init() {
	globalEnvMap["null-environment"] = procGen(nullEnv, 1, false)
}

func collectInput(r *bufio.Reader, prompt string, writePrompt bool) (string, error) {
	var stmt []byte

	leftCnt := 0
	rightCnt := 0

	for {

		if writePrompt {
			if _, err := os.Stdout.WriteString(prompt); err != nil {
				return "", err
			}
		}
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			return string(append(stmt, line...)), err
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
