(define (map f x)
  (if (eq? x '())
	'()
	(cons (f (car x))
		  (map f (cdr x)))))

(define (not x) (if x #f #t))

(define-syntax let
  (lambda (bindings body1 . rest)
	`((lambda ,(map car bindings) ,@(cons body1 rest)) ,@(map cadr bindings))))

(define-syntax let*
  (lambda (bindings body1 . rest)
	(if (eq? bindings `())
	  `(let () ,@(cons body1 rest))
	  `(let (,(car bindings))
		 (let* ,(cdr bindings) body1 ,@rest)))))

(define-syntax or
  (lambda vals
    (if (eq? vals '())
      `#f
      `(if ,(car vals)
         #t
         (or ,@(cdr vals))))))

(define-syntax and
  (lambda vals
    (if (eq? vals '())
      `#t
      `(if (not ,(car vals))
         #f
         (and ,@(cdr vals))))))
