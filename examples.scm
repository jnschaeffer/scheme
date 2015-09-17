(define (map f x)
  (if (eq? x '())
	'()
	(cons (f (car x))
		  (map f (cdr x)))))

(define-syntax let
  (lambda (bindings . body)
	`((lambda ,(map car bindings) ,@body) ,@(map cadr bindings))))

(define-syntax let*
  (lambda (bindings . body)
	(if (eq? bindings `())
	  `(let () ,@body)
	  `(let (,(car bindings))
		 (let* ,(cdr bindings) ,@body)))))
