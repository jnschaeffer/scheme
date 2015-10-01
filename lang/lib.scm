(define (map f x)
  (if (eq? x '())
	'()
	(cons (f (car x))
		  (map f (cdr x)))))

(define (not x) (if x #f #t))

(define (cadr x) (car (cdr x)))

(define (cdar x) (cdr (car x)))

(define-syntax let
  (lambda (bindings body1 . rest)
	`((lambda ,(map car bindings) ,@(cons body1 rest)) ,@(map cadr bindings))))

(define-syntax let*
  (lambda (bindings body1 . rest)
	(if (eq? bindings `())
	  `(let () ,@(cons body1 rest))
	  `(let (,(car bindings))
		 (let* ,(cdr bindings) body1 ,@rest)))))

(define-syntax letrec
  (lambda (bindings body1 . rest)
    `((lambda () ,@(map (lambda (x) `(define ,(car x) ,(cadr x))) bindings)
             ,@(cons body1 rest)))))

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

(define (caxr n)
  (letrec ((helper
            (lambda (n x)
              (if (eq? n 0)
                  x
                  (helper (- n 1) (car x))))))
    (lambda (x) (helper n x))))

(define (cdxr n)
  (letrec ((helper
            (lambda (n x)
              (if (eq? n 0)
                  x
                  (helper (- n 1) (cdr x))))))
    (lambda (x) (helper n x))))

(define (caxdr n)
  (let ((f (caxr n)))
    (lambda (x)
      (f (cdr x)))))

(define (cdxar n)
  (let ((f (cdxr n)))
    (lambda (x)
      (f (car x)))))

(define caar (caxr 2))
(define cddr (cdxr 2))

(define caaar (caxr 3))
(define caadr (caxdr 2))
(define cddar (cdxar 2))
(define cdddr (cdxr 3))

(define caaaar (caxr 4))
(define caaadr (caxdr 3))
(define cdddar (cdxar 3))
(define cddddr (cdxr 4))

(define (null? x)
  (eq? x '()))

(define (list? x)
  (if (not (pair? x))
      #f
      (if (eq? x '())
          #t
          (list? (cdr x)))))

(define (make-list k . fill)
  (let ((val (if (not (null? fill))
                 (car fill)
                 #f)))
    (if (eq? k 0)
        '()
        (cons val (make-list (- k 1) val)))))

(define (list . obj)
  obj)

(define (reverse x)
  (letrec ((helper (lambda (accum rest)
                     (if (null? rest)
                         accum
                         (helper (cons (car rest) accum) (cdr rest))))))
    (helper '() x)))
