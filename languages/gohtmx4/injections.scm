; Inject HTML into text nodes (content outside {{ }} template tags)
((text) @injection.content
  (#set! injection.language "html")
  (#set! injection.combined))
