// Copyright 2021 The CC Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cc // import "modernc.org/cc/v4"

import (
	"fmt"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf8"

	"modernc.org/strutil"
	mtoken "modernc.org/token"
)

var (
	extendedErrors = false // true: Errors will include origin info.
)

// origin returns caller's short position, skipping skip frames.
func origin(skip int) string {
	pc, fn, fl, _ := runtime.Caller(skip)
	f := runtime.FuncForPC(pc)
	var fns string
	if f != nil {
		fns = f.Name()
		if x := strings.LastIndex(fns, "."); x > 0 {
			fns = fns[x+1:]
		}
		if strings.HasPrefix(fns, "func") {
			num := true
			for _, c := range fns[len("func"):] {
				if c < '0' || c > '9' {
					num = false
					break
				}
			}
			if num {
				return origin(skip + 2)
			}
		}
	}
	return fmt.Sprintf("%s:%d:%s", filepath.Base(fn), fl, fns)
}

// todo prints and return caller's position and an optional message tagged with TODO. Output goes to stderr.
func todo(s string, args ...interface{}) string {
	switch {
	case s == "":
		s = fmt.Sprintf(strings.Repeat("%v ", len(args)), args...)
	default:
		s = fmt.Sprintf(s, args...)
	}
	r := fmt.Sprintf("%s\n\tTODO %s", origin(2), s)
	// fmt.Fprintf(os.Stderr, "%s\n", r)
	// os.Stdout.Sync()
	return r
}

// trc prints and return caller's position and an optional message tagged with TRC. Output goes to stderr.
func trc(s string, args ...interface{}) string {
	switch {
	case s == "":
		s = fmt.Sprintf(strings.Repeat("%v ", len(args)), args...)
	default:
		s = fmt.Sprintf(s, args...)
	}
	r := fmt.Sprintf("%s: TRC %s", origin(2), s)
	fmt.Fprintf(os.Stderr, "%s\n", r)
	os.Stderr.Sync()
	return r
}

// errorf constructs and error value. If extendedErrors is true, the error will
// continue its origin.
func errorf(s string, args ...interface{}) error {
	switch {
	case s == "":
		s = fmt.Sprintf(strings.Repeat("%v ", len(args)), args...)
	default:
		s = fmt.Sprintf(s, args...)
	}
	switch {
	case extendedErrors:
		return fmt.Errorf("%s (%v:)", s, origin(2))
	default:
		return fmt.Errorf("%s", s)
	}
}

// printHooks configure strutil.PrettyString for pretty printing Token values.
var printHooks = strutil.PrettyPrintHooks{
	reflect.TypeOf(Token{}): func(f strutil.Formatter, v interface{}, prefix, suffix string) {
		t := v.(Token)
		if t == (Token{}) {
			return
		}

		f.Format(prefix)
		if p := t.Position(); p.IsValid() {
			f.Format("%v: ", p)
		}
		f.Format("%s %q", t.Name(), t.Src())
		f.Format(suffix)
	},
	reflect.TypeOf((*Token)(nil)): func(f strutil.Formatter, v interface{}, prefix, suffix string) {
		t := v.(*Token)
		if t == nil {
			return
		}

		f.Format(prefix)
		if p := t.Position(); p.IsValid() {
			f.Format("%v: ", p)
		}
		f.Format("%s %q", t.Name(), t.Src())
		f.Format(suffix)
	},
}

// PrettyString returns a formatted representation of Tokens and AST nodes.
func PrettyString(v interface{}) string {
	return strutil.PrettyString(v, "", "", printHooks)
}

// runeName returns a human readable representation of ch.
func runeName(ch rune) string {
	switch {
	case ch < 0:
		return "<EOF>"
	case ch >= unicodePrivateAreaFirst && ch <= unicodePrivateAreaLast:
		return tokCh(ch).String()
	default:
		return fmt.Sprintf("%+q", ch)
	}
}

// env returns os.Getenv("key") or defaultVal if getenv returns an empty string.
func env(key, defaultVal string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}

	return defaultVal
}

func toksDump(v interface{}) string {
	// s0 := fmt.Sprintf("%T(%[1]p)", v)
	s0 := ""
	var a []string
	delta := 0
	switch x := v.(type) {
	case []cppToken:
		return toksDump(cppTokens(x))
	case *cppTokens:
		return toksDump(*x)
	case cppTokens:
		// if len(x) != 0 {
		// 	p := x[0].Position()
		// 	p.Filename = filepath.Base(p.Filename)
		// 	a = append(a, p.String())
		// 	delta = -1
		// }
		for _, v := range x {
			s := v.SrcStr()
			// if hs := v.hs.String(); hs != "[]" {
			// 	s = fmt.Sprintf("%s^%s", s, hs)
			// }
			a = append(a, s)
		}
	case []cppTokens:
		var a []string
		for _, v := range x {
			a = append(a, toksDump(v))
		}
		return fmt.Sprintf("%s%v.%d", s0, a, len(a))
	case controlLine:
		return toksDump([]Token(x))
	case *tokenizer:
		t := x.peek(0)
		return fmt.Sprintf("%s[%T %v]", s0, x, &t)
	case []Token:
		for _, v := range x {
			a = append(a, v.SrcStr())
		}
	case textLine:
		return toksDump([]Token(x))
	case *dequeue:
		d := *x
		for i := len(d) - 1; i >= 0; i-- {
			a = append(a, toksDump(d[i]))
		}
		return fmt.Sprintf("%s[%v].%d", s0, strings.Join(a, " · "), len(a))
	default:
		panic(todo("%T", x))
	}
	return fmt.Sprintf("%s%q.%d", s0, a, len(a)+delta)
}

