; Inject HTML into the full template, including children (template actions).
; This gives the HTML parser one continuous range so attribute values
; containing template expressions (e.g. value="{{.Email}}") don't create
; gaps that break HTML coloring for the rest of the file.
((template) @injection.content
  (#set! injection.language "html")
  (#set! injection.include-children))
