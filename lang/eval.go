package lang

import (
	"bufio"
	"fmt"
	"io"
	"log"
)

type closure struct {
	expr analyzedExpr
	env  *env
}

type evaluator struct {
	next chan closure
}

func newEvaluator() *evaluator {
	next := make(chan closure, 1)

	e := &evaluator{
		next: next,
	}

	return e
}

func (e *evaluator) writeAndQuitOp() *object {
	f := func(o ...*object) (*object, error) {
		v := o[0]
		fmt.Printf("%s\n", v.String())
		close(e.next)
		return nil, nil
	}

	p := procGen(f, 1, false)

	return p
}

func (e *evaluator) eval(expr analyzedExpr, env *env) {
	go func() {
		e.next <- closure{
			expr: expr,
			env: env,
		}
	}()

	for c := range e.next {
		_, err := c.expr(e, c.env)

		if err != nil {
			log.Fatalf("EVAL: %s", err.Error())
		}
	}
}

func evalDirect(expr analyzedExpr, e *env) (*object, error) {
	o, err := expr(nil, e)

	return o, err
}

func Run(r io.Reader) {
	input := bufio.NewReader(r)
	outer := &env{
		m:     globalPrimitiveMap,
		outer: nil,
	}

	globalEnv := &env{
		m:     map[string]*object{},
		outer: outer,
	}

	line, err := collectInput(input, "] ", false)
	if err == io.EOF {
		return
	}
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	p, err := parse(line)
	if err != nil {
		fmt.Printf("PARSE: %s\n", err)
		return
	}

	evaluator := newEvaluator()

	cps, err := cpsTransform(p, evaluator.writeAndQuitOp(), true)
	if err != nil {
		fmt.Printf("CPS: %s\n", err)
		return
	}

	fmt.Println(cps)
	
	expr, err := analyze(cps)

	if err != nil {
		fmt.Printf("ANALYZE: %s\n", err)
		return
	}

	evaluator.eval(expr, globalEnv)
}
