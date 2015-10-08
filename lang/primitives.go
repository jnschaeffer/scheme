package lang

import (
	"fmt"
	"os"
)

/* PRIMITIVES */

type primitiveFunc func(...*object) (*object, error)

type primitiveProc struct {
	f       primitiveFunc
	nArgs   int
	hasTail bool
}

func procGen(f primitiveFunc, nArgs int, hasTail bool) *object {
	p := primitiveProc{
		f:       f,
		nArgs:   nArgs,
		hasTail: hasTail,
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

func write(args ...*object) (*object, error) {
	fmt.Printf("%s\n", args[0])
	return nil, nil
}

func symbolToString(o ...*object) (*object, error) {
	s := o[0]
	if !isSymbol(s) {
		return nil, typeMismatch(symbolT, s.t)
	}

	ret := &object{
		t: strT,
		v: s.v,
	}

	return ret, nil
}
