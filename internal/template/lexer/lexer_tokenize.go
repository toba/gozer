package lexer

import (
	"bytes"
	"errors"
)

// tokenizeLine tokenizes the content inside a single {{...}} block into tokens.
func tokenizeLine(data []byte, initialPosition Range) (*StreamToken, []Error) {
	tokenHandler := createTokenizer()
	data, isCommentAllowed, isTrimmed, err := handleExternalWhiteSpaceTrimmer(
		data,
		initialPosition,
	)

	if err != nil {
		tokenHandler.Errs = append(tokenHandler.Errs, err)
	}

	if isTrimmed[0] {
		initialPosition.Start.Character++
	}

	// Use pre-compiled patterns
	regIgnore := compiledPatterns.whitespace
	patternTokens := compiledPatterns.tokenPatterns

	var loc []int

	var isCurrentTokenSeparatedFromPrevious = true
	var isPreviousTokenAcceptBindingToken = true
	var found bool

	var lengthDataStart = -1
	var currentLocalLineNumber, currentLocalColumnNumber int
	var indexFirstEqualOperator = -1

	var parenthesisUnclosed = make([]*Token, 0, 3)

	currentLocalLineNumber = initialPosition.Start.Line
	currentLocalColumnNumber = initialPosition.Start.Character

	for len(data) > 0 && lengthDataStart != len(data) {
		lengthDataStart = len(data)
		found = false

		// Ignore White Space
		loc = regIgnore.FindIndex(data)
		if loc != nil && loc[0] == 0 {
			content := data[loc[0]:loc[1]]

			position := ConvertSingleIndexToTextEditorPosition(content, loc[1])
			if position.Line != 0 {
				currentLocalColumnNumber = 0
			}

			currentLocalLineNumber += position.Line
			currentLocalColumnNumber += position.Character

			isCurrentTokenSeparatedFromPrevious = true
			isPreviousTokenAcceptBindingToken = true
			data = data[loc[1]:]
		}

		// Match a pattern to a token
		for _, pattern := range patternTokens {
			loc = pattern.Regex.FindIndex(data)

			isCurrentTokenSeparatedFromPrevious = isCurrentTokenSeparatedFromPrevious ||
				pattern.CanBeRightAfterToken || isPreviousTokenAcceptBindingToken

			if loc != nil && loc[0] == 0 {
				if !isCurrentTokenSeparatedFromPrevious {
					break
				}

				pos := convertRangeIndexToTextEditorPosition(
					data,
					loc,
					currentLocalLineNumber,
					currentLocalColumnNumber,
				)
				pos.End.Character++
				currentLocalColumnNumber += loc[1]

				text := trimSuperflousCharacter(data[0:loc[1]], pattern.ID)
				tokenHandler.appendToken(pattern.ID, pos, text)
				token := tokenHandler.LastToken

				isPreviousTokenAcceptBindingToken = pattern.CanBeRightAfterToken
				isCurrentTokenSeparatedFromPrevious = false
				found = true
				data = data[loc[1]:]

				switch pattern.ID {
				case Comment:
					if !isCommentAllowed {
						err := errors.New(
							"no whitespace or characters between 'Comment' and '{{' or '}}'",
						)
						tokenHandler.appendError(err, token)
					}

				case LeftParen:
					parenthesisUnclosed = append(parenthesisUnclosed, token)

				case RightParen:
					size := len(parenthesisUnclosed)
					if len(parenthesisUnclosed) == 0 {
						// tokenHandler.appendError(errors.New("extra closing parenthesis ')'"), token)
						tokenHandler.appendError(
							errors.New("missing opening parenthesis '('"),
							token,
						)
						break
					}

					parenthesisUnclosed = parenthesisUnclosed[:size-1]

				case Assignment, DeclarationAssignment:
					if indexFirstEqualOperator < 0 {
						indexFirstEqualOperator = len(tokenHandler.Tokens) - 1
					}
				}

				break
			}
		}

		// If no matching token found, add to error list
		if !found && len(data) > 0 {
			loc = regIgnore.FindIndex(data)

			if loc == nil {
				loc = []int{0, len(data)}
			} else {
				loc = []int{0, loc[0]}
			}

			pos := convertRangeIndexToTextEditorPosition(
				data,
				loc,
				currentLocalLineNumber,
				currentLocalColumnNumber,
			)
			pos.End.Character++
			currentLocalColumnNumber += loc[1]

			var err error
			if isCurrentTokenSeparatedFromPrevious {
				switch data[0] {
				case '"':
					err = errors.New(
						"characters not recognized, did you mean a string?",
					)
				case '/':
					err = errors.New("comment syntax error")
				default:
					err = errors.New("character(s) not recognized")
				}
			} else {
				err = errors.New(
					"character(s) not recognized, perhaps separate the word?",
				)
			}

			kindError := NotFound
			if bytes.Equal(data[:loc[1]], []byte("{{")) ||
				bytes.Equal(data[:loc[1]], []byte("}}")) {
				err = errors.New("missing matching template delimiter pair")
				kindError = Unexpected
			}

			tokenHandler.appendToken(kindError, pos, data[:loc[1]])
			token := tokenHandler.LastToken
			tokenHandler.appendError(err, token)

			data = data[loc[1]:]
		}
	}

	if len(data) > 0 {
		tokenHandler.appendToken(Unexpected, initialPosition, data)
		token := tokenHandler.LastToken
		tokenHandler.appendError(errors.New("unexpected character(s)"), token)
	}

	if len(tokenHandler.Tokens) == 0 {
		tokenHandler.appendError(
			errors.New("empty template"),
			&Token{ID: NotFound, Range: initialPosition},
		)
	}

	for _, LeftParenthesis := range parenthesisUnclosed {
		// tokenHandler.appendError(errors.New("unclosed parenthesis '('"), LeftParenthesis)
		tokenHandler.appendError(
			errors.New("missing closing parenthesis ')'"),
			LeftParenthesis,
		)
	}

	stream := NewStreamToken(
		tokenHandler.Tokens,
		tokenHandler.FirstError,
		initialPosition,
		indexFirstEqualOperator,
	)

	return stream, tokenHandler.Errs
}

