//go:generate -command yacc go tool yacc
//go:generate yacc -o scm.go -p "expr" scm.y

package lang

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/golang/glog"
)

const (
	eof rune = -1
)

var idMap = map[string]int{
	"lambda": LAMBDA,
	"if":     IF,
	"define": DEFINE,
}

type item struct {
	t     int
	input string
}

type stateFn func(l *lexer) stateFn

type lexer struct {
	name  string
	input string
	start int
	pos   int
	width int
	items chan item
	idMap map[string]int
}

func isAlphaNumeric(r rune) bool {
	return unicode.IsDigit(r) || unicode.IsLetter(r)
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t'
}

func lexStart(l *lexer) stateFn {
	switch r := l.next(); {
	case isWhitespace(r):
		return lexWhitespace
	case r == '(':
		l.emit(LPAREN)
		return lexStart
	case r == ')':
		l.emit(RPAREN)
		return lexStart
	case r == '\'':
		l.emit(QUOTE)
		return lexStart
	case r == '`':
		l.emit(BACKTICK)
		return lexStart
	case r == ',':
		if l.peek() == '@' {
			l.next()
			l.emit(COMMAAT)
		} else {
			l.emit(COMMA)
		}
		return lexStart
	case r == '.' && !unicode.IsDigit(l.peek()):
		l.emit(DOT)
		return lexStart
	case (r == '+' || r == '-') && !unicode.IsDigit(l.peek()):
		l.emit(IDENT)
		return lexStart
	case r == '.' || r == '+' || r == '-' || ('0' <= r && r <= '9'):
		l.backup()
		return lexNumber
	case r == '"':
		l.ignore()
		return lexString
	case r == '#':
		switch r = l.next(); {
		case r == 't' || r == 'f':
			l.backup()
			return lexBoolean
		case r == '\\':
			return lexCharacter
		case r == '(':
			l.emit(LVEC)
			return lexStart
		default:
			return l.errorf("bad # sequence")
		}
	case r == eof || r == '\n':
		return nil
	default:
		l.backup()
		return lexIdentifier
	}

	return lexStart
}

func lexWhitespace(l *lexer) stateFn {
	l.acceptRun(" \t\n")

	l.emit(WSPACE)

	return lexStart
}

func lexNumber(l *lexer) stateFn {
	// Optional leading sign.
	l.accept("+-")

	digits := "0123456789"

	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}

	if isAlphaNumeric(l.peek()) {
		l.next()
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	}

	l.emit(NUM)

	return lexStart
}

func lexString(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated quoted string")
		case '"':
			l.backup()
			break Loop
		}
	}

	l.emit(STRING)

	l.next()
	l.ignore()

	return lexStart
}

func lexBoolean(l *lexer) stateFn {
	switch r := l.next(); {
	case r == 't' || r == 'f':
		l.emit(BOOLEAN)
	default:
		return l.errorf("bad boolean value")
	}

	if isAlphaNumeric(l.peek()) {
		return l.errorf("bad boolean value")
	}

	return lexStart
}

func lexSymbol(l *lexer) stateFn {
	return lexIdentifier
}

func lexIdentifier(l *lexer) stateFn {
	letter := "abcdefghijklmnopqrstuvwxyz"
	letter += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digit := "0123456789"
	specialInit := "!$%&*/:<=>?^_~"
	explicitSign := "+-"
	specialSubseq := explicitSign + ".@"

	initial := letter + specialInit
	subseq := initial + digit + specialSubseq

	if !l.accept(initial) {
		return l.errorf("bad identifier")
	}

	l.acceptRun(subseq)

	idText := l.input[l.start:l.pos]

	glog.V(3).Infof("checking for id text %s", idText)

	if id, ok := l.idMap[idText]; ok {
		glog.V(3).Infof("emitting special ID %s", idText)
		l.emit(id)
	} else {
		l.emit(IDENT)
	}

	return lexStart
}

func lexCharacter(l *lexer) stateFn {
	glog.V(3).Infof("lexing character")
	if l.next() == 'x' {
		l.acceptRun("0123456789abcdefABCDEF")
	}

	l.emit(CHAR)

	return lexStart
}

func newLexer(input string, start stateFn) *lexer {
	l := &lexer{
		input: input,
		items: make(chan item, 2),
		idMap: idMap,
	}

	go l.run(start)

	return l
}

func (l *lexer) run(start stateFn) {
	for state := start; state != nil; {
		state = state(l)
	}

	close(l.items)
}

func (l *lexer) emit(t int) {
	i := item{
		t:     t,
		input: l.input[l.start:l.pos],
	}

	l.items <- i
	l.start = l.pos
}

func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width

	return r
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()

	return r
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}

	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}

	l.backup()
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		t:     -1,
		input: fmt.Sprintf(format, args...),
	}
	return nil
}
