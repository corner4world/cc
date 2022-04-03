// Copyright 2021 The CC Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate rm -f ast.go
//go:generate yy -o /dev/null -position -astImport "\"fmt\"\n\n\"modernc.org/token\"" -prettyString PrettyString -kind Case -noListKind -noPrivateHelpers -forceOptPos parser.yy
//go:generate sed -i "s/\\*.*Expression$/ExpressionNode/" ast.go
//go:generate stringer -output stringer.go -linecomment -type=tokCh,Kind
//go:generate sh -c "go test -run ^Example |fe"

// Package cc is a C99 compiler front end.
//
// Online documentation
//
// See https://godoc.org/modernc.org/cc/v4.
//
// Links
//
// Referenced from elsewhere:
//
//  [0]: http://www.open-std.org/jtc1/sc22/wg14/www/docs/n1256.pdf
//  [1]: https://www.spinellis.gr/blog/20060626/cpp.algo.pdf
//  [2]: https://jhjourdan.mketjh.fr/pdf/jourdan2017simple.pdf
package cc // import "modernc.org/cc/v4"

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"modernc.org/opt"
)

const (
	DmesgsFile = "/tmp/cc.log"

	// Builtin definitions used by package tests. Possibly usable also by consumers
	// of this package.
	Builtin = `
#define __extension__
#define __restrict_arr restrict

#ifndef __builtin_va_list
#define __builtin_va_list __builtin_va_list
typedef void *__builtin_va_list;
#endif

#ifndef __builtin_va_arg
#define __builtin_va_arg __builtin_va_arg
#define __builtin_va_arg(va, type) (*(type*)__builtin_va_arg_impl(va))
#endif

#define __builtin_offsetof(type, member) ((__SIZE_TYPE__)&(((type*)0)->member))
#define __builtin_types_compatible_p(t1, t2) __builtin_types_compatible_p_impl((t1)0, (t2)0)

#ifdef __SIZE_TYPE__
typedef __SIZE_TYPE__ __predefined_size_t;
#endif

#ifdef __WCHAR_TYPE__
typedef __WCHAR_TYPE__ __predefined_wchar_t;
#endif

#ifdef __PTRDIFF_TYPE__
typedef __PTRDIFF_TYPE__ __predefined_ptrdiff_t;
#endif

#define __FUNCTION__ __func__
#define __PRETTY_FUNCTION__ __func__

#ifdef __clang__
#define __builtin_convertvector(src, type) (*(type*)&src)
#endif
`
)

var (
	isTesting  bool
	traceFails bool
)

func init() { //TODO- DBG

}

// NewConfig returns the system C compiler configuration, or an error, if
// any. The function will look for the compiler first in the environment
// variable CC, then it'll try other options. Usually that means looking for
// the "cc" and "gcc" binary, in that order.
//
// Additional arguments (flags) in opts are passed to the system C compiler
// unchanged.  For example, the _REENTRANT preprocessor macro is defined when
// the -pthread flag is present.  The set of recognized keywords is adjusted to
// emulate gcc, see:
//
//	https://gcc.gnu.org/onlinedocs/gcc/Alternate-Keywords.html#Alternate-Keywords
//
// Execution of NewConfig is expensive, caching the results is recommended
// wherever possible.
func NewConfig(goos, goarch string, opts ...string) (r *Config, err error) {
	cc, predefined, includePaths, sysIncludePaths, keywords, err := newConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("NewConfig: %v", err)
	}

	switch fmt.Sprintf("%s/%s", goos, goarch) {
	case "netbsd/amd64":
		sysIncludePaths = append(sysIncludePaths, "/usr/pkg/include")
	case "freebsd/386":
		sysIncludePaths = append(sysIncludePaths, "/usr/local/include")
	}
	abi, err := NewABI(goos, goarch)
	if err != nil {
		return nil, err
	}

	includePaths = includePaths[:len(includePaths):len(includePaths)]
	sysIncludePaths = sysIncludePaths[:len(sysIncludePaths):len(sysIncludePaths)]
	return &Config{
		ABI:                 abi,
		CC:                  cc,
		Predefined:          predefined,
		HostIncludePaths:    includePaths,
		HostSysIncludePaths: sysIncludePaths,
		IncludePaths:        append([]string{""}, append(includePaths, sysIncludePaths...)...),
		SysIncludePaths:     sysIncludePaths,
		keywords:            keywords,
	}, nil
}

