package httprule

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

const (
	invalidChar = byte(0)
)

var (
	errParserInternal   = errors.New("parser internal error")
	errEmptyLiteral     = errors.New("empty literal is not allowed")
	errInitialCharAlpha = errors.New("initial char of identifier not alpha")
	errEmptyIdent       = errors.New("empty identifier")
	errNestedVar        = errors.New("nested variables are not allowed")
	errDeepWildcard     = errors.New("deep wildcard must be the last segment")
	errDupFieldPath     = errors.New("dup field path")
	errLeadingSlash     = errors.New("leading slash required")
)

// parser is the template parser.
type parser struct {
	urlPath string // the complete httprule URL path.
	curr    int    // current pointer position.
}

// Parse parses the httprule URL path into template.
func Parse(urlPath string) (*PathTemplate, error) {
	p := &parser{
		urlPath: urlPath,
	}

	tpl, err := p.parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse url path %s to template: %w, curr: %d", urlPath, err, p.curr)
	}

	return tpl, nil
}

// parse begins parsing.
func (p *parser) parse() (*PathTemplate, error) {
	// should start with '/'.
	if err := p.consume('/'); err != nil {
		return nil, err
	}

	// parse segments.
	segments, err := p.parseSegments()
	if err != nil {
		return nil, err
	}
	// parse verb.
	var verb string
	// If the last segment is of type literal, then verb has already been included.
	// Find the last position of ':' in the literal.
	lastSegment := segments[len(segments)-1]
	if lastSegment.kind() == kindLiteral {
		s := lastSegment.String()
		idx := strings.LastIndex(s, ":")
		if idx > 0 {
			verb = s[idx+1:]
			segments[len(segments)-1] = literal(s[:idx])
		}
	} else {
		if err := p.consume(':'); err == nil {
			verb, err = p.parseVerb()
			if err != nil {
				return nil, err
			}
		}
	}

	// check whether parsing is completed.
	if !p.done() {
		return nil, errParserInternal
	}

	// validate.
	tpl := &PathTemplate{
		segments: segments,
		verb:     verb,
	}
	if err := p.validate(tpl); err != nil {
		return nil, err
	}

	return tpl, nil
}

// validate validates the template:
// 1. whether has nested variables
// 2. whether ** is the last segment
// 3. whether exists duplicate variable names
func (p *parser) validate(tpl *PathTemplate) error {
	m := make(map[string]bool) // save duplicate variable names

	for i, segment := range tpl.segments {
		// If it is of type variable, first check whether it is duplicated,
		// then check its nested segments:
		// 1. whether has nested variables
		// 2. if i != len(tpl.segments) - 1, then nested variables should not have **
		// 3. if i == len(tpl.segments) - 1, then ** has to be the last nested variable
		if segment.kind() == kindVariable {
			// check duplication
			s := strings.Join(segment.fieldPath(), ".")
			if m[s] {
				return errDupFieldPath
			}
			m[s] = true

			// check nested segments.
			nestedSegments := segment.nestedSegments()
			for j, nestedSegment := range nestedSegments {
				// nested segment is of kind variable.
				if nestedSegment.kind() == kindVariable {
					return errNestedVar
				}

				// If i != len(tpl.segments) - 1, then nested variables should not have **.
				if i != len(tpl.segments)-1 && nestedSegment.kind() == kindDeepWildcard {
					return errDeepWildcard
				}

				// If i == len(tpl.segments) - 1, then ** has to be the last nested variable.
				if i == len(tpl.segments)-1 && j != len(nestedSegments)-1 &&
					nestedSegment.kind() == kindDeepWildcard {
					return errDeepWildcard
				}
			}
		}

		// It is illegal if ** does not appear as the last segment.
		if i != len(tpl.segments)-1 && segment.kind() == kindDeepWildcard {
			return errDeepWildcard
		}
	}

	return nil
}

// parseSegments parses segments.
func (p *parser) parseSegments() ([]segment, error) {
	// at lease has one segment.
	seg, err := p.parseSegment()
	if err != nil {
		return nil, err
	}

	result := []segment{seg}

	if err := p.consume('/'); err == nil {
		// parse segments recursively.
		segs, err := p.parseSegments()
		if err != nil {
			return nil, err
		}
		result = append(result, segs...)
	}

	return result, nil
}

// parseVerb parses verb.
func (p *parser) parseVerb() (string, error) {
	return p.parseLiteral()
}

// parseSegment parses a single segment.
func (p *parser) parseSegment() (segment, error) {
	switch p.currentChar() {
	case invalidChar:
		return nil, errParserInternal
	case '*':
		if p.peekN(1) == '*' {
			p.curr++
			p.curr++
			return deepWildcard{}, nil
		}
		p.curr++
		return wildcard{}, nil
	case '{':
		return p.parseVariableSegment()
	default:
		return p.parseLiteralSegment()
	}
}