func tokens2CppTokens(s []Token, skipFirstSep bool) (r []cppToken) {
	var spTok Token
	for i, v := range s {
		if (i != 0 || !skipFirstSep) && len(v.Sep()) != 0 {
			if spTok.s == nil {
				spTok = Token{s: s[0].s, Ch: ' '}
				spTok.Set(nil, sp)
			}
			r = append(r, cppToken{spTok, nil})
		}
		r = append(r, cppToken{v, nil})
	}
	return r
}

// func preprocessingTokens2Tokens(s []cppToken) (r []Token) {
// 	for _, v := range s {
// 		if v.Ch != ' ' {
// 			r = append(r, v.Token)
// 		}
// 	}
// 	return r
// }

func toksTrim(s cppTokens) cppTokens {
	for len(s) != 0 && s[0].Ch == ' ' {
		s = s[1:]
	}
	for len(s) != 0 && s[len(s)-1].Ch == ' ' {
		s = s[:len(s)-1]
	}
	return s
}

func stringConst(eh errHandler, tok Token) string {
	s := tok.SrcStr()
	switch tok.Ch {
	case rune(LONGSTRINGLITERAL):
		s = s[1:] // Remove leading 'L'.
		fallthrough
	case rune(STRINGLITERAL):
		var buf strings.Builder
		for i := 1; i < len(s)-1; {
			switch c := s[i]; c {
			case '\\':
				r, n := decodeEscapeSequence(eh, tok, s[i:])
				switch {
				case r < 0:
					buf.WriteByte(byte(-r))
				default:
					buf.WriteRune(r)
				}
				i += n
			default:
				buf.WriteByte(c)
				i++
			}
		}
		buf.WriteByte(0)
		return buf.String()
	}
	panic(todo("internal error"))
}

func charConst(eh errHandler, tok Token) rune {
	s := tok.SrcStr()
	switch tok.Ch {
	case rune(LONGCHARCONST):
		s = s[1:] // Remove leading 'L'.
		fallthrough
	case rune(CHARCONST):
		s = s[1 : len(s)-1] // Remove outer 's.
		if len(s) == 1 {
			return rune(s[0])
		}

		var r rune
		var n int
		switch s[0] {
		case '\\':
			r, n = decodeEscapeSequence(eh, tok, s)
			if r < 0 {
				r = -r
			}
		default:
			r, n = utf8.DecodeRuneInString(s)
		}
		if n != len(s) {
			eh("%v: invalid character constant", tok.Position())
		}
		return r
	}
	panic(todo("internal error"))
}

// escape-sequence		{simple-sequence}|{octal-escape-sequence}|{hexadecimal-escape-sequence}|{universal-character-name}
// simple-sequence		\\['\x22?\\abfnrtv]
// octal-escape-sequence	\\{octal-digit}{octal-digit}?{octal-digit}?
// hexadecimal-escape-sequence	\\x{hexadecimal-digit}+
func decodeEscapeSequence(eh errHandler, tok Token, s string) (rune, int) {
	if s[0] != '\\' {
		panic(todo("internal error"))
	}

	if len(s) == 1 {
		return rune(s[0]), 1
	}

	r := rune(s[1])
	switch r {
	case '\'', '"', '?', '\\':
		return r, 2
	case 'a':
		return 7, 2
	case 'b':
		return 8, 2
	case 'e':
		return 0x1b, 2
	case 'f':
		return 12, 2
	case 'n':
		return 10, 2
	case 'r':
		return 13, 2
	case 't':
		return 9, 2
	case 'v':
		return 11, 2
	case 'x':
		v, n := 0, 2
	loop2:
		for i := 2; i < len(s); i++ {
			r := s[i]
			switch {
			case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
				v = v<<4 | decodeHex(r)
				n++
			default:
				break loop2
			}
		}
		return -rune(v & 0xff), n
	case 'u', 'U':
		return decodeUCN(s)
	}

	if r < '0' || r > '7' {
		panic(todo(""))
	}

	v, n := 0, 1
	ok := false
loop:
	for i := 1; i < len(s); i++ {
		r := s[i]
		switch {
		case i < 4 && r >= '0' && r <= '7':
			ok = true
			v = v<<3 | (int(r) - '0')
			n++
		default:
			break loop
		}
	}
	if !ok {
		eh("%v: invalid octal sequence", tok.Position())
	}
	return -rune(v), n
}

