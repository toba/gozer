; Highlights for Go templates
; Based on https://github.com/hjr265/zed-gotmpl (MIT License)
; and https://github.com/ngalaiko/tree-sitter-go-template

; Identifiers
(variable) @variable
(field) @variable.member
(selector_expression
  operand: (field) @variable.member)
(function_call
  function: (identifier) @function)

; Punctuation
"." @punctuation.delimiter
"(" @punctuation.bracket
")" @punctuation.bracket
"|" @punctuation.delimiter

; Delimiters
"{{" @punctuation.special
"}}" @punctuation.special
"{{-" @punctuation.special
"-}}" @punctuation.special

; Keywords
"block" @keyword
"define" @keyword
"else" @keyword
"end" @keyword
"if" @keyword
"range" @keyword
"template" @keyword
"with" @keyword

; Operators
":=" @operator
"=" @operator
(function_call
  function: (identifier) @function.builtin
  (#any-of? @function.builtin
    "and" "call" "html" "index" "slice" "js" "len"
    "not" "or" "print" "printf" "println" "urlquery"
    "eq" "ge" "gt" "le" "lt" "ne"))

; Strings
(interpreted_string_literal) @string
(raw_string_literal) @string
(rune_literal) @string
(escape_sequence) @string.escape

; Numbers
(int_literal) @number
(float_literal) @number
(imaginary_literal) @number

; Booleans
(true) @constant.builtin
(false) @constant.builtin

; Nil
(nil) @constant.builtin

; Comments
(comment) @comment
