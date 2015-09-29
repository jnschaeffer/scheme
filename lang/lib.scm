(define (map f x)
  (if (eq? x '())
	'()
	(cons (f (car x))
		  (map f (cdr x)))))

(define (not x) (if x #f #t))

(define (cadr x) (car (cdr x)))


(defmacro let
  (lambda (bindings body1 . rest)
	`((lambda ,(map car bindings) ,@(cons body1 rest)) ,@(map cadr bindings))))

(defmacro let*
  (lambda (bindings body1 . rest)
	(if (eq? bindings `())
	  `(let () ,@(cons body1 rest))
	  `(let (,(car bindings))
		 (let* ,(cdr bindings) body1 ,@rest)))))

(defmacro letrec
  (lambda (bindings body1 . rest)
    `((lambda () ,@(map (lambda (x) `(define ,(car x) ,(cadr x))) bindings)
             ,@(cons body1 rest)))))

(defmacro or
  (lambda vals
    (if (eq? vals '())
      `#f
      `(if ,(car vals)
         #t
         (or ,@(cdr vals))))))

(defmacro and
  (lambda vals
    (if (eq? vals '())
      `#t
      `(if (not ,(car vals))
         #f
         (and ,@(cdr vals))))))

(define (cdxr n)
  (letrec ((helper
         (lambda (n x)
           (if (eq? n 0)
               x
               (helper (- n 1) (cdr x))))))
    (lambda (x) (helper n x))))

(define (cadxr n)
  (lambda (x)
    (car ((cdxr n) x))))

(define cddr (cdxr 2))
(define cdddr (cdxr 3))
(define cddddr (cdxr 4))
(define caddr (cadxr 2))
(define cadddr (cadxr 3))
(define caddddr (cadxr 4))
