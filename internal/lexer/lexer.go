// Package lexer implements a css lexer that is meant to be
// used in conjuction with its sibling parser package.
package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/stephen/cssc/internal/ast"
)

// Lexer lexes the input source. Callers push the lexer
// along with calls to Next(), which populate the current
// token and literals.
type Lexer struct {
	// ch is the last rune consumed with step(). If there are
	// no more runes, ch is -1.
	ch rune

	// pos is the current byte (not codepoint) offset within source.
	pos int

	// start is the position at the beginning of a Next() call. It's
	// only used for providing locations, not for processing.
	start int

	// lastPos is the last byte (not codepoint) offset within source.
	lastPos int

	// source is the current source code being lexed.
	source string

	// Current is the last token lexed by Next().
	Current Token

	// CurrentString is the last literal string lexed by Next(). It
	// is not cleared between valid literals.
	CurrentString string

	// CurrentNumeral is the last literal numeral lexed by Next(). It
	// is not cleared between valid literals.
	CurrentNumeral string
}

// NewLexer creates a new lexer for the source.
func NewLexer(source string) *Lexer {
	l := &Lexer{
		source: source,
	}
	l.step()
	l.Next()
	return l
}

// step consumes the next unicode rune and stores it.
func (l *Lexer) step() {
	cp, size := utf8.DecodeRuneInString(l.source[l.pos:])

	if size == 0 {
		l.ch = -1
		return
	}

	l.ch = cp
	l.lastPos = l.pos
	l.pos += size
}

// peek returns the next ith unconsumed rune but does not consume it.
// i is 0-indexed (0 is one ahead, 1 is two ahead, etc.)
func (l *Lexer) peek(i int) rune {
	cp, size := utf8.DecodeRuneInString(l.source[l.pos:])
	if size == 0 {
		return -1
	}
	return cp
}

// Location is the start offset of the current token in the source, i.e.
// the value of l.pos when Next() was called.
func (l *Lexer) Location() *ast.Loc {
	return &ast.Loc{Position: l.start}
}

// Range is the start to end offset of the current token in the source. The returned
// start should be the same as Location() and the end is the last position stepped through,
// i.e. l.lastPos.
func (l *Lexer) Range() (int, int) {
	return l.start, l.lastPos
}

// Expect is like Next, except it asserts the current token before moving on. Callers should
// pull CurrentLiteral / CurrentNumeral before calling this function.
func (l *Lexer) Expect(token Token) {
	if l.Current != token {
		l.errorf("expected %s, but got %s instead", token, l.Current)
	}
	l.Next()
}

