package lang

import (
	"bufio"
	"bytes"
	_ "fmt"
	"io"
	"log"
	"os"
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
		//fmt.Printf("%s\n", v.String())
		close(e.next)
		return v, nil
	}

	p := procGen(f, 1, false)

	return p
}

func (e *evaluator) eval(expr analyzedExpr, env *env) (*object, error) {
	go func() {
		e.next <- closure{
			expr: expr,
			env:  env,
		}
	}()

	var (
		o   *object
		err error
	)

	for c := range e.next {
		o, err = c.expr(e, c.env)

		if err != nil {
			return nil, err
		}
	}

	return o, nil
}

func evalDirect(expr analyzedExpr, e *env) (*object, error) {
	o, err := expr(nil, e)

	return o, err
}

type Runtime struct {
	e *env
}

func NewRuntime() *Runtime {
	outer := &env{
		m: globalPrimitiveMap,
		outer: nil,
	}

	globalEnv := &env{
		m: map[string]*object{},
		outer: outer,
	}

	return &Runtime{
		e: globalEnv,
	}
}

func (rt *Runtime) run(r io.Reader, isREPL bool) (*object, error) {
	input := bufio.NewReader(r)

	line, err := collectInput(input, "] ", isREPL)
	if err != nil && err != io.EOF {
		log.Printf("ERROR: %s\n", err)
		return nil, err
	}

	p, err := parse(line)
	if err != nil {
		log.Printf("PARSE: %s\n", err)
		return nil, err
	}

	evaluator := newEvaluator()

	cps, err := cpsTransform(p, evaluator.writeAndQuitOp(), true)
	if err != nil {
		log.Printf("CPS: %s\n", err)
		return nil, err
	}

	log.Print(cps)
	expr, err := analyze(cps)

	if err != nil {
		log.Printf("ANALYZE: %s\n", err)
		return nil, err
	}

	return evaluator.eval(expr, rt.e)
}

func (rt *Runtime) Eval(r io.Reader) (*object, error) {
	return rt.run(r, false)
}

func (rt *Runtime) REPL() error {
	for {
		_, err := rt.run(os.Stdin, true)
		if err != nil {
			log.Print(err)
		}
	}

	return nil
}

func (rt *Runtime) EvalString(s string) (*object, error) {
	b := bytes.NewBufferString(s)

	return rt.Eval(b)
}