func newConfig(opts []string) (cc, predefined string, includePaths, sysIncludePaths []string, keywords map[string]rune, err error) {
	if Dmesgs {
		Dmesg("newConfig(%v)", opts)
		defer func() {
			var s string
			if err != nil {
				s = " (FAIL)"
			}
			Dmesg("newConfig: cc: %q includePaths: %v sysIncludePaths: %v err: %v%s", cc, includePaths, sysIncludePaths, err, s)
		}()
	}
	clone := func() {
		if keywords == nil {
			keywords = make(map[string]rune, len(defaultKeywords))
			for k, v := range defaultKeywords {
				keywords[k] = v
			}
		}
	}
	var args []string
	set := opt.NewSet()

	// https://gcc.gnu.org/onlinedocs/gcc/C-Dialect-Options.html

	set.Opt("ansi", func(opt string) error {
		args = append(args, opt)
		clone()
		delete(keywords, "asm")
		delete(keywords, "inline")
		delete(keywords, "typeof")
		return nil
	})
	set.Opt("fno-asm", func(opt string) error {
		args = append(args, opt)
		clone()
		delete(keywords, "asm")
		delete(keywords, "typeof")
		return nil
	})
	set.Arg("std", false, func(opt, val string) error {
		args = append(args, fmt.Sprintf("%s=%s", opt, val))
		if !strings.HasPrefix(val, "gnu") {
			clone()
			delete(keywords, "asm")
			delete(keywords, "typeof")
		}
		switch val {
		case "c89", "c90", "iso9899:1990", "iso9899:199409":
			clone()
			delete(keywords, "inline")
		}
		return nil
	})

	if err := set.Parse(opts, func(arg string) error {
		args = append(args, arg)
		return nil
	}); err != nil {
		return "", "", nil, nil, nil, errorf("parsing %v: %v", opts, err)
	}

	opts = args[:len(args):len(args)]
	for _, cc = range []string{os.Getenv("CC"), "cc", "gcc"} {
		if cc == "" {
			continue
		}

		cc, err = exec.LookPath(cc)
		if err != nil {
			continue
		}

		args := append(opts, "-dM", "-E", "-")
		pre, err := exec.Command(cc, args...).CombinedOutput()
		if err != nil {
			if Dmesgs {
				Dmesg("cc: %s %v ----\n%s\n----: %v", cc, args, pre, err)
			}
			continue
		}

		sep := "\n"
		if env("GOOS", runtime.GOOS) == "windows" {
			sep = "\r\n"
		}
		a := strings.Split(string(pre), sep)
		w := 0
		for _, v := range a {
			if strings.HasPrefix(v, "#") {
				a[w] = v
				w++
			}
		}
		predefined = strings.Join(a[:w], "\n")
		args = append(opts, "-v", "-E", "-")
		out, err := exec.Command(cc, args...).CombinedOutput()
		if err != nil {
			if Dmesgs {
				Dmesg("cc: %s %v ----\n%s\n----: %v", cc, args, pre, err)
			}
			continue
		}

		a = strings.Split(string(out), sep)
		for i := 0; i < len(a); {
			switch a[i] {
			case "#include \"...\" search starts here:":
			loop:
				for i = i + 1; i < len(a); {
					switch v := a[i]; {
					case strings.HasPrefix(v, "#") || v == "End of search list.":
						break loop
					default:
						includePaths = append(includePaths, strings.TrimSpace(v))
						i++
					}
				}
			case "#include <...> search starts here:":
				for i = i + 1; i < len(a); {
					switch v := a[i]; {
					case strings.HasPrefix(v, "#") || v == "End of search list.":
						return cc, predefined, includePaths, sysIncludePaths, keywords, nil
					default:
						sysIncludePaths = append(sysIncludePaths, strings.TrimSpace(v))
						i++
					}
				}
			default:
				i++
			}
		}
	}
	return "", "", nil, nil, nil, errorf("cannot determine C compiler configuration")
}

// Source is a named part of a translation unit. The name argument is used for
// reporting Token positions.  The value argument can be a string, []byte,
// fs.File, io.ReadCloser, io.Reader or nil. If the value argument is nil an
// attempt is made to open a file using the name argument.
//
// When the value argument is an *os.File, io.ReadCloser or fs.File,
// Value.Close() is called before returning from Preprocess, Parse or
// Translate.
//
// If FS is not nil it overrides the FS from Config.
type Source struct {
	Name  string
	Value interface{}
	FS    fs.FS
}

// Config configures the preprocessor, parser and type checker.
//
// Search paths listed in IncludePaths and SysIncludePaths are used to resolve
// #include "foo.h" and #include <foo.h> preprocessing directives respectively.
// A special search path "@" is interpreted as 'the same directory as where the
// file with the #include directive is'.
//
// If FS is nil, os.Open is used to open named files.
type Config struct {
	ABI                 *ABI
	CC                  string // The configured C compiler, filled by NewConfig.
	FS                  fs.FS
	HostIncludePaths    []string
	HostSysIncludePaths []string
	IncludePaths        []string
	PragmaHandler       func([]Token) error
	Predefined          string // The predefined macros from CC, filled by NewConfig.
	SysIncludePaths     []string
	keywords            map[string]rune

	doNotInjectFunc bool // testing
	fakeIncludes    bool // testing
}

