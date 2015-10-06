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
		 (let* ,(cdr bindings) ,body1 ,@rest)))))

(define-syntax letrec*
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
  (letrec* ((helper
            (lambda (n x)
              (if (eq? n 0)
                  x
                  (helper (- n 1) (car x))))))
    (lambda (x) (helper n x))))

(define (cdxr n)
  (letrec* ((helper
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
  (letrec* ((helper (lambda (accum rest)
                     (if (null? rest)
                         accum
                         (helper (cons (car rest) accum) (cdr rest))))))
    (helper '() x)))

(define-syntax include
  (lambda (f)
    (letrec* ((p (open-input-file f))
           (read-file (lambda (accum p)
                        (let ((o (read p)))
                          (if (eof-object? o)
                              (reverse accum)
                              (read-file (cons o accum) p))))))
      `(begin ,@(read-file `() p)))))

(define inc/k
  (lambda (x k)
    (k (+ x 1))))

(define add/k
  (lambda (x1 x2 k)
    (k (+ x1 x2))))

(define twice/k
  (lambda (f k1)
    (k1 (lambda (x k2)
          (f x (lambda (fx)
                 (f fx k2)))))))

(define compose-twice/k
  (lambda (g f k1)
    (k1 (lambda (x k2)
          (f x (lambda (fx1)
                 (f x (lambda (fx2)
                        (g fx1 fx2 k2)))))))))

(define (eq?/k k x y) (k (eq? x y)))
(define (sub/k k x y) (k (- x y)))
(define (*/k k x y) (k (* x y)))

(define f-aux/k
  (lambda (n a)
    (if (eq?/k n 0)
        a
        (f-aux/k (sub/k n 1) (*/k n a)))))

(define f-aux/k2
  (lambda (k0 n a)
    (eq?/k (lambda (k1)
             (if k1
                 (k0 a)
                 (*/k (lambda (k3)
                        (sub/k (lambda (k2)
                                 (f-aux/k2 k0 k2 k3)) n 1)) n a))) n 0)))

(define fact/k
  (lambda (n)
    (if (eq?/k n 0)
        1
        (*/k n (fact/k (sub/k n 1))))))

(define fact/k2
  (lambda (k7 n)
    (eq?/k (lambda (k8)
             (if k8
                 (k7 1)
                 (sub/k (lambda (k10)
                          (fact/k2 (lambda (k9)
                                    (*/k k7 n k9)) k10)) n 1))) n 0)))

