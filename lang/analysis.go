package lang

import (
	"fmt"
	"github.com/golang/glog"
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
	isBegin            = isTaggedListGen("begin")
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

/* ANALYSIS */

type analyzedExpr func(*evaluator, *env) (*object, error)

type compoundProc struct {
	params  []string
	body    analyzedExpr
	nArgs   int
	e       *env
	hasTail bool
}

func (p *compoundProc) bindArgs(o []*object) (*env, error) {
	var tail []*object
	var boundVals []*object

	for i, v := range o {
		switch {
		case i < len(p.params)-1:
			boundVals = append(boundVals, v)
		case i == len(p.params)-1:
			if p.hasTail {
				tail = append(tail, v)
			} else {
				boundVals = append(boundVals, v)
			}
		case i > len(p.params)-1:
			if !p.hasTail {
				return nil, fmt.Errorf("too many arguments")
			}

			tail = append(tail, v)
		}
	}

	if p.hasTail {
		boundVals = append(boundVals, vecToList(tail))
	}

	if len(boundVals) < len(p.params) {
		return nil, fmt.Errorf("not enough arguments")
	}

	extended := p.e.extend(p.params, boundVals)

	return extended, nil
}

func noOp(ev *evaluator, e *env) (*object, error) {
	return nil, nil
}

func analyze(o *object) (analyzedExpr, error) {
	switch {
	case o == nil:
		return noOp, nil
	case isSelfEvaluating(o) || isPrimitive(o):
		return analyzeSelfEvaluating(o)
	case isSymbol(o):
		return analyzeIdent(o)
	case isBegin(o):
		seq, _ := cdr(o)
		v := listToVec(seq)
		return analyzeSequence(v)
	case isQuoted(o):
		return analyzeQuoted(o)
	case isDefinition(o):
		return analyzeDefinition(o)
	case isAssignment(o):
		return analyzeAssignment(o)
	case isIf(o):
		return analyzeIf(o)
	case isLambda(o):
		return analyzeLambda(o)
	case isList(o):
		return analyzeApplication(o)
	default:
		return nil, fmt.Errorf("unknown expression %s", o)
	}
}

func analyzeSelfEvaluating(o *object) (analyzedExpr, error) {
	f := func(ev *evaluator, e *env) (*object, error) {
		return o, nil
	}

	return f, nil
}

func analyzeQuoted(o *object) (analyzedExpr, error) {
	f := func(ev *evaluator, e *env) (*object, error) {
		c, _ := cadr(o)
		return c, nil
	}

	return f, nil
}

func analyzeIdent(o *object) (analyzedExpr, error) {
	if !isSymbol(o) {
		return nil, typeMismatch(symT, o.t)
	}

	f := func(ev *evaluator, e *env) (*object, error) {
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
	first, _ := cadr(o)
	body, _ := cddr(o)

	var id *object

	// working with lambda - rewrite and evaluate
	if isList(first) {
		id, _ = car(first)
		params, _ := cdr(first)
		glog.V(3).Infof("splitting %s into %s and %s", first, id, params)

		body = cons(symbolObj("lambda"),
			cons(params, body))
		glog.V(3).Infof("rewritten as %s", body)
	} else {
		id = first
		body, _ = car(body)
	}

	if !isSymbol(id) {
		return nil, typeMismatch(symT, id.t)
	}

	bodyExpr, err := analyze(body)

	if err != nil {
		return nil, err
	}

	f := func(ev *evaluator, e *env) (*object, error) {
		o, err := evalDirect(bodyExpr, e)

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
		return nil, fmt.Errorf("length mismatch: %d", len(v))
	}

	var (
		pExpr, cExpr, aExpr analyzedExpr
		err                 error
	)

	if pExpr, err = analyze(pred); err != nil {
		return nil, err
	}

	if cExpr, err = analyze(conseq); err != nil {
		return nil, err
	}

	if aExpr, err = analyze(alt); err != nil {
		return nil, err
	}

	f := func(ev *evaluator, e *env) (*object, error) {
		p, err := evalDirect(pExpr, e)
		if err != nil {
			return nil, err
		}

		var next analyzedExpr
		if isTrue(p) {
			next = cExpr
		} else {
			next = aExpr
		}

		// Pass along to evaluator
		c := closure{
			expr: next,
			env:  e,
		}

		ev.next <- c

		return nil, nil
	}

	return f, nil
}

func analyzeLambdaParams(params *object) ([]string, bool, error) {
	switch {
	case !isList(params):
		if !isSymbol(params) {
			return nil, false, typeMismatch(symbolT, params.t)
		}
		return []string{params.v.(string)}, true, nil
	case isEmptyList(params):
		return nil, false, nil
	}

	var paramObjs []*object

	hasTail := false
	glog.V(3).Infof("params are %s", params)

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

	glog.V(3).Infof("params are now %s", paramObjs)

	paramStrs := make([]string, len(paramObjs))
	for i, p := range paramObjs {
		if !isSymbol(p) {
			return nil, false, fmt.Errorf("invalid parameter value %s", p)
		}

		paramStrs[i] = p.v.(string)
	}

	return paramStrs, hasTail, nil
}

func analyzeSequence(o []*object) (analyzedExpr, error) {
	exprs := make([]analyzedExpr, len(o))

	for i, v := range o {
		expr, err := analyze(v)
		if err != nil {
			return nil, err
		}
		exprs[i] = expr
	}

	f := func(ev *evaluator, e *env) (*object, error) {
		for i := 0; i < len(exprs)-1; i++ {
			_, err := evalDirect(exprs[i], e)
			if err != nil {
				return nil, err
			}
		}

		c := closure{
			expr: exprs[len(exprs)-1],
			env:  e,
		}

		ev.next <- c

		return nil, nil
	}

	return f, nil
}

func analyzeLambda(o *object) (analyzedExpr, error) {
	v := listToVec(o)

	if len(v) < 3 {
		return nil, fmt.Errorf("LAMBDA: not enough arguments")
	}

	v = v[1:]

	paramsObj := v[0]

	params, hasTail, err := analyzeLambdaParams(paramsObj)

	if err != nil {
		return nil, err
	}

	nArgs := len(params)
	if hasTail {
		nArgs--
	}

	bodyObjs := v[1:]
	body, err := analyzeSequence(bodyObjs)

	if err != nil {
		return nil, err
	}

	f := func(ev *evaluator, e *env) (*object, error) {
		proc := &compoundProc{
			params:  params,
			body:    body,
			nArgs:   nArgs,
			e:       e,
			hasTail: hasTail,
		}

		ret := &object{
			t: procT,
			v: proc,
		}

		return ret, nil
	}

	return f, nil
}

func wrapPrimitive(p primitiveProc, o []*object) (analyzedExpr, error) {
	var boundVals, tail []*object

	for i, v := range o {
		switch {
		case i < p.nArgs-1:
			boundVals = append(boundVals, v)
		case i == p.nArgs-1:
			if p.hasTail {
				tail = append(tail, v)
			} else {
				boundVals = append(boundVals, v)
			}
		case i > p.nArgs-1:
			if !p.hasTail {
				return nil, fmt.Errorf("too many arguments")
			}

			tail = append(tail, v)
		}
	}

	if p.hasTail {
		boundVals = append(boundVals, vecToList(tail))
	}

	if len(boundVals) < p.nArgs {
		return nil, fmt.Errorf("not enough arguments")
	}

	f := func(ev *evaluator, e *env) (*object, error) {
		return p.f(o...)
	}

	return f, nil
}

func analyzeApplication(o *object) (analyzedExpr, error) {
	v := listToVec(o)
	exprs := make([]analyzedExpr, len(v))

	for i := range v {
		e, err := analyze(v[i])
		if err != nil {
			return nil, err
		}

		exprs[i] = e
	}

	f := func(ev *evaluator, e *env) (*object, error) {
		p, err := evalDirect(exprs[0], e)
		if err != nil {
			return nil, err
		}
		if !(isProc(p) || isPrimitive(p)) {
			return nil, typeMismatch(procT, p.t)
		}

		objs := make([]*object, len(exprs)-1)
		for i := 1; i < len(exprs); i++ {
			expr := exprs[i]
			o, err := evalDirect(expr, e)
			if err != nil {
				return nil, err
			}

			objs[i-1] = o
		}

		var (
			next    analyzedExpr
			nextEnv *env
		)

		if isProc(p) {
			proc := p.v.(*compoundProc)
			next = proc.body
			ne, err := proc.bindArgs(objs)
			if err != nil {
				return nil, err
			}

			nextEnv = ne
		} else {
			n, err := wrapPrimitive(p.v.(primitiveProc), objs)
			if err != nil {
				return nil, err
			}

			next = n
			nextEnv = e

			return evalDirect(next, nextEnv)
		}

		c := closure{
			expr: next,
			env:  nextEnv,
		}

		ev.next <- c

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

func isPrimitiveApplication(o *object) bool {
	if !(isList(o) && !isEmptyList(o)) {
		return false
	}

	f, err := car(o)
	if err != nil {
		panic(err)
	}

	if !isSymbol(f) {
		return false
	}

	s := f.v.(string)

	_, ok := globalPrimitiveMap[s]

	return ok
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
	case isDefinition(expr) || isAssignment(expr):
		return cpsDefinition(expr, k, true)
	case isIf(expr):
		return cpsIf(expr, k)
	case isBegin(expr):
		return cpsBegin(expr, k)
	case isPrimitiveApplication(expr):
		return cpsPrimitiveApplication(expr, k)
	case isList(expr) && !(isEmptyList(expr) || isQuoted(expr)):
		return cpsApplication(expr, k)
	default:
		if wrapValues {
			expr = cpsWrap(expr, k)
		}

		return expr, nil
	}

}

func cpsPrimitiveApplication(o *object, k *object) (*object, error) {
	args, err := cdr(o)
	if err != nil {
		return nil, err
	}

	argVec := listToVec(args)
	argVec, kSubs, err := substituteSequence(argVec)
	if err != nil {
		return nil, err
	}

	f, err := car(o)
	if err != nil {
		return nil, err
	}

	args = vecToList(argVec)

	renamed := cons(k, cons(cons(f, args), emptyList))

	return wrapSequence(renamed, kSubs)
}

func cpsDefinition(o *object, k *object, wrapValues bool) (*object, error) {
	v := listToVec(o)

	if len(v) != 3 {
		return nil, fmt.Errorf("not enough arguments to definition/assignment")
	}

	cpsBody, err := cpsTransform(v[2], k, false)
	if err != nil {
		return nil, err
	}

	v[2] = cpsBody

	rewritten := vecToList(v)
	if wrapValues {
		rewritten = cpsWrap(rewritten, k)
	}

	return rewritten, nil
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
	lambdaObj, err := car(o)
	if err != nil {
		return nil, err
	}

	formals, err := cadr(o)
	if err != nil {
		return nil, err
	}

	body, err := cddr(o)
	if err != nil {
		return nil, err
	}
	body = cons(symbolObj("begin"), body)

	// Rewrite formals
	kObj, newFormals, err := cpsFormals(formals)
	if err != nil {
		return nil, err
	}

	cpsBody, err := cpsBegin(body, kObj)
	if err != nil {
		return nil, err
	}

	fmt.Printf("cps body: k object is %s\n", kObj)
	fmt.Printf("cps body: rewrote %s as %s\n", body, cpsBody)

	rewritten := cons(lambdaObj, cons(newFormals, cons(cpsBody, emptyList)))

	return rewritten, nil
}

func cpsIf(o *object, k *object) (*object, error) {
	fmt.Printf("CPS-IF: rewriting %s with k as %s\n", o, k)
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

type kSubstitution struct {
	expr *object
	k *object
}

func wrapSequence(o *object, ks []kSubstitution) (*object, error) {
	for i := len(ks)-1; i >= 0; i-- {
		kSub := ks[i]
		newK := cpsWrapLambda(o, kSub.k)
		rewritten, err := cpsTransform(kSub.expr, newK, false)
		if err != nil {
			return nil, err
		}

		o = rewritten
	}

	return o, nil
}

func substituteSequence(v []*object) ([]*object, []kSubstitution, error) {
	var kSubs []kSubstitution
	for i := 0; i < len(v); i++ {
		p := v[i]
		switch {
		case isLambda(p):
			lExpr, err := cpsLambda(p)
			if err != nil {
				return nil, nil, err
			}

			v[i] = lExpr
		case isList(p) && !(isIf(p) || isQuoted(p)):
			exprK := gensym("k")
			kSubs = append(kSubs, kSubstitution{expr: p, k: exprK})
			v[i] = exprK
		}
	}

	return v, kSubs, nil
}

func cpsApplication(o *object, k *object) (*object, error) {
	args, err := cdr(o)
	if err != nil {
		return nil, err
	}

	argVec := listToVec(args)
	argVec, kSubs, err := substituteSequence(argVec)
	if err != nil {
		return nil, err
	}

	f, err := car(o)
	if err != nil {
		return nil, err
	}

	args = vecToList(argVec)

	renamed := cons(f, cons(k, args))

	return wrapSequence(renamed, kSubs)
}

func cpsBegin(o *object, k *object) (*object, error) {
	stmts, err := cdr(o)
	if err != nil {
		return nil, err
	}
	begin, err := car(o)
	if err != nil {
		return nil, err
	}

	stmtVec := listToVec(stmts)
	switch len(stmtVec) {
	case 0:
		return nil, fmt.Errorf("empty begin")
	case 1:
		r, err := cpsTransform(stmtVec[0], k, true)
		if err != nil {
			return nil, err
		}

		return cons(begin, cons(r, emptyList)), nil
	}

	intermediates := stmtVec[:len(stmtVec)-1]
	intermediates, kSubs, err := substituteSequence(intermediates)
	if err != nil {
		return nil, err
	}

	last, err := cpsTransform(stmtVec[len(stmtVec)-1], k, true)
	if err != nil {
		return nil, err
	}

	stmtVec = append(intermediates, last)
	stmts = vecToList(stmtVec)
	renamed := cons(begin, stmts)

	return wrapSequence(renamed, kSubs)
}
