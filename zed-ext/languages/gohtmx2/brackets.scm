; Bracket matching for Go HTML templates with HTMX 2.x support
; Based on https://github.com/hjr265/zed-gotmpl (MIT License)

("{{" @open "}}" @close)
("{{-" @open "-}}" @close)
("(" @open ")" @close)
("\"" @open "\"" @close)
