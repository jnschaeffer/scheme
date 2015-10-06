package lang

import (
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
	next := make(chan closure)

	e := &evaluator{
		next: next,
	}

	go e.eval()

	return e
}

func (e *evaluator) eval() {
	for c := range e.next {
		_, err := eval_(c.expr, c.env)

		if err != nil {
			log.Fatalf("EVAL: %s", err.Error())
		}
	}
}

func eval_(expr analyzedExpr, e *env) (*object, error) {
	return expr(e)
}