type errors []string

// Error implements error.
func (e errors) Error() string { return strings.Join(e, "\n") }

func (e *errors) add(err error) { *e = append(*e, err.Error()) }

func (e errors) err() error {
	w := 0
	for i, v := range e {
		if i != 0 {
			if prev, ok := extractPos(e[i-1]); ok {
				if cur, ok := extractPos(v); ok && prev.Filename == cur.Filename && prev.Line == cur.Line {
					continue
				}
			}
		}
		e[w] = v
		w++
	}
	e = e[:w]
	if len(e) == 0 {
		return nil
	}

	return e
}

// Preprocess preprocesses a translation unit, consisting of inputs in sources,
// and writes the result to w.
func Preprocess(cfg *Config, sources []Source, w io.Writer) (err error) {
	cpp, err := newCPP(cfg, newFset(), sources, nil)
	if err != nil {
		return err
	}

	return preprocess(cpp, w)
}

func preprocess(cpp *cpp, w io.Writer) (err error) {
	var errors errors
	cpp.eh = func(msg string, args ...interface{}) { errors = append(errors, fmt.Sprintf(msg, args...)) }
	var prev rune
	for {
		if cpp.rune() == eof {
			return errors.err()
		}

		tok := cpp.shift()
		switch c := tok.Ch; {
		case
			// Prevent the textual form of certain adjacent tokens to form a "false" token,
			// not present in the source.
			c == '#' && prev == '#',
			c == '&' && prev == '&',
			c == '+' && prev == '+',
			c == '+' && prev == rune(PPNUMBER),
			c == '-' && prev == '-',
			c == '-' && prev == rune(PPNUMBER),
			c == '.' && prev == '.',
			c == '.' && prev == rune(PPNUMBER),
			c == '<' && prev == '<',
			c == '=' && prev == '!',
			c == '=' && prev == '%',
			c == '=' && prev == '&',
			c == '=' && prev == '*',
			c == '=' && prev == '+',
			c == '=' && prev == '/',
			c == '=' && prev == '<',
			c == '=' && prev == '=',
			c == '=' && prev == '^',
			c == '=' && prev == '|',
			c == '>' && prev == '-',
			c == '>' && prev == '>',
			c == '|' && prev == '|',
			c == rune(DEC) && prev == '-',
			c == rune(IDENTIFIER) && prev == rune(IDENTIFIER),
			c == rune(INC) && prev == '+':

			if _, err := w.Write(sp); err != nil {
				return err
			}
		}
		if prev == ' ' && tok.Ch == ' ' {
			continue
		}

		prev = tok.Ch
		if _, err = w.Write(tok.Src()); err != nil {
			return err
		}
	}
}

// Parse preprocesses and parses a translation unit, consisting of inputs in
// sources.
func Parse(cfg *Config, sources []Source) (*AST, error) {
	p, err := newParser(cfg, newFset(), sources)
	if err != nil {
		return nil, err
	}

	return p.parse()
}

// Translate preprocesses, parses and type checks a translation unit,
// consisting of inputs in sources.
func Translate(cfg *Config, sources []Source) (*AST, error) {
	ast, err := Parse(cfg, sources)
	if err != nil {
		return nil, err
	}

	if err := ast.check(); err != nil {
		return nil, err
	}

	return ast, nil
}

// NodeSource returns the source form of s. Non-empty separators preceding
// tokens are replaced by a single ' '.
func NodeSource(s ...Node) string {
	var a []Token
	for _, n := range s {
		nodeSource(n, &a)
	}
	sort.Slice(a, func(i, j int) bool { return a[i].seq < a[j].seq })
	var b strings.Builder
	for _, t := range a {
		if len(t.Sep()) != 0 {
			b.WriteByte(' ')
		}
		b.Write(t.Src())
	}
	return b.String()
}

func nodeSource(n Node, a *[]Token) {
	if n == nil {
		return
	}

	t := reflect.TypeOf(n)
	v := reflect.ValueOf(n)
	var zero reflect.Value
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		v = v.Elem()
		if v == zero {
			return
		}
	}

	if t.Kind() != reflect.Struct {
		return
	}

	if x, ok := n.(Token); ok && x.seq != 0 {
		*a = append(*a, x)
		return
	}

	nf := t.NumField()
	for i := 0; i < nf; i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		if strings.HasPrefix(f.Name, "Token") {
			if x, ok := v.Field(i).Interface().(Token); ok && x.seq != 0 {
				*a = append(*a, x)
			}
			continue
		}

		if v == zero || v.IsZero() {
			continue
		}

		if m, ok := v.Field(i).Interface().(Node); ok {
			nodeSource(m, a)
		}
	}
}
