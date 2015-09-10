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

%type <obj> datum simple_datum compound_datum list vector expr
%type <objs> list_items

%start expr

%%

expr:
  datum
  {
    root = $$
  }

datum:
  simple_datum | compound_datum

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
  LPAREN list_items RPAREN
  {
    $$ = emptyList
    for i := len($2)-1; i >= 0; i-- {
      $$ = cons($2[i], $$)
    }
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
    yylval.obj = &object{
      t: numT,
      v: item.input,
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
