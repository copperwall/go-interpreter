package lexer

import (
	"monkey/token"
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
}

// Read characters until we've read past the whitespace
// This "consumes" the whitespace
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

	switch l.ch {
	case '=':
		if l.peakChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{
				Type:    token.EQ,
				Literal: literal,
			}
		} else {
			tok = newToken(token.ASSIGN, '=')
		}
	case '!':
		if l.peakChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{
				Type:    token.NOT_EQ,
				Literal: literal,
			}
		} else {
			tok = newToken(token.BANG, '!')
		}
	case '/':
		tok = newToken(token.SLASH, '/')
	case '*':
		tok = newToken(token.ASTERISK, '*')
	case '<':
		tok = newToken(token.LT, '<')
	case '>':
		tok = newToken(token.GT, '>')
	case ';':
		tok = newToken(token.SEMICOLON, ';')
	case '(':
		tok = newToken(token.LPAREN, '(')
	case ')':
		tok = newToken(token.RPAREN, ')')
	case ',':
		tok = newToken(token.COMMA, ',')
	case ':':
		tok = newToken(token.COLON, ':')
	case '+':
		tok = newToken(token.PLUS, '+')
	case '-':
		tok = newToken(token.MINUS, '-')
	case '{':
		tok = newToken(token.LBRACE, '{')
	case '}':
		tok = newToken(token.RBRACE, '}')
	case '[':
		tok = newToken(token.LBRACKET, '[')
	case ']':
		tok = newToken(token.RBRACKET, ']')
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}
	}

	l.readChar()
	return tok
}

// Read until it's not a letter
func (l *Lexer) readIdentifier() string {
	// this is the start index of our identifier
	position := l.position

	// advance isLetter
	for isLetter(l.ch) {
		l.readChar()
	}

	// At this point readPosition is the character
	// past the end of the identifer, but position is
	// at the end of the identifier

	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	// Need an index to start
	position := l.position

	for isDigit(l.ch) {
		l.readChar()
	}

	return l.input[position:l.position]
}

func (l *Lexer) readString() string {
	// Record start position of the string
	position := l.position + 1

	// Advance lexer until we get the next '"' or EOF
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}

	return l.input[position:l.position]
}

func newToken(tokenType token.TokenType, literal byte) token.Token {
	return token.Token{
		Type:    tokenType,
		Literal: string(literal),
	}
}

func isLetter(char byte) bool {
	return 'a' <= char && char <= 'z' || 'A' <= char && char <= 'Z' || char == '_'
}

func isDigit(char byte) bool {
	return '0' <= char && char <= '9'
}

// Read the next character into ch and update existing state
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}

	l.position = l.readPosition
	l.readPosition += 1
}

func (l *Lexer) peakChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	} else {
		return l.input[l.readPosition]
	}
}

func New(input string) *Lexer {
	l := &Lexer{
		input: input,
	}
	l.readChar()

	return l
}
