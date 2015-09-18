%{
  package lang

  import (
    "fmt"
    "log"
    "strings"
  )

  var emptyList = &object{
    t: listT,
    v: nil,
  }

%}

%union {
  obj *object
  objs []*object
}

%token <obj> NUM STRING IDENT BOOLEAN CHAR
%token LPAREN RPAREN LVEC LU8VEC QUOTE BACKTICK COMMA COMMAAT DOT
%token WSPACE
%token <obj> IF LAMBDA DEFINE

%type <obj> datum simple_datum compound_datum list vector expr quotation
%type <obj> literal self_evaluating procedure
%type <obj> quasiquote qq_template list_qq_template unquote derived
%type <obj> qq_template_or_splice splicing_unquotation
%type <objs> list_items exprs qq_templates_or_splices idents
%type <obj> conditional lambda formals program definition def_formals datum_ident

%start start

%%

start:
  program
  {
    root = $1
  }
|
  {
    root = nil
  }

program:
  expr
| definition

definition:
  LPAREN DEFINE IDENT expr RPAREN
  {
    $$ = cons(symbolObj("define"), cons($3, cons($4, emptyList)))
  }
| LPAREN DEFINE LPAREN def_formals RPAREN exprs RPAREN
  {
    definition := $4
    body := vecToList($6)
    $$ = cons(symbolObj("define"), cons(definition, body))
  }

def_formals:
 idents
  {
    $$ = vecToList($1)
  }
| idents DOT IDENT
  {
    $$ = vecToImproperList(append($1, $3))
  }

expr:
  IDENT
| literal
| procedure
| conditional
| lambda
| LPAREN RPAREN
  {
    $$ = emptyList
  }
| derived

conditional:
  LPAREN IF expr expr RPAREN
  {
    $$ = cons(symbolObj("if"), cons($3, cons($4, emptyList)))
  }
| LPAREN IF expr expr expr RPAREN
  {
    $$ = cons(symbolObj("if"), cons($3, cons($4, cons($5, emptyList))))
  }

lambda:
  LPAREN LAMBDA formals exprs RPAREN
  {
	log.Printf("parsed lambda")
	e := vecToList($4)
    $$ = cons(symbolObj("lambda"), cons($3, e))
  }

formals:
  LPAREN RPAREN
  {
    $$ = emptyList
  }
| LPAREN idents RPAREN
  {
    $$ = vecToList($2)
  }
| IDENT
| LPAREN idents DOT IDENT RPAREN
  {
    o := append($2, $4)
    $$ = vecToImproperList(o)
  }

idents:
  IDENT
  {
    $$ = []*object{$1}
  }
| idents IDENT
  {
    $$ = append($1, $2)
  }

derived:
  quasiquote

procedure:
  LPAREN exprs RPAREN
  {
	$$ = vecToList($2)
  }

exprs:
  expr
  {
    $$ = []*object{$1}
  }
| exprs expr
  {
    $$ = append($1, $2)
  }

literal:
  quotation
| self_evaluating

quasiquote:
  BACKTICK qq_template
  {
    $$ = cons(symbolObj("quasiquote"), cons($2, emptyList))
  }

qq_template:
  simple_datum
| list_qq_template
| unquote

qq_template_or_splice:
  qq_template
| splicing_unquotation

qq_templates_or_splices:
  qq_template_or_splice
  {
    $$ = []*object{$1}
  }
| qq_templates_or_splices qq_template_or_splice
  {
    $$ = append($1, $2)
  }

list_qq_template:
  quasiquote
| LPAREN RPAREN
  {
    $$ = emptyList
  }
|  LPAREN qq_templates_or_splices RPAREN
  {
    $$ = vecToList($2)
  }
| LPAREN qq_templates_or_splices DOT qq_template RPAREN
  {
    $$ = $4
    for i := len($2)-1; i >= 0; i-- {
      $$ = cons($2[i], $$)
    }
  }

unquote:
  COMMA qq_template
  {
    $$ = cons(symbolObj("unquote"), cons($2, emptyList))
  }

splicing_unquotation:
  COMMAAT qq_template
  {
    $$ = cons(symbolObj("unquote-splicing"), cons($2, emptyList))
  }

self_evaluating:
  BOOLEAN
| NUM
| vector
| CHAR
| STRING

quotation:
  QUOTE datum
  {
	$$ = cons(symbolObj("quote"), cons($2, emptyList))
  }

datum:
  simple_datum | compound_datum | quotation

simple_datum:
  NUM
| STRING
| BOOLEAN
| datum_ident
| CHAR

datum_ident:
  IDENT
| IF
| LAMBDA
| DEFINE

compound_datum:
  list
| vector

list:
  LPAREN RPAREN
  {
    $$ = emptyList
  }
| LPAREN list_items RPAREN
  {
	$$ = vecToList($2)
  }
| LPAREN list_items DOT datum RPAREN
  {
    $$ = $4
    for i := len($2)-1; i >= 0; i-- {
      $$ = cons($2[i], $$)
    }
  }

list_items:
  datum
  {
    $$ = []*object{$1}
  }
| list_items datum
  {
    $$ = append($1, $2)
  }

vector:
  LVEC list_items RPAREN
  {
    $$ = &object{
      t: vecT,
      v: $2,
    }
  }

%%

const EOF = 0

type exprLex struct {
  lexer *lexer
}

func (x *exprLex) Lex(yylval *exprSymType) int {
  var (
    item item
    ok bool
  )

  for item, ok = <-x.lexer.items; item.t == WSPACE; item, ok = <-x.lexer.items {
  }

  if !ok {
    return 0
  }

  switch item.t {
  case NUM:
	n := parseNum(item.input)
    yylval.obj = &object{
      t: numT,
      v: n,
    }

    return NUM
  case STRING:
    yylval.obj = &object{
      t: strT,
      v: item.input,
    }
    
    return STRING
  case IDENT, IF, DEFINE, LAMBDA:
    yylval.obj = &object{
      t: identT,
      v: item.input,
    }

    return item.t
  case BOOLEAN:
    yylval.obj = &object{
      t: boolT,
      v: strings.HasPrefix("#t", item.input),
    }

    return BOOLEAN
  case CHAR:
    yylval.obj = &object{
      t: charT,
      v: item.input,
    }

    return CHAR
  default:
    return item.t
  }

  return EOF
}

func (x *exprLex) Error(e string) {
  log.Printf("error: %s\n", e)
  err = fmt.Errorf(e)
}

var root *object
var err error

func parse(s string) (*object, error) {
  root = nil
  err = nil

  exprParse(&exprLex{lexer: newLexer(s, lexStart)})

  return root, err
}
