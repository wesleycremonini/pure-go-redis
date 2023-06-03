package main

import (
	"bufio"
	"errors"
	"net"
	"strconv"
)

// Parser parses commands from the tcp connection.
type Parser struct {
	conn net.Conn
	r    *bufio.Reader
	line []byte
	pos  int
}

// NewParser returns a new Parser for a connection.
func NewParser(conn net.Conn) *Parser {
	return &Parser{
		conn: conn,
		r:    bufio.NewReader(conn),
		line: make([]byte, 0),
		pos:  0,
	}
}

func (p *Parser) current() byte {
	if p.atEnd() {
		return '\r'
	}
	return p.line[p.pos]
}

func (p *Parser) next() {
	p.pos++
}

func (p *Parser) atEnd() bool {
	return p.pos >= len(p.line)
}

func (p *Parser) readLine() ([]byte, error) {
	// '\r' indicates the end of a line or header in protocols like TCP and HTTP
	line, err := p.r.ReadBytes('\r')
	if err != nil {
		return nil, err
	}
	if _, err := p.r.ReadByte(); err != nil {
		return nil, err
	}

	// exclude the last byte, which is the '\r'
	return line[:len(line)-1], nil
}

// consumeString reads a string from the current line.
// Assumes that the initial " has been consumed before entering the function.
func (p *Parser) consumeString() (s []byte, err error) {
	for p.current() != '"' && !p.atEnd() { // TODO:
		cur := p.current()
		p.next()
		next := p.current()

		if cur == '\\' && next == '"' {
			// its an escaped quote
			s = append(s, '"')

			// call next again because we already consumed the "next" variable
			// which would be the new current char in the next iteration
			p.next()
			continue
		}

		s = append(s, cur)
	}
	if p.current() != '"' {
		return nil, errors.New("missing closing quotes in req")
	}
	p.next()
	return
}

// command parses and returns a Command.
func (p *Parser) command() (Command, error) {
	// check first character
	b, err := p.r.ReadByte()
	if err != nil {
		return Command{}, err
	}

	if b == '*' {
		// its an array
		return p.respArray()
	}

	line, err := p.readLine()
	if err != nil {
		return Command{}, err
	}
	p.pos = 0
	p.line = append(append([]byte{}, b), line...)
	return p.inline()
}

// inline parses an inline message and returns a Command.
func (p *Parser) inline() (Command, error) {
	for p.current() == ' ' {
		// skip whitespaces
		p.next()
	}
	cmd := Command{conn: p.conn}

	for !p.atEnd() {
		arg, err := p.consumeArg()
		if err != nil {
			return cmd, err
		}
		if arg != "" {
			cmd.args = append(cmd.args, arg)
		}
	}
	return cmd, nil
}

// consumeArg reads an argument from the current line.
func (p *Parser) consumeArg() (s string, err error) {
	for p.current() == ' ' {
		// skip whitespaces
		p.next()
	}
	if p.current() == '"' {
		p.next()
		buf, err := p.consumeString()
		return string(buf), err
	}
	for !p.atEnd() && p.current() != ' ' && p.current() != '\r' {
		s += string(p.current())
		p.next()
	}
	return
}

// respArray parses a RESP array and returns a Command.
func (p *Parser) respArray() (Command, error) {
	cmd := Command{}
	elements, err := p.readLine()
	if err != nil {
		return cmd, err
	}
	// in this case, the first line is the quantity of items in the array
	qty, _ := strconv.Atoi(string(elements))

	// loop all items
	for i := 0; i < qty; i++ {
		lineType, err := p.r.ReadByte()
		if err != nil {
			return cmd, err
		}
		switch lineType {
		case ':': // integer
			arg, err := p.readLine()
			if err != nil {
				return cmd, err
			}
			cmd.args = append(cmd.args, string(arg))
		case '$': // string length
			arg, err := p.readLine()
			if err != nil {
				return cmd, err
			}
			length, _ := strconv.Atoi(string(arg))
			text := make([]byte, 0)
			for i := 0; len(text) <= length; i++ {
				// read until the specified length if fulfilled
				line, err := p.readLine()
				if err != nil {
					return cmd, err
				}
				text = append(text, line...)
			}
			cmd.args = append(cmd.args, string(text[:length]))
		case '*': // its an array of arrays
			next, err := p.respArray()
			if err != nil {
				return cmd, err
			}
			cmd.args = append(cmd.args, next.args...)
		}
	}
	return cmd, nil
}