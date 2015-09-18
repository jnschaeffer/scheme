(define (map f x)
  (if (eq? x '())
	'()
	(cons (f (car x))
		  (map f (cdr x)))))

(define (not x) (if x #f #t))

(define-syntax let
  (lambda (bindings . body)
	`((lambda ,(map car bindings) ,@body) ,@(map cadr bindings))))

(define-syntax let*
  (lambda (bindings . body)
	(if (eq? bindings `())
	  `(let () ,@body)
	  `(let (,(car bindings))
		 (let* ,(cdr bindings) ,@body)))))

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