// handleExternalWhiteSpaceTrimmer processes the "-" trim markers at the edges of a
// template block (e.g., {{- or -}}). Returns the trimmed data, whether comments are
// allowed at each edge, which edges were trimmed, and any errors.
func handleExternalWhiteSpaceTrimmer(
	data []byte,
	pos Range,
) ([]byte, bool, [2]bool, *LexerError) {
	isLeftCommentAllowed, isRightCommentAllowed := false, false
	isRightTrimmed, isLeftTrimmed := false, false

	var err *LexerError = nil

	if len(data) < 2 {
		isCommentAllowed := isLeftCommentAllowed && isRightCommentAllowed
		isTrimmed := [2]bool{isLeftTrimmed, isRightTrimmed}

		return data, isCommentAllowed, isTrimmed, err
	}

	// Check if first char is '/' (for comments like {{/* ... */}})
	if data[0] == '/' {
		isLeftCommentAllowed = true
	}

	lastElement := len(data) - 1
	if data[lastElement] == '/' {
		isRightCommentAllowed = true
	}

	// Handle right trim marker (e.g., {{- /* ... */ -}})
	if data[lastElement] == '-' {
		isRightTrimmed = true
		data = data[:lastElement] // Trim right '-'

		lastElement = len(data) - 1
		isOkay := lastElement > 0

		if isOkay && bytes.ContainsAny(data[lastElement:], " \r\n\t\f\v") {
			isOkay = lastElement > 1
			if isOkay && data[lastElement-1] == '/' {
				isRightCommentAllowed = true
			}
		} else {
			pos.Start.Character--

			err = &LexerError{
				Err:   errors.New("'-' left operator cannot be next to non-whitespace"),
				Range: pos,
				Token: &Token{Value: []byte(".-"), ID: SpaceEater, Range: pos},
			}
		}
	}

	if data[0] == '-' {
		isLeftTrimmed = true
		data = data[1:] // Trim left '-'

		isOkay := len(data) > 0

		if isOkay && bytes.ContainsAny(data[:1], " \r\n\t\f\v") {
			isOkay = len(data) > 1
			if isOkay && data[1] == '/' {
				isLeftCommentAllowed = true
			}
		} else {
			pos.End.Character = pos.Start.Character + 2

			err = &LexerError{
				Err:   errors.New("'-' right operator cannot be next to non-whitespace"),
				Range: pos,
				Token: &Token{Value: []byte("-."), ID: SpaceEater, Range: pos},
			}
		}
	}

	isCommentAllowed := isLeftCommentAllowed && isRightCommentAllowed
	isTrimmed := [2]bool{isLeftTrimmed, isRightTrimmed}

	return data, isCommentAllowed, isTrimmed, err
}
