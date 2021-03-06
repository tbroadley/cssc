package lexer

import "unicode"

// Token is the set of lexical tokens in CSS.
type Token int

// https://www.w3.org/TR/css-syntax-3/#consume-token
const (
	Illegal Token = iota

	EOF

	Whitespace

	Comma     // ,
	Colon     // :
	Semicolon // ;
	LParen    // (
	RParen    // )
	CDO       // <!--
	CDC       // -->
	LBracket  // [
	RBracket  // ]
	LCurly    // {
	RCurly    // }

	Comment       // /* comment */
	URL           // url(...)
	FunctionStart // something(
	At            // @keyword
	Hash          // #hash
	Number        // Number literal
	Percentage    // Percentage literal
	Dimension     // Dimension literal
	String        // String literal
	Ident         // Identifier
	Delim         // Delimiter (used for preserving tokens for subprocessors)
)

func (t Token) String() string {
	return tokens[t]
}

var tokens = [...]string{
	Illegal: "Illegal",

	EOF: "EOF",

	Whitespace: "WHITESPACE",

	Comment: "COMMENT",
	Delim:   "DELIMITER",

	Hash:       "HASH",
	Number:     "NUMBER",
	Percentage: "PERCENTAGE",
	Dimension:  "DIMENSION",
	String:     "STRING",
	Ident:      "IDENT",
	URL:        "URL",

	Comma:         ",",
	Colon:         ":",
	Semicolon:     ";",
	At:            "@",
	FunctionStart: "FUNCTION",

	CDO: "<!--",
	CDC: "-->",

	LParen: "(",
	RParen: ")",

	LBracket: "[",
	RBracket: "]",

	LCurly: "{",
	RCurly: "}",
}

// isHexDigit implements https://www.w3.org/TR/css-syntax-3/#hex-digit.
func isHexDigit(r rune) bool {
	return unicode.IsDigit(r) || (r >= 'A' && r <= 'F') || (r >= 'a' && r <= 'f')
}

// isWhitespace implements https://www.w3.org/TR/css-syntax-3/#whitespace.
func isWhitespace(r rune) bool {
	return r == '\n' || r == '\u0009' || r == ' '
}

// isNameStartCodePoint implements https://www.w3.org/TR/css-syntax-3/#name-start-code-point.
func isNameStartCodePoint(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r >= 0x80 || r == '_'
}

// isNonPrintable implements https://www.w3.org/TR/css-syntax-3/#non-printable-code-point.
func isNonPrintable(r rune) bool {
	return (r >= 0 && r <= 0x008) || (r == 0x0b) || (r >= 0x0e && r <= 0x1f) || r == 0x7f
}

// isNameCodePoint implements https://www.w3.org/TR/css-syntax-3/#name-code-point.
func isNameCodePoint(r rune) bool {
	return isNameStartCodePoint(r) || (r <= '9' && r >= '0') || r == '-'
}
