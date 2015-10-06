package lang

import (
	"fmt"
)

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
		if isSymbol(l.car) {
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

type analyzedExpr func(e *env) (*object, error)

func analyze(o *object) (analyzedExpr, error) {
	return nil, nil
}

func analyzeSelfEvaluating(o *object) (analyzedExpr, error) {
	f := func(e *env) (*object, error) {
		return o, nil
	}

	return f, nil
}

func analyzeQuoted(o *object) (analyzedExpr, error) {
	f := func(e *env) (*object, error) {
		return cadr(o)
	}

	return f, nil
}

func analyzeIdent(o *object) (analyzedExpr, error) {
	if !isSymbol(o) {
		return nil, typeMismatch(symT, o.t)
	}

	f := func(e *env) (*object, error) {
		id := o.v.(string)

		v, ok := e.lookup(id)
		if !ok {
			return nil, fmt.Errorf("unbound identifier %s", id)
		}

		return v, nil
	}

	return f, nil
}

func analyzeDefinition(o *object) (analyzedExpr, error) {
	v := listToVec(o)[1:]

	if len(v) != 2 {
		return nil, fmt.Errorf("length mismatch: %d != %d", 2, len(v))
	}

	id, expr := v[1], v[2]

	if !isSymbol(id) {
		return nil, typeMismatch(symT, id.t)
	}

	exprResult, err := analyze(expr)

	if err != nil {
		return nil, err
	}

	f := func(e *env) (*object, error) {
		o, err := exprResult(e)

		if err != nil {
			return nil, err
		}

		e.set(id.v.(string), o)

		return nil, nil
	}

	return f, nil
}

// TODO: throw an error if id does not already exist
func analyzeAssignment(o *object) (analyzedExpr, error) {
	return analyzeDefinition(o)
}

func analyzeIf(o *object) (analyzedExpr, error) {
	/*
		v := listToVec(o)[1:]

		var pred, conseq, alt *object

		switch len(v) {
		case 2:
			pred = v[0]
			conseq = v[1]
		case 3:
			pred = v[0]
			conseq = v[1]
			alt = v[2]
		default:
			return nil, fmt.Errorf("length mismatch: %d", len(v))
		}

		var (
			pExpr, cExpr, aExpr analyzedExpr
			err error
		)

		if pExpr, err = analyze(pred); err != nil {
			return nil, err
		}

		if cExpr, err = analyze(conseq); err != nil {
			return nil, err
		}

		if alt != nil {
			if aExpr, err = analyze(alt); err != nil {
				return nil, err
			}
		}
	*/
	f := func(e *env) (*object, error) {
		return nil, nil
	}

	return f, nil
}

/* CPS TRANSFORM */

var gensym = func() func(string) *object {
	i := -1

	return func(x string) *object {
		if x == "" {
			x = "a"
		}

		i++

		return symbolObj(fmt.Sprintf("%s%d", x, i))
	}
}()

func cpsTransformOp(o ...*object) (*object, error) {
	return cpsTransform(o[0], o[1], false)
}

func cpsTransform(expr, k *object, wrapValues bool) (*object, error) {
	switch {
	case isLambda(expr):
		rewritten, err := cpsLambda(expr)
		if err != nil {
			return nil, err
		}

		if wrapValues {
			rewritten = cpsWrap(rewritten, k)
		}

		return rewritten, nil
	case isDefinition(expr) || isAssignment(expr) || isQuoted(expr):
		return expr, nil
	case isIf(expr):
		return cpsIf(expr, k)
	case isList(expr):
		return cpsApplication(expr, k)
	default:
		if wrapValues {
			expr = cpsWrap(expr, k)
		}

		return expr, nil
	}

}

func cpsWrap(o *object, k *object) *object {
	return cons(k, cons(o, emptyList))
}

func cpsWrapLambda(o *object, k *object) *object {
	var kList *object
	if k != nil {
		kList = cons(k, emptyList)
	} else {
		kList = emptyList
	}

	return cons(symbolObj("lambda"),
		cons(kList,
			cons(o, emptyList)))
}

func cpsFormals(o *object) (*object, *object, error) {
	if !(isSymbol(o) || isList(o)) {
		return nil, nil, fmt.Errorf("bad type for CPS formal")
	}

	k := gensym("k")
	formals := cons(k, o)

	return k, formals, nil
}

func cpsLambda(o *object) (*object, error) {
	v := listToVec(o)

	formals := v[1]

	// Rewrite formals
	kObj, newFormals, err := cpsFormals(formals)

	if err != nil {
		return nil, err
	}

	v[1] = newFormals

	last := len(v) - 1
	lastStmt := v[last]

	// Rewrite body of lambda
	newLastStmt, err := cpsTransform(lastStmt, kObj, true)

	if err != nil {
		return nil, err
	}

	v[last] = newLastStmt

	return vecToList(v), nil
}

func cpsIf(o *object, k *object) (*object, error) {
	v := listToVec(o)

	var pred, conseq, alt *object

	switch len(v) {
	case 3:
		pred = v[1]
		conseq = v[2]
		alt = boolObj(false)
	case 4:
		pred = v[1]
		conseq = v[2]
		alt = v[3]
	default:
		return nil, fmt.Errorf("bad if statement")
	}

	p := pred
	wrapIfStmt := false
	var predK, predExpr *object

	switch {
	case isLambda(p):
		lExpr, err := cpsLambda(p)
		if err != nil {
			return nil, err
		}

		p = lExpr
	case isList(p) && !(isDefinition(p) || isQuoted(p) || isIf(p) || isAssignment(p)):
		predK = gensym("k")
		predExpr = p
		v[1] = predK
		wrapIfStmt = true
	}

	// Wrap branches in thunks
	transformedConseq, err := cpsTransform(conseq, k, true)
	if err != nil {
		return nil, err
	}

	transformedAlt, err := cpsTransform(alt, k, true)
	if err != nil {
		return nil, err
	}

	v[2] = transformedConseq
	v[3] = transformedAlt

	result := vecToList(v)
	if wrapIfStmt {
		result = cpsWrapLambda(result, predK)
		wrapped, err := cpsTransform(predExpr, result, false)
		if err != nil {
			return nil, err
		}

		result = wrapped
	}

	return result, nil
}

func cpsApplication(o *object, k *object) (*object, error) {
	// Rename all arguments to application first
	v := listToVec(o)

	ks := make(map[int][2]*object) // map of [k-symbol, sub-expr] pairs

	for i := 1; i < len(v); i++ {
		p := v[i]
		switch {
		case isLambda(p):
			lExpr, err := cpsLambda(p)
			if err != nil {
				return nil, err
			}

			v[i] = lExpr
		case isList(p) && !(isDefinition(p) || isQuoted(p) || isIf(p) || isAssignment(p)):
			exprK := gensym("k")
			ks[i] = [2]*object{p, exprK}
			v[i] = exprK
		default:
			v[i] = p
		}
	}

	renamed := make([]*object, len(v)+1)
	renamed[0] = v[0]
	renamed[1] = k
	for i := 2; i < len(renamed); i++ {
		renamed[i] = v[i-1]
	}

	result := vecToList(renamed)

	for i := 1; i < len(v); i++ {
		pair, ok := ks[i]
		if ok {
			subexp, subK := pair[0], pair[1]
			fmt.Printf("wrapping %s around %s\n", subexp, result)
			result = cpsWrapLambda(result, subK)
			rewrittenSubexp, err := cpsTransform(subexp, result, false)
			if err != nil {
				return nil, err
			}
			result = rewrittenSubexp
		}
	}

	return result, nil
}
