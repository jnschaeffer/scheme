package lang

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type port struct {
	f        *os.File
	r        *bufio.Reader
	isInput  bool
	isBinary bool
	isOpen   bool
}

func eofObject(args ...*object) (*object, error) {
	eofObj := &object{
		t: eofT,
		v: nil,
	}
	return eofObj, nil
}

func openInputFile(args ...*object) (*object, error) {
	o := args[0]

	if !isString(o) {
		return nil, typeMismatch(strT, o.t)
	}

	f, err := os.Open(o.v.(string))

	if err != nil {
		return nil, fmt.Errorf("runtime error: %s", err.Error())
	}

	r := bufio.NewReader(f)
	p := &object{
		t: portT,
		v: &port{
			f:        f,
			r:        r,
			isInput:  true,
			isBinary: false,
			isOpen:   true,
		},
	}

	return p, nil
}

func closePort(args ...*object) (*object, error) {
	o := args[0]

	if !isPort(o) {
		return nil, typeMismatch(portT, o.t)
	}

	p := o.v.(*port)

	if !p.isOpen {
		return nil, nil
	}

	err := p.f.Close()
	if err != nil {
		return nil, fmt.Errorf("runtime error: %s", err.Error())
	}

	p.isOpen = false

	return nil, nil
}

func read(args ...*object) (*object, error) {
	var (
		r      *bufio.Reader
		prompt bool
	)

	switch {
	case len(args) == 0:
		r = bufio.NewReader(os.Stdin)
		prompt = true
	case len(args) == 1:
		o := args[0]
		if !isPort(o) {
			return nil, typeMismatch(portT, o.t)
		}

		r = o.v.(*port).r
		prompt = false
	default:
		return nil, fmt.Errorf("too many arguments")
	}

	s, err := collectInput(r, "> ", prompt)
	switch err {
	case nil:
		return parse(s)
	case io.EOF:
		if s == "" {
			return eofObject()
		}

		return parse(s)
	default:
		return nil, err
	}
}