// Next consumes the most recent r.
func (l *Lexer) Next() {
	// Run in a for-loop so that some types (e.g. whitespace) can use continue to
	// move on to the next token. Other codepaths will end in a return statement
	// at the end of a single iteration.
	for {
		// Mark the start after all whitespace has been skipped.
		l.start = l.lastPos
		switch l.ch {
		case -1:
			l.Current = EOF
			return

		case ';':
			l.Current = Semicolon
			l.step()

		case ':':
			l.Current = Colon
			l.step()

		case '+':
			if startsNumber(l.ch, l.peek(0), l.peek(1)) {
				l.nextNumericToken()
				return
			}

			l.nextDelimToken()

		case '-':
			if startsNumber(l.ch, l.peek(0), l.peek(1)) {
				l.nextNumericToken()
				return
			}

			if p0, p1 := l.peek(0), l.peek(1); p0 == '-' && p1 == '>' {
				l.Current = CDC
				return
			}

			if startsIdentifier(l.ch, l.peek(0), l.peek(1)) {
				l.nextIdentLikeToken()
			}

			l.nextDelimToken()

		case '<':
			if p0, p1, p2 := l.peek(0), l.peek(1), l.peek(2); p0 == '!' && p1 == '-' && p2 == '-' {
				l.Current = CDO
				return
			}

			// Otherwise save it as a delimiter.
			l.nextDelimToken()

		case '@':
			if startsIdentifier(l.peek(0), l.peek(1), l.peek(2)) {
				l.step() // Consume @.
				l.Current = AtKeyword

				start := l.lastPos
				l.nextName()
				l.CurrentString = l.source[start:l.lastPos]
				return
			}

			l.nextDelimToken()

		case '#':
			if isNameCodePoint(l.peek(0)) || startsEscape(l.peek(0), l.peek(1)) {
				l.Current = Hash

				l.step()
				start := l.lastPos
				l.nextName()
				l.CurrentString = l.source[start:l.lastPos]
				return
			}

			l.nextDelimToken()

		case ',':
			l.Current = Comma
			l.step()

		case '(':
			l.Current = LParen
			l.step()

		case ')':
			l.Current = RParen
			l.step()

		case '[':
			l.Current = LBracket
			l.step()

		case ']':
			l.Current = RBracket
			l.step()

		case '{':
			l.Current = LCurly
			l.step()

		case '}':
			l.Current = RCurly
			l.step()

		case '.':
			if startsNumber(l.peek(0), l.peek(1), l.peek(2)) {
				l.nextNumericToken()
				return
			}

			l.nextDelimToken()

		case '\\':
			if !startsEscape(l.ch, l.peek(0)) {
				l.errorf("parse error")
			}

			l.nextIdentLikeToken()

		case '/':
			l.step()
			if l.ch != '*' {
				l.errorf("expected * but got %c", l.ch)
			}
			l.step()
			start, end := l.lastPos, -1

		commentToken:
			for {
				switch l.ch {
				case '*':
					maybeEnd := l.lastPos
					l.step()
					if l.ch == '/' {
						l.step()
						end = maybeEnd
						break commentToken
					}
					l.step()
				case -1:
					l.errorf("unexpected EOF")
				default:
					l.step()
				}
			}
			l.Current = Comment
			l.CurrentString = l.source[start:end]

		case '"', '\'':
			mark := l.ch

			l.step()
			start, end := l.lastPos, -1

		stringToken:
			for {
				switch l.ch {
				case mark:
					end = l.lastPos
					l.step()
					break stringToken
				case '\n':
					l.errorf("unclosed string: unexepected newline")
				case '\\':
					l.step()

					switch l.ch {
					case '\n':
						l.step()
					case -1:
						l.errorf("unexpected EOF")
					default:
						if startsEscape(l.ch, l.peek(0)) {
							l.nextEscaped()
						}
					}
				case -1:
					l.errorf("unexpected EOF")
				default:
					l.step()
				}
			}

			l.Current = String
			l.CurrentString = l.source[start:end]

		default:
			if isWhitespace(l.ch) {
				l.step()
				// Don't return out because we only processed whitespace and
				// there's nothing interesting for the caller yet. We don't emit
				// whitespace-token.
				continue
			}

			if unicode.IsDigit(l.ch) {
				l.nextNumericToken()
				return
			}

			// https://www.w3.org/TR/css-syntax-3/#consume-ident-like-token
			if isNameStartCodePoint(l.ch) {
				l.nextIdentLikeToken()
				return
			}

			l.nextDelimToken()
		}

		return
	}
}

// startsIdentifier implements https://www.w3.org/TR/css-syntax-3/#would-start-an-identifier.
func startsIdentifier(p0, p1, p2 rune) bool {
	switch p0 {
	case '-':
		return p1 == '-' || startsEscape(p1, p2)
	case '\n':
		return false
	default:
		return isNameCodePoint(p1)
	}
}

// startsEscape implements https://www.w3.org/TR/css-syntax-3/#starts-with-a-valid-escape
func startsEscape(p0, p1 rune) bool {
	if p0 != '\\' {
		return false
	}

	if p1 == '\n' {
		return false
	}

	return true
}

// startsNumber implements https://www.w3.org/TR/css-syntax-3/#starts-with-a-number.
func startsNumber(p0, p1, p2 rune) bool {
	if p0 == '+' || p0 == '-' {
		if unicode.IsDigit(p1) {
			return true
		}

		if p1 == '.' && unicode.IsDigit(p2) {
			return true
		}

		return false
	}

	if p0 == '.' && unicode.IsDigit(p1) {
		return true
	}

	return unicode.IsDigit(p0)
}

// nextNumericToken implements https://www.w3.org/TR/css-syntax-3/#consume-a-numeric-token
// and sets the lexer state.
func (l *Lexer) nextNumericToken() {
	start := l.lastPos
	l.nextNumber()
	l.CurrentNumeral = l.source[start:l.lastPos]

	if startsIdentifier(l.ch, l.peek(0), l.peek(1)) {
		dimenStart := l.lastPos
		l.nextName()
		l.CurrentString = l.source[dimenStart:l.lastPos]
		l.Current = Dimension
	} else if l.ch == '%' {
		l.Current = Percentage
	} else {
		l.Current = Number
	}
}