// parseLiteral parses literal type.
// https://www.ietf.org/rfc/rfc3986.txt, P.49
//
//	pchar         = unreserved / pct-encoded / sub-delims / ":" / "@"
//	unreserved    = ALPHA / DIGIT / "-" / "." / "_" / "~"
//	sub-delims    = "!" / "$" / "&" / "'" / "(" / ")"
//	              / "*" / "+" / "," / ";" / "="
//	pct-encoded   = "%" HEXDIG HEXDIG
func (p *parser) parseLiteral() (string, error) {
	lit := bytes.Buffer{}

	for {
		// pchar = unreserved / pct-encoded / sub-delims / ":" / "@"
		if isUnreserved(rune(p.currentChar())) || isSubDelims(rune(p.currentChar())) ||
			p.currentChar() == '@' || p.currentChar() == ':' {
			lit.WriteByte(p.currentChar())
			p.curr++
			continue
		} else if isPCTEncoded(rune(p.currentChar()), rune(p.peekN(1)), rune(p.peekN(2))) {
			lit.WriteByte(p.currentChar())
			p.curr++
			lit.WriteByte(p.currentChar())
			p.curr++
			lit.WriteByte(p.currentChar())
			p.curr++
			continue
		} else {
			break
		}
	}

	// empty literal.
	if lit.Len() == 0 {
		return "", errEmptyLiteral
	}

	return lit.String(), nil
}

// parseLiteralSegment parses literal segment.
func (p *parser) parseLiteralSegment() (segment, error) {
	lit, err := p.parseLiteral()
	if err != nil {
		return nil, err
	}
	return literal(lit), nil
}

// parseVariableSegment parses variable segment.
func (p *parser) parseVariableSegment() (segment, error) {
	var v variable

	// variable must start with '{'.
	if err := p.consume('{'); err != nil {
		return nil, err
	}

	// parse fieldPath.
	fieldPath, err := p.parseFieldPath()
	if err != nil {
		return nil, err
	}
	v.fp = fieldPath

	// check whether has segments.
	if err := p.consume('='); err == nil {
		segments, err := p.parseSegments()
		if err != nil {
			return nil, err
		}
		v.segments = segments
	} else { // no segments, defaults to *.
		v.segments = []segment{wildcard{}}
	}

	// variable must end with '}'.
	if err := p.consume('}'); err != nil {
		return nil, err
	}

	return v, nil
}

// parseFieldPath parses field path.
func (p *parser) parseFieldPath() ([]string, error) {
	// at least has one ident.
	ident, err := p.parseIdent()
	if err != nil {
		return nil, err
	}

	result := []string{ident}

	if err := p.consume('.'); err == nil {
		// parse fieldPath recursively.
		fp, err := p.parseFieldPath()
		if err != nil {
			return nil, err
		}
		result = append(result, fp...)
	}
	return result, nil
}

// parseIdent parses ident, the valid format of ident is ([[:alpha:]_][[:alphanum:]_]*).
func (p *parser) parseIdent() (string, error) {
	ident := bytes.Buffer{}

	for {
		if ident.Len() == 0 && !isAlpha(rune(p.currentChar())) {
			return "", errInitialCharAlpha
		}
		if isAlpha(rune(p.currentChar())) || isDigit(rune(p.currentChar())) || p.currentChar() == '_' {
			ident.WriteByte(p.currentChar())
			p.curr++
			continue
		}
		break
	}

	// empty ident.
	if ident.Len() == 0 {
		return "", errEmptyIdent
	}
	return ident.String(), nil
}

func (p *parser) done() bool {
	return p.curr >= len(p.urlPath)
}

func (p *parser) currentChar() byte {
	if p.done() {
		return invalidChar
	}
	return p.urlPath[p.curr]
}

// consume consumes the given character.
func (p *parser) consume(c byte) error {
	if p.currentChar() == c {
		p.curr++
		return nil
	}
	return fmt.Errorf("failed to consume `%c`", c)
}

// peekN gets the character at position p.curr+n.
func (p *parser) peekN(n int) byte {
	peekIdx := p.curr + n
	if peekIdx < len(p.urlPath) {
		return p.urlPath[peekIdx]
	}
	return invalidChar
}

// isUnreserved checks whether the given rune is of type unreserved.
func isUnreserved(r rune) bool {
	if isAlpha(r) || isDigit(r) {
		return true
	}
	switch r {
	case '-', '.', '_', '~':
		return true
	default:
		return false
	}
}

func isAlpha(r rune) bool {
	return ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func isSubDelims(r rune) bool {
	switch r {
	case '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=':
		return true
	default:
		return false
	}
}

func isPCTEncoded(r1, r2, r3 rune) bool {
	return r1 == '%' && isHexDigit(r2) && isHexDigit(r3)
}

func isHexDigit(r rune) bool {
	switch {
	case '0' <= r && r <= '9':
		return true
	case 'A' <= r && r <= 'F':
		return true
	case 'a' <= r && r <= 'f':
		return true
	default:
		return false
	}
}
