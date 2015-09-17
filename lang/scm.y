%{
  package lang

  import (
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
%token IF

%type <obj> datum simple_datum compound_datum list vector expr quotation
%type <obj> literal self_evaluating procedure
%type <obj> quasiquote qq_template list_qq_template unquote derived
%type <objs> list_items exprs qq_templates

%start start

%%

start:
  expr
  {
    root = $1
  }

expr:
  IDENT
| literal
| procedure
| LPAREN RPAREN
  {
    $$ = emptyList
  }
| derived

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

qq_templates:
  qq_template
  {
    $$ = []*object{$1}
  }
| qq_templates qq_template
  {
    $$ = append($1, $2)
  }

list_qq_template:
  quasiquote
| LPAREN RPAREN
  {
    $$ = emptyList
  }
|  LPAREN qq_templates RPAREN
  {
    $$ = vecToList($2)
  }
| LPAREN qq_templates DOT qq_template RPAREN
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
| IDENT
| CHAR

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
  case IDENT:
    yylval.obj = &object{
      t: identT,
      v: item.input,
    }

    return IDENT
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
}

var root *object

func parse(s string) *object {
  root = nil

  exprParse(&exprLex{lexer: newLexer(s, lexStart)})

  return root
}
