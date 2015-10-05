package lang

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
	v := listToVec(o[1:])

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
	v := listToVec(o[1:])

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

	f := func(e *env) (*object, error) {
		return nil, nil
	}

	return f, nil
}
