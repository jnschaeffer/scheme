package lang

import (
	"fmt"
	"strconv"
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
		return fmt.Sprintf("%g", n.floatVal)
	default:
		return "?"
	}
}

type unaryOp func(n number) number
type binaryOp func(n1, n2 number) number

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

func applyBinaryOp(f binaryOp, n1, n2 number) number {
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

func binaryOpGen(f binaryOp, initial number, isSubDiv bool) primitiveFunc {
	return func(o ...*object) (*object, error) {
		var result number

		switch {
		case len(o) == 0:
			result = initial
		case len(o) == 1:
			n := o[0]
			if !isNum(n) {
				return nil, typeMismatch(numT, n.t)
			}
			if isSubDiv {
				result = applyBinaryOp(f, initial, n.v.(number))
			} else {
				result = n.v.(number)
			}
		default:
			n := o[0]
			if !isNum(n) {
				return nil, typeMismatch(numT, n.t)
			}
			result = n.v.(number)
			o = o[1:]

			for _, n := range o {
				if !isNum(n) {
					return nil, typeMismatch(numT, n.t)
				}

				result = applyBinaryOp(f, result, n.v.(number))
			}
		}

		ret := &object{
			t: numT,
			v: result,
		}

		return ret, nil
	}
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

func sub(n1, n2 number) number {
	switch n1.t {
	case intT:
		return number{
			t:      intT,
			intVal: n1.intVal - n2.intVal,
		}
	case realT:
		return number{
			t:        realT,
			floatVal: n1.floatVal - n2.floatVal,
		}
	}

	panic("unknown number type")
}

func mul(n1, n2 number) number {
	switch n1.t {
	case intT:
		return number{
			t:      intT,
			intVal: n1.intVal * n2.intVal,
		}
	case realT:
		return number{
			t:        realT,
			floatVal: n1.floatVal * n2.floatVal,
		}
	}

	panic("unknown number type")
}

func div(n1, n2 number) number {
	switch n1.t {
	case intT:
		return number{
			t:      intT,
			intVal: n1.intVal / n2.intVal,
		}
	case realT:
		return number{
			t:        realT,
			floatVal: n1.floatVal / n2.floatVal,
		}
	}

	panic("unknown number type")
}

func floor(n number) number {
	switch n.t {
	case intT:
		return n
	case realT:
		return number{
			t: intT,
			intVal: int(n.floatVal),
		}
	}

	panic("unknown number type")
}

func ceiling(n number) number {
	switch n.t {
	case intT:
		return n
	case realT:
		return number{
			t: intT,
			intVal: int(n.floatVal + 0.5),
		}
	}

	panic("unknown number type")
}