// nextIdentLikeToken implements https://www.w3.org/TR/css-syntax-3/#consume-an-ident-like-token.
// The spec tells us to return a bad-url-token, but we
// are uninterested in best-effort interpretation for compilation.
func (l *Lexer) nextIdentLikeToken() {
	start := l.lastPos
	l.nextName()
	l.CurrentString = l.source[start:l.lastPos]

	// Here, we need to special case the url function because it supports unquoted string content.
	if strings.ToLower(l.CurrentString) == "url" && l.ch == '(' {
		l.step()
		for i := l.lastPos; i < len(l.source) && isWhitespace(l.ch); i++ {
			l.step()
		}

		if l.ch == '\'' || l.ch == '"' {
			l.Current = FunctionStart
			return
		}

		l.Current = URL
		urlStart := l.lastPos
		for {
			switch l.ch {
			case ')':
				l.CurrentString = l.source[urlStart:l.lastPos]
				l.step()
				return
			case -1:
				l.errorf("unexpected EOF")
			case '"', '\'', '(':
				l.errorf("unexpected token: %c", l.ch)
			case '\\':
				if startsEscape(l.ch, l.peek(0)) {
					l.nextEscaped()
					continue
				}

				l.errorf("unexpected token: %c", l.ch)
			default:
				if isWhitespace(l.ch) {
					l.step()
					continue
				}

				if isNonPrintable(l.ch) {
					l.errorf("unexpected token: %c", l.ch)
				}

				l.step()
			}
		}
	}

	// Otherwise, it's probably a normal function.
	if l.peek(0) == '(' {
		l.Current = FunctionStart
		return
	}

	l.Current = Ident
}

// nextNumber implements https://www.w3.org/TR/css-syntax-3/#consume-a-number
// and consumes a number. We don't distinguish between number and integer because
// it doesn't matter for us.
func (l *Lexer) nextNumber() {
	if l.ch == '+' || l.ch == '-' {
		l.step()
	}

	for unicode.IsDigit(l.ch) {
		l.step()
	}

	if l.ch == '.' && unicode.IsDigit(l.peek(0)) {
		l.step()
		l.step()

		for unicode.IsDigit(l.ch) {
			l.step()
		}
	}

	if p0, p1 := l.peek(0), l.peek(1); (l.ch == 'e' || l.ch == 'E') && (unicode.IsDigit(p0) ||
		((p0 == '+' || p0 == '-') && unicode.IsDigit(p1))) {
		l.step()
		if l.ch == '+' || l.ch == '-' {
			l.step()
		}

		for unicode.IsDigit(l.ch) {
			l.step()
		}
	}
}

// nextDelim consumes a codepoint and saves it as a delimiter token.
func (l *Lexer) nextDelimToken() {
	start := l.lastPos
	l.step()
	l.Current = Delim
	l.CurrentString = l.source[start:l.lastPos]
}

// nextName implements https://www.w3.org/TR/css-syntax-3/#consume-a-name.
// It consumes and returns a name, stepping the lexer forward.
func (l *Lexer) nextName() {
	for {
		if isNameCodePoint(l.ch) {
			l.step()
		} else if startsEscape(l.ch, l.peek(0)) {
			l.nextEscaped()
		} else {
			return
		}
	}
}

// nextEscaped consumes and returns an escaped codepoint, stepping the lexer forward.
// It implements https://www.w3.org/TR/css-syntax-3/#consume-escaped-code-point.
//
// Note that we do not need to interpret the codepoint for our purposes - we can record
// the byte offsets as-is for transformation.
func (l *Lexer) nextEscaped() {
	l.step()
	for i := 0; i < 5 && isHexDigit(l.ch); i++ {
		l.step()
		if isWhitespace(l.ch) {
			l.step()
		}
	}
}

// LocationError is an error that happened at a specific location
// in the source.
type LocationError struct {
	Message string

	source string
	start  int
	length int
}

// Error implements error. It's relatively slow because it needs to
// rescan the source to figure out line and column numbers.
func (l *LocationError) Error() string {
	lineNumber, lineStart := 1, 0
	for i, ch := range l.source[:l.start] {
		if ch == '\n' {
			lineNumber++
			lineStart = i
		}
	}

	lineEnd := len(l.source)
	for i, ch := range l.source[l.start:] {
		if ch == '\n' {
			lineEnd = i + l.start
			break
		}
	}

	line := l.source[lineStart:lineEnd]
	col := l.start - lineStart
	len := lineEnd - l.start

	return fmt.Sprintf("%s: %s\n%d:%d through %d", l.Message, line, lineNumber, col, len)
}

// locationErrof sends up a lexer panic with a custom location.
func (l *Lexer) locationErrorf(start, end int, f string, args ...interface{}) {
	panic((&LocationError{fmt.Sprintf(f, args...), l.source, start, end - start}).Error())
}

// errorf sends up a lexer panic at the range from start to lastPos.
func (l *Lexer) errorf(f string, args ...interface{}) {
	l.locationErrorf(l.start, l.lastPos, f, args...)
}