// universal-character-name	\\u{hex-quad}|\\U{hex-quad}{hex-quad}
func decodeUCN(s string) (rune, int) {
	if s[0] != '\\' {
		panic(todo(""))
	}

	s = s[1:]
	switch s[0] {
	case 'u':
		return rune(decodeHexQuad(s[1:])), 6
	case 'U':
		return rune(decodeHexQuad(s[1:])<<16 | decodeHexQuad(s[5:])), 10
	}
	panic(todo(""))
}

// hex-quad	{hexadecimal-digit}{hexadecimal-digit}{hexadecimal-digit}{hexadecimal-digit}
func decodeHexQuad(s string) int {
	n := 0
	for i := 0; i < 4; i++ {
		n = n<<4 | decodeHex(s[i])
	}
	return n
}

func decodeHex(r byte) int {
	switch {
	case r >= '0' && r <= '9':
		return int(r) - '0'
	default:
		x := int(r) &^ 0x20
		return x - 'A' + 10
	}
}

func extractPos(s string) (p token.Position, ok bool) {
	var prefix string
	if len(s) > 1 && s[1] == ':' { // c:\foo
		prefix = s[:2]
		s = s[2:]
	}
	// "testdata/parser/bug/001.c:1193:6: ..."
	a := strings.SplitN(s, ":", 4)
	// ["testdata/parser/bug/001.c" "1193" "6" "..."]
	if len(a) < 3 {
		return p, false
	}

	line, err := strconv.Atoi(a[1])
	if err != nil {
		return p, false
	}

	col, err := strconv.Atoi(a[2])
	if err != nil {
		return p, false
	}

	return token.Position{Filename: prefix + a[0], Line: line, Column: col}, true
}

// func findPosition(n Node) (r mtoken.Position) {
// 	var out Node
// 	var depth int
// 	findNode("Token", n, mathutil.MaxInt, &out, &depth)
// 	if x, ok := out.(Token); ok {
// 		return x.Position()
// 	}
//
// 	return r
// }

func findNode(typ string, n Node, depth int, out *Node, pdepth *int) {
	if depth >= *pdepth {
		return
	}

	v := reflect.ValueOf(n)
	if v.Kind() != reflect.Ptr {
		return
	}

	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return
	}

	t := reflect.TypeOf(elem.Interface())
	if t.Name() == typ {
		*pdepth = depth
		*out = n
		return
	}

	for i := 0; i < elem.NumField(); i++ {
		fld := t.Field(i)
		if nm := fld.Name[0]; nm < 'A' || nm > 'Z' {
			continue
		}

		if x, ok := elem.FieldByIndex([]int{i}).Interface().(Node); ok {
			findNode(typ, x, depth+1, out, pdepth)
		}
	}
}

func roundup(n, to int64) int64 {
	if r := n % to; r != 0 {
		return n + to - r
	}

	return n
}

func bits2AccessBytes(n int64) int64 {
	switch {
	case n <= 8:
		return 1
	case n <= 16:
		return 2
	case n <= 32:
		return 4
	default:
		return 8
	}
}

func bool2int(b bool) Int64Value {
	if b {
		return int1
	}

	return int0
}

//lint:ignore U1000 debug helper
func pos(n Node) (r token.Position) {
	if n != nil {
		r = token.Position(n.Position())
		r.Filename = filepath.Base(r.Filename)
	}
	return r
}

//lint:ignore U1000 debug helper
func position(n Node) (r token.Position) {
	if n != nil {
		r = token.Position(n.Position())
	}
	return r
}

type fsetItem struct {
	file *mtoken.File

	off  int32
	size int32
}

type fset struct {
	items []fsetItem

	off int32
	sz  int32
}

func newFset() *fset { return &fset{} }

func (f *fset) add(file *mtoken.File) (r int32, err error) {
	r = f.off
	if off := int(f.off) + file.Size() + 1; off > 0 && off <= math.MaxInt32 {
		it := fsetItem{file, r, int32(file.Size())}
		f.items = append(f.items, it)
		f.off = int32(off)
		return r, nil
	}

	return -1, fmt.Errorf("file set size overflow")
}

func (f *fset) Position(pos int32) mtoken.Position {
	l, h := 0, int(len(f.items))-1
	var it fsetItem
	for l <= h {
		m := (l + h) / 2
		switch it = f.items[m]; {
		case pos < it.off:
			h = m - 1
		case pos > it.off+it.size:
			l = m + 1
		default:
			return it.file.Position(mtoken.Pos(pos - it.off))
		}
	}
	it = f.items[l-1]
	return it.file.Position(mtoken.Pos(pos - it.off))
}
