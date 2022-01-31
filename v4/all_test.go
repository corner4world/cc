// Copyright 2021 The CC Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cc // import "modernc.org/cc/v4"

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/pmezard/go-difflib/difflib"
	"modernc.org/ccorpus"
	"modernc.org/httpfs"
	"modernc.org/mathutil"
	//TODO "modernc.org/scannertest"
)

var (
	cfs         = ccorpus.FileSystem()
	cFS         = &corpusFS{cfs}
	corpus      = map[string][]byte{}
	corpusIndex []string
	re          *regexp.Regexp
	testCfg0    = &Config{}
	predefined  string
	builtin     = `
#define __builtin_offsetof(type, member) ((__SIZE_TYPE__)&(((type*)0)->member))
#define __extension__


#ifndef __builtin_va_list
#define __builtin_va_list __builtin_va_list
typedef void *__builtin_va_list;
#endif

#ifndef __builtin_va_arg
#define __builtin_va_arg(va, type) (*(type*)__builtin_va_arg_sink(0, va))
void *__builtin_va_arg_sink(int, ...);
#endif

#ifdef __clang__
#define __builtin_convertvector(src, type) (*(type*)&src)
#endif
`

	oTrace = flag.Bool("trc", false, "Print tested paths.")
)

func init() {
	isTesting = true
	var err error
	if predefined, testCfg0.IncludePaths, testCfg0.SysIncludePaths, err = HostConfig(env("CC_TEST_CPP", "cpp")); err != nil {
		panic(errorf("cannot acquire host configuration: %v", err))
	}

	switch fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH) {
	case "netbsd/amd64":
		testCfg0.SysIncludePaths = append(testCfg0.SysIncludePaths, "/usr/pkg/include")
	case "freebsd/386":
		testCfg0.SysIncludePaths = append(testCfg0.SysIncludePaths, "/usr/local/include")
	}
	testCfg0.SysIncludePaths = testCfg0.SysIncludePaths[:len(testCfg0.SysIncludePaths):len(testCfg0.SysIncludePaths)]
	testCfg0.IncludePaths = append([]string{""}, testCfg0.IncludePaths...)
	testCfg0.IncludePaths = append(testCfg0.IncludePaths, testCfg0.SysIncludePaths...)
	testCfg0.IncludePaths = testCfg0.IncludePaths[:len(testCfg0.IncludePaths):len(testCfg0.IncludePaths)]
	if testCfg0.ABI, err = NewABI(runtime.GOOS, runtime.GOARCH); err != nil {
		panic(errorf("cannot configure ABI: %v", err))
	}

	var chars int
	if err := walk("/", func(pth string, fi os.FileInfo) error {
		if fi.IsDir() {
			return nil
		}

		f, err := cfs.Open(pth)
		if err != nil {
			return errorf("%v: %v", pth, err)
		}

		b, err := io.ReadAll(f)
		if err != nil {
			return errorf("%v: %v", pth, err)
		}

		switch filepath.Ext(pth) {
		case ".c", ".h":
			if len(b) != 0 && b[len(b)-1] != '\n' {
				b = append(b, '\n')
			}
		}
		chars += len(b)
		corpus[pth] = b
		corpusIndex = append(corpusIndex, pth)
		return nil
	}); err != nil {
		panic(err)
	}
}

type corpusFS struct {
	*httpfs.FileSystem
}

func (c *corpusFS) Open(name string) (fs.File, error) {
	name = filepath.ToSlash(name)
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	f, err := c.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}

	return fs.File(f), nil
}

func walk(dir string, f func(pth string, fi os.FileInfo) error) error {
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	root, err := cfs.Open(dir)
	if err != nil {
		return err
	}

	fi, err := root.Stat()
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		return fmt.Errorf("%s: not a directory", fi.Name())
	}

	fis, err := root.Readdir(-1)
	if err != nil {
		return err
	}

	for _, v := range fis {
		switch {
		case v.IsDir():
			if err = walk(v.Name(), f); err != nil {
				return err
			}
		default:
			if err = f(v.Name(), v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Produce the AST used in examples documentation.
func exampleAST(rule int, src string) (r interface{}) {
	defer func() {
		if err := recover(); err != nil {
			r = fmt.Sprintf("%v (%v:)", err, origin(5))
		}
	}()

	src = strings.Replace(src, "\\n", "\n", -1)
	cfg := &Config{}
	ast, _ := Parse(cfg, []Source{{Name: "example.c", Value: src}})
	if ast == nil {
		return "FAIL"
	}

	pc, _, _, _ := runtime.Caller(1)
	typ := runtime.FuncForPC(pc - 1).Name()
	i := strings.LastIndexByte(typ, '.')
	typ = typ[i+1+len("Example"):]
	i = strings.LastIndexByte(typ, '_')
	typ = typ[:i]
	var node Node
	depth := mathutil.MaxInt
	findNode(typ, ast.TranslationUnit, 0, &node, &depth)
	return node
}

func TestMain(m *testing.M) {
	oRE := flag.String("re", "", "")
	flag.Parse()
	if *oRE != "" {
		re = regexp.MustCompile(*oRE)
	}
	os.Exit(m.Run())
}

func TestScannerSource(t *testing.T) {
	const fn = "all_test.go"
	exp, err := os.ReadFile(fn)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(fn)
	if err != nil {
		t.Fatal(err)
	}

	testScannerSource(t, fn, f, exp, false)
	testScannerSource(t, fn, exp, exp, false)
	testScannerSource(t, fn, string(exp), exp, false)
	testScannerSource(t, fn, bytes.NewReader(exp), exp, false)
	testScannerSource(t, fn, nil, exp, false)
	testScannerSource(t, fn, 42, nil, true)
}

func testScannerSource(t *testing.T, name string, value interface{}, exp []byte, mustFail bool) {
	ss, err := newScannerSource(Source{name, value, nil})
	if err != nil != mustFail {
		t.Fatalf("(%q, %T): %v", name, value, err)
	}

	if err != nil {
		return
	}

	if !bytes.Equal(ss.buf, exp) {
		t.Fatal("buf does not match")
	}
}

func TestToken(t *testing.T) {
	s, err := newScannerSource(Source{"test", `abc
def
 ghi
`, nil})
	// abc\ndef\n ghi\n
	//             1
	// 0123 4567 89012
	if err != nil {
		t.Fatal(err)
	}

	s.file.AddLine(4)
	s.file.AddLine(8)
	s.file.AddLine(13)
	for itest, test := range []struct {
		Token
		line int
		col  int
		ch   rune
		sep  string
		src  string
	}{
		{newToken(s, 0, 0, 0, 3), 1, 1, 0, "", "abc"},
		{newToken(s, 1, 3, 4, 3), 2, 1, 1, "\n", "def"},
		{newToken(s, 2, 7, 9, 3), 3, 2, 2, "\n ", "ghi"},
		{newToken(s, eof, 13, 13, 0), 3, 6, eof, "", ""},
	} {
		tok := test.Token
		if g, e := tok.Position().Line, test.line; g != e {
			t.Fatal(itest, g, e)
		}
		if g, e := tok.Position().Column, test.col; g != e {
			t.Fatal(itest, g, e)
		}
		if g, e := tok.Ch, test.ch; g != e {
			t.Fatal(itest, g, e)
		}
		if g, e := string(tok.Sep()), test.sep; g != e {
			t.Fatalf("%v %q %q", itest, g, e)
		}
		if g, e := string(tok.Src()), test.src; g != e {
			t.Fatalf("%v %q %q", itest, g, e)
		}

		tok2 := tok
		tok2.Set([]byte("xyz0123"), []byte("456789"))
		if g, e := string(tok.Sep()), test.sep; g != e {
			t.Fatalf("%v %q %q", itest, g, e)
		}
		if g, e := string(tok.Src()), test.src; g != e {
			t.Fatalf("%v %q %q", itest, g, e)
		}
		if g, e := string(tok2.Sep()), "xyz0123"; g != e {
			t.Fatalf("%v %q %q", itest, g, e)
		}
		if g, e := string(tok2.Src()), "456789"; g != e {
			t.Fatalf("%v %q %q", itest, g, e)
		}
	}
}

type parallel struct {
	limit chan struct{}
	sync.Mutex
	wg sync.WaitGroup
}

func newParallel() *parallel {
	return &parallel{
		limit: make(chan struct{}, runtime.GOMAXPROCS(0)),
	}
}

func (p *parallel) exec(run func()) {
	p.limit <- struct{}{}
	p.wg.Add(1)

	go func() {
		defer func() {
			p.wg.Done()
			<-p.limit
		}()

		run()
	}()
}

var tokSink []Token

func TestScanner(t *testing.T) {
	defer func() { tokSink = nil }()

	var files, tokens, chars int64
	var m0, m runtime.MemStats
	p := newParallel()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m0)
	for _, path := range corpusIndex {
		path := path
		switch filepath.Ext(path) {
		case ".c", ".h":
			files++
			p.exec(func() {
				var err error
				var chars0, tokens0 int64
				var toks []Token

				defer func() {
					p.Lock()
					chars += chars0
					tokens += tokens0
					tokSink = append(tokSink, toks...)
					if err != nil {
						t.Error(err)
					}
					p.Unlock()
				}()

				buf := corpus[path]
				chars0 += int64(len(buf))
				var s *scanner
				if s, err = newScanner(Source{path, buf, nil}, func(msg string, args ...interface{}) {
					s.close()
					err = fmt.Errorf(msg, args...)
				}); err != nil {
					err = fmt.Errorf("%v: %v", path, err)
					return
				}

				for {
					tok := s.cppScan()
					if tok.Ch == eof {
						return
					}

					toks = append(toks, tok)
					tokens0++
				}
			})
		}
	}
	p.wg.Wait()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m)
	t.Logf("files %v; tokens %v; bytes %v; heap %v; alloc %v", h(files), h(tokens), h(chars), h(m.HeapAlloc-m0.HeapAlloc), h(m.TotalAlloc-m0.TotalAlloc))
}

func h(v interface{}) string {
	switch x := v.(type) {
	case int64:
		return humanize.Comma(x)
	case uint64:
		if x <= math.MaxInt64 {
			return humanize.Comma(int64(x))
		}
	}
	return fmt.Sprint(v)
}

func BenchmarkScanner(b *testing.B) {
	debug.FreeOSMemory()
	for i := 0; i < b.N; i++ {
		var chars int64
		for _, path := range corpusIndex {
			switch filepath.Ext(path) {
			case ".c", ".h":
				buf := corpus[path]
				chars += int64(len(buf))
				var s *scanner
				var err error
				if s, err = newScanner(Source{path, buf, nil}, func(msg string, args ...interface{}) {
					s.close()
					b.Fatalf(msg, args...)
				}); err != nil {
					b.Fatal(path, err)
				}
				for {
					tok := s.cppScan()
					if tok.Ch == eof {
						break
					}
				}
			}
		}
		b.SetBytes(chars)
	}
}

var (
	cppParseBlacklist = map[string]struct{}{
		"/github.com/vnmakarov/mir/c-tests/new/endif.c": {}, // 1:1: unexpected #endif
	}
	astSink []group
)

func TestCPPParse0(t *testing.T) {
	defer func() { astSink = nil }()

	var files, lines, chars int64
	var m0, m runtime.MemStats
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m0)
	for _, path := range corpusIndex {
		if _, ok := cppParseBlacklist[path]; ok {
			continue
		}

		switch filepath.Ext(path) {
		case ".c", ".h":
			buf := corpus[path]
			chars += int64(len(buf))
			var p *cppParser
			var err error
			if p, err = newCppParser(Source{path, buf, nil}, func(msg string, args ...interface{}) {
				p.close()
				t.Fatalf(msg, args...)
			}); err != nil {
				t.Fatal(path, err)
			}

			files++
			ast := p.preprocessingFile()
			if len(ast) == 0 {
				t.Fatalf("%v: empty AST", path)
			}

			eol := ast[len(ast)-1]
			x, ok := eol.(eofLine)
			if !ok {
				t.Fatalf("%v: AST not terminated: %T", p.pos(), eol)
			}

			eof := Token(x)
			lines += int64(eof.Position().Line)
			astSink = append(astSink, ast)
		}
	}
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m)
	astSink = nil
	t.Logf("files %v; lines %v bytes %v; heap %v; alloc %v", h(files), h(lines), h(chars), h(m.HeapAlloc-m0.HeapAlloc), h(m.TotalAlloc-m0.TotalAlloc))
}

func TestCPPParse(t *testing.T) {
	defer func() { astSink = nil }()

	var files, lines, chars int64
	var m0, m runtime.MemStats
	p := newParallel()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m0)
	for _, path := range corpusIndex {
		path := path
		if _, ok := cppParseBlacklist[path]; ok {
			continue
		}

		switch filepath.Ext(path) {
		case ".c", ".h":
			files++
			p.exec(func() {
				buf := corpus[path]
				var err error
				var ast group
				var eof Token

				defer func() {
					p.Lock()
					chars += int64(len(buf))
					lines += int64(eof.Position().Line)
					astSink = append(astSink, ast)
					if err != nil {
						t.Error(err)
					}
					p.Unlock()
				}()

				var p *cppParser
				if p, err = newCppParser(Source{path, buf, nil}, func(msg string, args ...interface{}) {
					p.close()
					err = fmt.Errorf(msg, args...)
				}); err != nil {
					t.Fatal(path, err)
				}

				if ast = p.preprocessingFile(); len(ast) == 0 {
					t.Fatalf("%v: empty AST", path)
				}

				eol := ast[len(ast)-1]
				x, ok := eol.(eofLine)
				if !ok {
					err = fmt.Errorf("%v: AST not terminated: %T", p.pos(), eol)
					return
				}

				eof = Token(x)
			})
		}
	}
	p.wg.Wait()
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m)
	astSink = nil
	t.Logf("files %v; lines %v bytes %v; heap %v; alloc %v", h(files), h(lines), h(chars), h(m.HeapAlloc-m0.HeapAlloc), h(m.TotalAlloc-m0.TotalAlloc))
}

func BenchmarkCPPParse(b *testing.B) {
	debug.FreeOSMemory()
	for i := 0; i < b.N; i++ {
		var chars int64
		for _, path := range corpusIndex {
			if _, ok := cppParseBlacklist[path]; ok {
				continue
			}

			switch filepath.Ext(path) {
			case ".c", ".h":
				buf := corpus[path]
				chars += int64(len(buf))
				var p *cppParser
				var err error
				if p, err = newCppParser(Source{path, buf, nil}, func(msg string, args ...interface{}) {
					p.close()
					b.Fatalf(msg, args...)
				}); err != nil {
					b.Fatal(path, err)
				}

				ast := p.preprocessingFile()
				if len(ast) == 0 {
					b.Fatalf("%v: empty AST", path)
				}

				eol := ast[len(ast)-1]
				if _, ok := eol.(eofLine); !ok {
					b.Fatalf("%v: AST not terminated: %T", p.pos(), eol)
				}
			}
		}
		b.SetBytes(chars)
	}
}

func testCfg() *Config {
	c := *testCfg0
	return &c
}

func TestCPPExpand(t *testing.T) {
	testCPPExpand(t, "testdata/cpp-expand/", nil, true)
}

func testCPPExpand(t *testing.T, dir string, blacklist map[string]struct{}, fakeIncludes bool) {
	var fails []string
	var files, ok, skip int
	var c *cpp
	cfg := testCfg()
	cfg.fakeIncludes = fakeIncludes
	cfg.PragmaHandler = func(s []Token) error {
		a := textLine{pragmaTestTok}
		for i, v := range s {
			if i == 0 {
				v.Set(sp, v.Src())
			}
			a = append(a, v)
		}
		c.push(append(a, nlTok))
		return nil
	}
	err := filepath.Walk(filepath.FromSlash(dir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || (!strings.HasSuffix(path, ".c") && !strings.HasSuffix(path, ".h")) {
			return nil
		}

		files++
		switch {
		case re != nil:
			if !re.MatchString(path) {
				skip++
				return nil
			}
		default:
			if _, ok := blacklist[filepath.Base(path)]; ok {
				skip++
				return nil
			}
		}

		if *oTrace {
			fmt.Fprintln(os.Stderr, path)
		}
		var b strings.Builder
		if c, err = newCPP(cfg, []Source{{path, nil, nil}}, nil); err != nil {
			t.Fatalf("%v: %v", path, err)
		}

		if err := preprocess(c, &b); err != nil {
			fails = append(fails, path)
			t.Fatalf("%v: %v", path, err)
		}

		if strings.Contains(filepath.ToSlash(path), "/mustfail/") {
			if err != nil {
				return nil
			}

			fails = append(fails, path)
			return fmt.Errorf("%v: unexpected success", path)
		}

		if err != nil {
			fails = append(fails, path)
			return err
		}

		expFn := path + ".expect"
		exp, err := os.ReadFile(expFn)
		if err != nil {
			fails = append(fails, path)
			t.Error(err)
		}

		g := strings.ReplaceAll(b.String(), "\r", "")
		g = strings.TrimSpace(g)
		e := strings.ReplaceAll(string(exp), "\r", "")
		e = strings.TrimSpace(e)
		if g != e {
			fails = append(fails, path)
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(e),
				B:        difflib.SplitLines(g),
				FromFile: expFn,
				ToFile:   path,
				Context:  0,
			}
			s, err := difflib.GetUnifiedDiffString(diff)
			if err != nil {
				t.Fatalf("%v: %v", path, err)
			}

			t.Errorf("%v\ngot\n%s\nexp\n%s\ngot\n%s\nexp\n%s", s, g, e, hex.Dump([]byte(g)), hex.Dump([]byte(e)))
			return nil
		}
		ok++
		return nil
	})
	for _, v := range fails {
		t.Log(v)
	}
	t.Logf("files %v, skip %v, ok %v, fails %v", files, skip, ok, len(fails))
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreprocess(t *testing.T) {
	testCPPExpand(t, "testdata/preprocess/", nil, true)
}

func TestTCCExpand(t *testing.T) {
	testCPPExpand(t, "testdata/tcc-0.9.27/tests/pp/", map[string]struct{}{
		"11.c": {}, // https://gcc.gnu.org/onlinedocs/gcc/Variadic-Macros.html#Variadic-Macros
		"16.c": {}, // We don't produce warnings on macro redefinition.
	}, true)
}

func TestInclude(t *testing.T) {
	testCPPExpand(t, "testdata/include/", nil, false)
}

func TestTranslationPhase4(t *testing.T) {
	cfgGame := testCfg()
	cfgGame.FS = cFS
	cfgGame.SysIncludePaths = append(
		cfgGame.SysIncludePaths,
		"/Library/Developer/CommandLineTools/SDKs/MacOSX11.1.sdk/usr/include/libxml",
		"/Library/Developer/CommandLineTools/SDKs/MacOSX11.1.sdk/usr/include/malloc",
		"/Library/Developer/CommandLineTools/SDKs/MacOSX12.1.sdk/usr/include/libxml",
		"/Library/Developer/CommandLineTools/SDKs/MacOSX12.1.sdk/usr/include/malloc",
		"/benchmarksgame-team.pages.debian.net/Include",
		"/opt/homebrew/include",
		"/usr/include/sys",
		"/usr/lib/clang/11.1.0/include",
		"/usr/local/Cellar/gcc/11.2.0_1/lib/gcc/11/gcc/x86_64-apple-darwin19/11.2.0/include",
		"/usr/local/include",
	)
	cfgGame.IncludePaths = append(
		cfgGame.IncludePaths,
		"/opt/homebrew/include",
		"/usr/local/include",
	)
	cfg := testCfg()
	cfg.FS = cFS
	var blacklistCompCert, blacklistCxgo map[string]struct{}
	blacklistGame := map[string]struct{}{
		// Missing <apr_pools.h>
		"binary-trees-2.c": {},
		"binary-trees-3.c": {},

		"binary-trees-5.c":       {}, //TODO
		"fannkuchredux-2.c":      {}, //TODO
		"fannkuchredux-3.c":      {}, //TODO
		"fannkuchredux-4.c":      {}, //TODO
		"fasta-4.c":              {}, //TODO
		"fasta.c":                {}, //TODO
		"k-nucleotide.c":         {}, //TODO
		"mandelbrot-3.c":         {}, //TODO
		"mandelbrot-4.c":         {}, //TODO
		"mandelbrot-6.c":         {}, //TODO
		"mandelbrot-7.c":         {}, //TODO
		"mandelbrot.c":           {}, //TODO
		"nbody-4.c":              {}, //TODO
		"nbody-8.c":              {}, //TODO
		"nbody-9.c":              {}, //TODO
		"pidigits-2.c":           {}, //TODO
		"pidigits-6.c":           {}, //TODO
		"pidigits.c":             {}, //TODO
		"regex-redux-3.c":        {}, //TODO
		"regex-redux-4.c":        {}, //TODO
		"regex-redux-5.c":        {}, //TODO
		"reverse-complement-2.c": {}, //TODO
		"reverse-complement-4.c": {}, //TODO
		"reverse-complement-5.c": {}, //TODO
		"spectral-norm-4.c":      {}, //TODO
		"spectral-norm-5.c":      {}, //TODO
		"spectral-norm-6.c":      {}, //TODO
	}
	blacklistGCC := map[string]struct{}{
		// assertions are deprecated.
		"950919-1.c": {},

		// Need include files not in ccorpus.
		"pr88347.c": {},
		"pr88423.c": {},
	}
	blacklistVNMakarov := map[string]struct{}{
		// #endif without #if
		"endif.c": {},
	}
	blacklictTCC := map[string]struct{}{
		"11.c": {}, // https://gcc.gnu.org/onlinedocs/gcc/Variadic-Macros.html#Variadic-Macros
	}
	switch fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH) {
	case "linux/s390x":
		blacklistCompCert = map[string]struct{}{"aes.c": {}} // Unsupported endianness.
		fallthrough
	case "linux/arm", "linux/arm64":
		// Uses sse2 headers.
		blacklistGame["fannkuchredux-4.c"] = struct{}{}
		blacklistGame["mandelbrot-6.c"] = struct{}{}
		blacklistGame["nbody-4.c"] = struct{}{}
		blacklistGame["nbody-8.c"] = struct{}{}
		blacklistGame["nbody-9.c"] = struct{}{}
		blacklistGame["spectral-norm-5.c"] = struct{}{}
		blacklistGame["spectral-norm-6.c"] = struct{}{}
	case "freebsd/386", "darwin/amd64", "darwin/arm64", "freebsd/amd64", "linux/386":
		blacklistCompCert = map[string]struct{}{"aes.c": {}} // include file not found: "../endian.h"
	case "windows/amd64", "windows/386":
		blacklistCompCert = map[string]struct{}{"aes.c": {}} // include file not found: "../endian.h"
		blacklistCxgo = map[string]struct{}{"inet.c": {}}    // include file not found: <arpa/inet.h>
		blacklistGCC["loop-2f.c"] = struct{}{}               // include file not found: <sys/mman.h>
		blacklistGCC["loop-2g.c"] = struct{}{}               // include file not found: <sys/mman.h>
		blacklistGame["fasta-4.c"] = struct{}{}              // include file not found: <err.h>
		blacklistGame["pidigits-2.c"] = struct{}{}           // include file not found: <gmp.h>
		blacklistGame["pidigits-6.c"] = struct{}{}           // include file not found: <threads.h>
		blacklistGame["pidigits-9.c"] = struct{}{}           // include file not found: <gmp.h>
		blacklistGame["pidigits.c"] = struct{}{}             // include file not found: <gmp.h>
		blacklistGame["regex-redux-2.c"] = struct{}{}        // include file not found: <pcre.h>
		blacklistGame["regex-redux-3.c"] = struct{}{}        // include file not found: <pcre.h>
		blacklistGame["regex-redux-4.c"] = struct{}{}        // include file not found: <pcre.h>
		blacklistGame["regex-redux-5.c"] = struct{}{}        // include file not found: <pcre2.h>
	case "openbsd/amd64":
		blacklistCompCert = map[string]struct{}{"aes.c": {}} // include file not found: "../endian.h"
		blacklistGame["mandelbrot-7.c"] = struct{}{}         // include file not found: <omp.h>
		blacklistGame["pidigits-6.c"] = struct{}{}           // include file not found: <threads.h>
		blacklistGame["regex-redux-3.c"] = struct{}{}        // include file not found: <omp.h>
		blacklistGame["spectral-norm-4.c"] = struct{}{}      // include file not found: <omp.h>
	}
	var files, ok, skip, fails int
	for _, v := range []struct {
		cfg       *Config
		dir       string
		blacklist map[string]struct{}
	}{
		{cfg, "CompCert-3.6/test/c", blacklistCompCert},
		{cfg, "ccgo", nil},
		{cfg, "gcc-9.1.0/gcc/testsuite/gcc.c-torture", blacklistGCC},
		{cfg, "github.com/AbsInt/CompCert/test/c", blacklistCompCert},
		{cfg, "github.com/cxgo", blacklistCxgo},
		{cfg, "github.com/gcc-mirror/gcc/gcc/testsuite", blacklistGCC},
		{cfg, "github.com/vnmakarov", blacklistVNMakarov},
		{cfg, "sqlite-amalgamation-3370200", nil},
		{cfg, "tcc-0.9.27/tests", blacklictTCC},
		{cfgGame, "benchmarksgame-team.pages.debian.net", blacklistGame},
	} {
		t.Run(v.dir, func(t *testing.T) {
			f, o, s, n := testTranslationPhase4(t, v.cfg, "/"+v.dir, v.blacklist)
			files += f
			ok += o
			skip += s
			fails += n
		})
	}
	t.Logf("TOTAL: files %v, skip %v, ok %v, fails %v", files, skip, ok, fails)
}

func testTranslationPhase4(t *testing.T, cfg *Config, dir string, blacklist map[string]struct{}) (files, ok, skip, nfails int) {
	var fails []string
	p := newParallel()
	err := walk(dir, func(pth string, fi os.FileInfo) error {
		if fi.IsDir() {
			return nil
		}

		if filepath.Ext(pth) != ".c" {
			return nil
		}

		switch {
		case re != nil:
			if !re.MatchString(pth) {
				skip++
				return nil
			}
		default:
			if _, ok := blacklist[filepath.Base(pth)]; ok {
				skip++
				return nil
			}
		}

		files++
		p.exec(func() {
			if *oTrace {
				fmt.Fprintln(os.Stderr, pth)
			}
			err := Preprocess(
				cfg,
				[]Source{
					{Name: "<predefined>", Value: predefined},
					{Name: "<builtin>", Value: builtin},
					{Name: pth, FS: cFS},
				},
				io.Discard,
			)
			p.Lock()

			defer p.Unlock()

			if err != nil {
				fails = append(fails, pth)
				t.Errorf("%v: %v", pth, err)
			} else {
				ok++
			}
		})
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	p.wg.Wait()
	for _, v := range fails {
		t.Log(v)
	}
	t.Logf("files %v, skip %v, ok %v, fails %v", files, skip, ok, len(fails))
	return files, ok, skip, len(fails)
}

// https://gitlab.com/cznic/cc/-/issues/127
func TestIssue127(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	if err := os.Chdir(filepath.FromSlash("testdata/issue127/")); err != nil {
		t.Error(err)
		return
	}

	cd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("working directory: %s", cd)
	cfg := testCfg()
	cfg.IncludePaths = append(cfg.IncludePaths, "include")
	if err := Preprocess(
		cfg,
		[]Source{{Name: "main.c"}},
		io.Discard,
	); err != nil {
		t.Fatalf("%v", err)
	}
}

func TestBOM(t *testing.T) {
	for i, v := range []struct {
		src string
		err string
	}{
		{"int main() {}", ""},
		{"\xEF\xBB\xBFint main() {}", ""},
	} {
		switch _, err := Parse(testCfg(), []Source{{Value: v.src}}); {
		case v.err == "" && err != nil:
			t.Errorf("%v: unexpected error %v", i, err)
		case v.err != "" && err == nil:
			t.Errorf("%v: unexpected success, expected error matching %v", i, v.err)
		case v.err != "":
			if !regexp.MustCompile(v.err).MatchString(err.Error()) {
				t.Errorf("%v: error %v does not match %v", i, err, v.err)
			}
		}
	}
}

func TestStrCatSep(t *testing.T) {
	for i, v := range []struct {
		src         string
		lit         string
		sep         string
		trailingSep string
	}{
		{`int f() {  "a";}`, `"a"`, "  ", "\n"},
		{`int f() {  L"a";}`, `L"a"`, "  ", "\n"},
		{`int f() { "a" "b";}`, `"ab"`, "  ", "\n"},
		{`int f() { "a""b";}`, `"ab"`, " ", "\n"},
		{`int f() { "a";}`, `"a"`, " ", "\n"},
		{`int f() { "a"` + "\n\t" + `"b"; }`, `"ab"`, " \n\t", "\n"},
		{`int f() { /*x*/ /*y*/ "a";}`, `"a"`, " /*x*/ /*y*/ ", "\n"},
		{`int f() { /*x*/` + "\n" + `/*y*/ "a";}`, `"a"`, " /*x*/\n/*y*/ ", "\n"},
		{`int f() { //x` + "\n" + ` "a";}`, `"a"`, " //x\n ", "\n"},
		{`int f() { //x` + "\n" + `"a";}`, `"a"`, " //x\n", "\n"},
		{`int f() { L"a" L"b";}`, `L"ab"`, "  ", "\n"},
		{`int f() { ` + "\n" + ` "a";}`, `"a"`, " \n ", "\n"},
		{`int f() { ` + "\n" + `"a";}`, `"a"`, " \n", "\n"},
		{`int f() {"a" "b";}`, `"ab"`, " ", "\n"},
		{`int f() {"a"/*y*/"b";}`, `"ab"`, "/*y*/", "\n"},
		{`int f() {"a";} /*x*/ `, `"a"`, "", " /*x*/ \n"},
		{`int f() {"a";} /*x*/`, `"a"`, "", " /*x*/\n"},
		{`int f() {"a";} /*x` + "\n" + `*/ `, `"a"`, "", " /*x\n*/ \n"},
		{`int f() {"a";} `, `"a"`, "", " \n"},
		{`int f() {"a";}/*x*/`, `"a"`, "", "/*x*/\n"},
		{`int f() {"a";}` + "\n", `"a"`, "", "\n"},
		{`int f() {"a";}`, `"a"`, "", "\n"},
		{`int f() {/*x*/ /*y*/ "a";}`, `"a"`, "/*x*/ /*y*/ ", "\n"},
		{`int f() {/*x*/"a""b";}`, `"ab"`, "/*x*/", "\n"},
		{`int f() {/*x*/"a"/*y*/"b";}`, `"ab"`, "/*x*//*y*/", "\n"},
		{`int f() {/*x*/"a";}`, `"a"`, "/*x*/", "\n"},
		{`int f() {/*x*//*y*/ "a";}`, `"a"`, "/*x*//*y*/ ", "\n"},
		{`int f() {/*x*//*y*/"a";}`, `"a"`, "/*x*//*y*/", "\n"},
		{`int f() {//` + "\n" + `"a";}`, `"a"`, "//\n", "\n"},
		{`int f() {//x` + "\n" + `"a";}`, `"a"`, "//x\n", "\n"},
		{`int f() {` + "\n" + ` "a";}`, `"a"`, "\n ", "\n"},
		{`int f() {` + "\n" + `"a";}`, `"a"`, "\n", "\n"},
	} {
		ast, err := Parse(&Config{}, []Source{{Name: "test", Value: v.src}})
		if err != nil {
			t.Errorf("%v: %v", i, err)
			continue
		}

		var n Node
		depth := mathutil.MaxInt
		findNode("PrimaryExpression", ast.TranslationUnit, 0, &n, &depth)
		tok := n.(*PrimaryExpression).Token
		if g, e := string(tok.Src()), v.lit; g != e {
			t.Errorf("%v: %q %q", i, g, e)
		}
		if g, e := string(tok.Sep()), v.sep; g != e {
			t.Errorf("%v: %q %q", i, g, e)
		}
		if g, e := string(ast.EOF.Sep()), v.trailingSep; g != e {
			t.Errorf("%v: %q %q", i, g, e)
		}
	}
}

func TestParserBug(t *testing.T) {
	cfg := testCfg()
	var fails []string
	var files, ok, skip int
	err := filepath.Walk(filepath.FromSlash("testdata/parser/bug"), func(pth string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		if filepath.Ext(pth) != ".c" {
			return nil
		}

		switch {
		case re != nil:
			if !re.MatchString(pth) {
				skip++
				return nil
			}
		}

		files++
		if *oTrace {
			fmt.Fprintln(os.Stderr, pth)
		}
		var ast *AST
		ast, err = Parse(
			cfg,
			[]Source{
				{Name: "<predefined>", Value: predefined},
				{Name: "<builtin>", Value: builtin},
				{Name: pth, FS: cFS},
			},
		)
		_ = ast //TODO-
		if err != nil {
			fails = append(fails, pth)
			t.Errorf("%v: %v", pth, err)
		} else {
			ok++
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	for _, v := range fails {
		t.Log(v)
	}
	t.Logf("files %v, skip %v, ok %v, fails %v", files, skip, ok, len(fails))
}

func TestParse(t *testing.T) {
	cfgGame := testCfg()
	cfgGame.FS = cFS
	cfgGame.SysIncludePaths = append(
		cfgGame.SysIncludePaths,
		"/Library/Developer/CommandLineTools/SDKs/MacOSX11.1.sdk/usr/include/libxml",
		"/Library/Developer/CommandLineTools/SDKs/MacOSX11.1.sdk/usr/include/malloc",
		"/Library/Developer/CommandLineTools/SDKs/MacOSX12.1.sdk/usr/include/libxml",
		"/Library/Developer/CommandLineTools/SDKs/MacOSX12.1.sdk/usr/include/malloc",
		"/benchmarksgame-team.pages.debian.net/Include",
		"/opt/homebrew/include",
		"/usr/include/sys",
		"/usr/lib/clang/11.1.0/include",
		"/usr/local/Cellar/gcc/11.2.0_1/lib/gcc/11/gcc/x86_64-apple-darwin19/11.2.0/include",
		"/usr/local/include",
	)
	cfgGame.IncludePaths = append(
		cfgGame.IncludePaths,
		"/opt/homebrew/include",
		"/usr/local/include",
	)
	cfg := testCfg()
	cfg.FS = cFS
	blacklistGame := map[string]struct{}{
		// Missing <apr_pools.h>
		"binary-trees-2.c": {},
		"binary-trees-3.c": {},

		"k-nucleotide.c":    {}, //TODO
		"mandelbrot-3.c":    {}, //TODO
		"mandelbrot-4.c":    {}, //TODO
		"mandelbrot-6.c":    {}, //TODO
		"mandelbrot-7.c":    {}, //TODO
		"mandelbrot.c":      {}, //TODO
		"nbody-4.c":         {}, //TODO
		"nbody-8.c":         {}, //TODO
		"nbody-9.c":         {}, //TODO
		"pidigits-2.c":      {}, //TODO
		"pidigits-6.c":      {}, //TODO
		"pidigits.c":        {}, //TODO
		"regex-redux-4.c":   {}, //TODO
		"regex-redux-5.c":   {}, //TODO
		"spectral-norm-6.c": {}, //TODO
	}
	blacklistGCC := map[string]struct{}{
		// assertions are deprecated.
		"950919-1.c": {},

		// Need include files not in ccorpus.
		"pr88347.c": {},
		"pr88423.c": {},

		//TODO What's the type of bar?
		"20010313-1.c": {},

		"20000205-1.c":                 {}, //TODO
		"20000605-3.c":                 {}, //TODO
		"20000914-1.c":                 {}, //TODO
		"20000917-1.c":                 {}, //TODO
		"20010114-1.c":                 {}, //TODO
		"20010118-1.c":                 {}, //TODO
		"20010124-1.c":                 {}, //TODO
		"20010325-1.c":                 {}, //TODO
		"20010328-1.c":                 {}, //TODO
		"20010518-1.c":                 {}, //TODO
		"20010605-1.c":                 {}, //TODO
		"20010605-2.c":                 {}, //TODO
		"20010701-1.c":                 {}, //TODO
		"20010903-1.c":                 {}, //TODO
		"20010903-2.c":                 {}, //TODO
		"20010911-1.c":                 {}, //TODO
		"20011113-1.c":                 {}, //TODO
		"20020303-1.c":                 {}, //TODO
		"20020411-1.c":                 {}, //TODO
		"20020412-1.c":                 {}, //TODO
		"20020709-1.c":                 {}, //TODO
		"20030903-1.c":                 {}, //TODO
		"20030910-1.c":                 {}, //TODO
		"20031112-1.c":                 {}, //TODO
		"20040101-1.c":                 {}, //TODO
		"20040302-1.c":                 {}, //TODO
		"20040614-1.c":                 {}, //TODO
		"20041214-1.c":                 {}, //TODO
		"20050121-1.c":                 {}, //TODO
		"20050510-1.c":                 {}, //TODO
		"20051216-1.c":                 {}, //TODO
		"20060609-1.c":                 {}, //TODO
		"20070603-1.c":                 {}, //TODO
		"20070623-1.c":                 {}, //TODO
		"20071029-1.c":                 {}, //TODO
		"20071202-1.c":                 {}, //TODO
		"20071210-1.c":                 {}, //TODO
		"20071220-1.c":                 {}, //TODO
		"20071220-2.c":                 {}, //TODO
		"20081108-1.c":                 {}, //TODO
		"20100609-1.c":                 {}, //TODO
		"20101216-1.c":                 {}, //TODO
		"20110131-1.c":                 {}, //TODO
		"20111208-1.c":                 {}, //TODO
		"20121107-1.c":                 {}, //TODO
		"20190820-1.c":                 {}, //TODO
		"920301-1.c":                   {}, //TODO
		"920302-1.c":                   {}, //TODO
		"920415-1.c":                   {}, //TODO
		"920428-2.c":                   {}, //TODO
		"920428-3.c":                   {}, //TODO
		"920501-1.c":                   {}, //TODO
		"920501-3.c":                   {}, //TODO
		"920501-4.c":                   {}, //TODO
		"920501-5.c":                   {}, //TODO
		"920501-7.c":                   {}, //TODO
		"920502-1.c":                   {}, //TODO
		"920502-2.c":                   {}, //TODO
		"920721-3.c":                   {}, //TODO
		"920721-4.c":                   {}, //TODO
		"920826-1.c":                   {}, //TODO
		"920831-1.c":                   {}, //TODO
		"920928-5.c":                   {}, //TODO
		"921012-1.c":                   {}, //TODO
		"921215-1.c":                   {}, //TODO
		"930510-1.c":                   {}, //TODO
		"930513-1.c":                   {}, //TODO
		"930926-1.c":                   {}, //TODO
		"940718-1.c":                   {}, //TODO
		"950613-1.c":                   {}, //TODO
		"961031-1.c":                   {}, //TODO
		"980505-1.c":                   {}, //TODO
		"980526-1.c":                   {}, //TODO
		"980618-1.c":                   {}, //TODO
		"980929-1.c":                   {}, //TODO
		"981001-2.c":                   {}, //TODO
		"981001-4.c":                   {}, //TODO
		"981223-1.c":                   {}, //TODO
		"990326-1.c":                   {}, //TODO
		"990413-2.c":                   {}, //TODO
		"990928-1.c":                   {}, //TODO
		"991026-2.c":                   {}, //TODO
		"991213-1.c":                   {}, //TODO
		"991213-2.c":                   {}, //TODO
		"991213-3.c":                   {}, //TODO
		"DFcmp.c":                      {}, //TODO
		"HIcmp.c":                      {}, //TODO
		"HIset.c":                      {}, //TODO
		"SFset.c":                      {}, //TODO
		"SIcmp.c":                      {}, //TODO
		"SIset.c":                      {}, //TODO
		"UHIcmp.c":                     {}, //TODO
		"USIcmp.c":                     {}, //TODO
		"align-3.c":                    {}, //TODO
		"alloca-1.c":                   {}, //TODO
		"arith-1.c":                    {}, //TODO
		"band.c":                       {}, //TODO
		"builtin-prefetch-4.c":         {}, //TODO
		"builtin-types-compatible-p.c": {}, //TODO
		"builtin_constant_p.c":         {}, //TODO
		"callind.c":                    {}, //TODO
		"comp-goto-1.c":                {}, //TODO
		"comp-goto-2.c":                {}, //TODO
		"complex-1-lib.c":              {}, //TODO
		"complex-1.c":                  {}, //TODO
		"complex-2.c":                  {}, //TODO
		"complex-3.c":                  {}, //TODO
		"complex-4.c":                  {}, //TODO
		"compound-literal-3.c":         {}, //TODO
		"conv.c":                       {}, //TODO
		"conversion.c":                 {}, //TODO
		"copysign1.c":                  {}, //TODO
		"cp.c":                         {}, //TODO
		"dll.c":                        {}, //TODO
		"enum-3.c":                     {}, //TODO
		"ffs-1.c":                      {}, //TODO
		"fp-cmp-4.c":                   {}, //TODO
		"fp-cmp-4f.c":                  {}, //TODO
		"fp-cmp-4l.c":                  {}, //TODO
		"fp-cmp-5.c":                   {}, //TODO
		"fp-cmp-8.c":                   {}, //TODO
		"fp-cmp-8f.c":                  {}, //TODO
		"fp-cmp-8l.c":                  {}, //TODO
		"fputs.c":                      {}, //TODO
		"labels-2.c":                   {}, //TODO
		"labels-3.c":                   {}, //TODO
		"mzero2.c":                     {}, //TODO
		"nest-align-1.c":               {}, //TODO
		"packed-aligned.c":             {}, //TODO
		"pc44485.c":                    {}, //TODO
		"pr17397.c":                    {}, //TODO
		"pr17529.c":                    {}, //TODO
		"pr17913.c":                    {}, //TODO
		"pr18903.c":                    {}, //TODO
		"pr19449.c":                    {}, //TODO
		"pr21356.c":                    {}, //TODO
		"pr22061-3.c":                  {}, //TODO
		"pr22098-1.c":                  {}, //TODO
		"pr22098-2.c":                  {}, //TODO
		"pr22098-3.c":                  {}, //TODO
		"pr22422.c":                    {}, //TODO
		"pr24135.c":                    {}, //TODO
		"pr25224.c":                    {}, //TODO
		"pr25860.c":                    {}, //TODO
		"pr26781-1.c":                  {}, //TODO
		"pr27341-1.c":                  {}, //TODO
		"pr27863.c":                    {}, //TODO
		"pr28489.c":                    {}, //TODO
		"pr29006.c":                    {}, //TODO
		"pr29128.c":                    {}, //TODO
		"pr30984.c":                    {}, //TODO
		"pr32139.c":                    {}, //TODO
		"pr32482.c":                    {}, //TODO
		"pr32584.c":                    {}, //TODO
		"pr32919.c":                    {}, //TODO
		"pr33173.c":                    {}, //TODO
		"pr33382.c":                    {}, //TODO
		"pr34154.c":                    {}, //TODO
		"pr34334.c":                    {}, //TODO
		"pr34648.c":                    {}, //TODO
		"pr34768-1.c":                  {}, //TODO
		"pr34768-2.c":                  {}, //TODO
		"pr34885.c":                    {}, //TODO
		"pr34966.c":                    {}, //TODO
		"pr34993.c":                    {}, //TODO
		"pr36666.c":                    {}, //TODO
		"pr37026.c":                    {}, //TODO
		"pr37207.c":                    {}, //TODO
		"pr37381.c":                    {}, //TODO
		"pr37418-3.c":                  {}, //TODO
		"pr37418-4.c":                  {}, //TODO
		"pr37669.c":                    {}, //TODO
		"pr38016.c":                    {}, //TODO
		"pr38554.c":                    {}, //TODO
		"pr38771.c":                    {}, //TODO
		"pr38969.c":                    {}, //TODO
		"pr39120.c":                    {}, //TODO
		"pr39845.c":                    {}, //TODO
		"pr39937.c":                    {}, //TODO
		"pr40233.c":                    {}, //TODO
		"pr40657.c":                    {}, //TODO
		"pr42164.c":                    {}, //TODO
		"pr42196-1.c":                  {}, //TODO
		"pr42196-2.c":                  {}, //TODO
		"pr42196-3.c":                  {}, //TODO
		"pr42398.c":                    {}, //TODO
		"pr42691.c":                    {}, //TODO
		"pr42716.c":                    {}, //TODO
		"pr42717.c":                    {}, //TODO
		"pr43255.c":                    {}, //TODO
		"pr43661.c":                    {}, //TODO
		"pr43679.c":                    {}, //TODO
		"pr43845.c":                    {}, //TODO
		"pr44119.c":                    {}, //TODO
		"pr44197.c":                    {}, //TODO
		"pr44784.c":                    {}, //TODO
		"pr45109.c":                    {}, //TODO
		"pr46107.c":                    {}, //TODO
		"pr46866.c":                    {}, //TODO
		"pr47428.c":                    {}, //TODO
		"pr48717.c":                    {}, //TODO
		"pr49049.c":                    {}, //TODO
		"pr49218.c":                    {}, //TODO
		"pr49474.c":                    {}, //TODO
		"pr51495.c":                    {}, //TODO
		"pr51694.c":                    {}, //TODO
		"pr53519.c":                    {}, //TODO
		"pr53645.c":                    {}, //TODO
		"pr53748.c":                    {}, //TODO
		"pr54103-1.c":                  {}, //TODO
		"pr54103-2.c":                  {}, //TODO
		"pr54103-3.c":                  {}, //TODO
		"pr54103-4.c":                  {}, //TODO
		"pr54103-5.c":                  {}, //TODO
		"pr54103-6.c":                  {}, //TODO
		"pr54471.c":                    {}, //TODO
		"pr54552-1.c":                  {}, //TODO
		"pr56571.c":                    {}, //TODO
		"pr58164.c":                    {}, //TODO
		"pr59417.c":                    {}, //TODO
		"pr61375.c":                    {}, //TODO
		"pr63302.c":                    {}, //TODO
		"pr64242.c":                    {}, //TODO
		"pr65170.c":                    {}, //TODO
		"pr65595.c":                    {}, //TODO
		"pr68249.c":                    {}, //TODO
		"pr68532.c":                    {}, //TODO
		"pr70190.c":                    {}, //TODO
		"pr70199.c":                    {}, //TODO
		"pr70240.c":                    {}, //TODO
		"pr70355.c":                    {}, //TODO
		"pr70460.c":                    {}, //TODO
		"pr70903.c":                    {}, //TODO
		"pr70916.c":                    {}, //TODO
		"pr71494.c":                    {}, //TODO
		"pr72749.c":                    {}, //TODO
		"pr72802.c":                    {}, //TODO
		"pr77754-1.c":                  {}, //TODO
		"pr77754-2.c":                  {}, //TODO
		"pr77754-3.c":                  {}, //TODO
		"pr77754-4.c":                  {}, //TODO
		"pr77754-5.c":                  {}, //TODO
		"pr77754-6.c":                  {}, //TODO
		"pr78617.c":                    {}, //TODO
		"pr78694.c":                    {}, //TODO
		"pr80692.c":                    {}, //TODO
		"pr82052.c":                    {}, //TODO
		"pr82337.c":                    {}, //TODO
		"pr83547.c":                    {}, //TODO
		"pr84136.c":                    {}, //TODO
		"pr84169.c":                    {}, //TODO
		"pr84305.c":                    {}, //TODO
		"pr84748.c":                    {}, //TODO
		"pr84960.c":                    {}, //TODO
		"pr85582-2.c":                  {}, //TODO
		"pr85582-3.c":                  {}, //TODO
		"pr86659-1.c":                  {}, //TODO
		"pr86659-2.c":                  {}, //TODO
		"pr87110.c":                    {}, //TODO
		"pr89369.c":                    {}, //TODO
		"pr89655.c":                    {}, //TODO
		"pr91597.c":                    {}, //TODO
		"pr91635.c":                    {}, //TODO
		"pr92904.c":                    {}, //TODO
		"pr93213.c":                    {}, //TODO
		"pr98474.c":                    {}, //TODO
		"restrict-1.c":                 {}, //TODO
		"section.c":                    {}, //TODO
		"simd-2.c":                     {}, //TODO
		"simd-3.c":                     {}, //TODO
		"sra-1.c":                      {}, //TODO
		"stkalign.c":                   {}, //TODO
		"stpncpy-chk.c":                {}, //TODO
		"strcat-chk.c":                 {}, //TODO
		"strcat.c":                     {}, //TODO
		"strcspn.c":                    {}, //TODO
		"strncat-chk.c":                {}, //TODO
		"strncat.c":                    {}, //TODO
		"strncmp-2.c":                  {}, //TODO
		"strncmp.c":                    {}, //TODO
		"strncpy-chk.c":                {}, //TODO
		"strncpy.c":                    {}, //TODO
		"strspn.c":                     {}, //TODO
		"unsafe-fp-assoc-1.c":          {}, //TODO
		"va-arg-22.c":                  {}, //TODO
		"widechar-1.c":                 {}, //TODO
		"zerolen-2.c":                  {}, //TODO
	}
	blacklistCxgo := map[string]struct{}{
		"forward enum.c":      {}, //TODO
		"literal statement.c": {}, //TODO
	}
	blacklistVNMakarov := map[string]struct{}{
		// #endif without #if
		"endif.c": {},

		"0022-namespaces1.c":     {}, //TODO
		"declarator-abstract.c":  {}, //TODO
		"expression.c":           {}, //TODO
		"initialize-call.c":      {}, //TODO
		"initialize-object.c":    {}, //TODO
		"short-circuit-comma.c":  {}, //TODO
		"unary-plus.c":           {}, //TODO
		"enum_test.c":            {}, //TODO
		"fermian-2.c":            {}, //TODO
		"typedef-member-scope.c": {}, //TODO
		"typedef.c":              {}, //TODO
	}
	blacklictTCC := map[string]struct{}{
		"11.c": {}, // https://gcc.gnu.org/onlinedocs/gcc/Variadic-Macros.html#Variadic-Macros

		"02.c":                        {}, //TODO
		"03.c":                        {}, //TODO
		"04.c":                        {}, //TODO
		"06.c":                        {}, //TODO
		"07.c":                        {}, //TODO
		"08.c":                        {}, //TODO
		"09.c":                        {}, //TODO
		"10.c":                        {}, //TODO
		"14.c":                        {}, //TODO
		"15.c":                        {}, //TODO
		"17.c":                        {}, //TODO
		"18.c":                        {}, //TODO
		"19.c":                        {}, //TODO
		"20.c":                        {}, //TODO
		"21.c":                        {}, //TODO
		"pp-counter.c":                {}, //TODO
		"17_enum.c":                   {}, //TODO
		"39_typedef.c":                {}, //TODO
		"76_dollars_in_identifiers.c": {}, //TODO
		"81_types.c":                  {}, //TODO
		"88_codeopt.c":                {}, //TODO
		"90_struct-init.c":            {}, //TODO
		"92_enum_bitfield.c":          {}, //TODO
		"93_integer_promotion.c":      {}, //TODO
		"94_generic.c":                {}, //TODO
		"99_fastcall.c":               {}, //TODO
	}
	blacklistCompCert := map[string]struct{}{}
	blacklistCcgo := map[string]struct{}{}
	switch fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH) {
	case "linux/s390x":
		blacklistCompCert["aes.c"] = struct{}{} // Unsupported endianness.
		fallthrough
	case "linux/arm", "linux/arm64":
		// Uses sse2 headers.
		blacklistGame["fannkuchredux-4.c"] = struct{}{}
		blacklistGame["mandelbrot-6.c"] = struct{}{}
		blacklistGame["nbody-4.c"] = struct{}{}
		blacklistGame["nbody-8.c"] = struct{}{}
		blacklistGame["nbody-9.c"] = struct{}{}
		blacklistGame["spectral-norm-5.c"] = struct{}{}
		blacklistGame["spectral-norm-6.c"] = struct{}{}
	case "linux/386":
		blacklistCompCert["aes.c"] = struct{}{} // include file not found: "../endian.h"
	case "darwin/amd64", "darwin/arm64", "freebsd/amd64", "freebsd/386":
		blacklistCompCert["aes.c"] = struct{}{}         // include file not found: "../endian.h"
		blacklistGame["spectral-norm-4.c"] = struct{}{} // no cpu_set_t on darwin/freebsd.
	case "windows/amd64", "windows/386":
		blacklistCompCert["aes.c"] = struct{}{}       // include file not found: "../endian.h"
		blacklistCxgo["inet.c"] = struct{}{}          // include file not found: <arpa/inet.h>
		blacklistGCC["loop-2f.c"] = struct{}{}        // include file not found: <sys/mman.h>
		blacklistGCC["loop-2g.c"] = struct{}{}        // include file not found: <sys/mman.h>
		blacklistGame["fasta-4.c"] = struct{}{}       // include file not found: <err.h>
		blacklistGame["pidigits-2.c"] = struct{}{}    // include file not found: <gmp.h>
		blacklistGame["pidigits-6.c"] = struct{}{}    // include file not found: <threads.h>
		blacklistGame["pidigits-9.c"] = struct{}{}    // include file not found: <gmp.h>
		blacklistGame["pidigits.c"] = struct{}{}      // include file not found: <gmp.h>
		blacklistGame["regex-redux-2.c"] = struct{}{} // include file not found: <pcre.h>
		blacklistGame["regex-redux-3.c"] = struct{}{} // include file not found: <pcre.h>
		blacklistGame["regex-redux-4.c"] = struct{}{} // include file not found: <pcre.h>
		blacklistGame["regex-redux-5.c"] = struct{}{} // include file not found: <pcre2.h>
	case "netbsd/amd64":
		blacklistGame["spectral-norm-4.c"] = struct{}{}
	case "openbsd/amd64":
		blacklistCompCert["aes.c"] = struct{}{}         // include file not found: "../endian.h"
		blacklistGame["binary-trees.c"] = struct{}{}    // u_short undefined.
		blacklistGame["mandelbrot-7.c"] = struct{}{}    // include file not found: <omp.h>
		blacklistGame["pidigits-6.c"] = struct{}{}      // include file not found: <threads.h>
		blacklistGame["regex-redux-3.c"] = struct{}{}   // include file not found: <omp.h>
		blacklistGame["spectral-norm-4.c"] = struct{}{}
	}
	var files, ok, skip, fails int
	for _, v := range []struct {
		cfg       *Config
		dir       string
		blacklist map[string]struct{}
	}{
		{cfg, "CompCert-3.6/test/c", blacklistCompCert},
		{cfg, "ccgo", blacklistCcgo},
		{cfg, "gcc-9.1.0/gcc/testsuite/gcc.c-torture", blacklistGCC},
		{cfg, "github.com/AbsInt/CompCert/test/c", blacklistCompCert},
		{cfg, "github.com/cxgo", blacklistCxgo},
		{cfg, "github.com/gcc-mirror/gcc/gcc/testsuite", blacklistGCC},
		{cfg, "github.com/vnmakarov", blacklistVNMakarov},
		{cfg, "sqlite-amalgamation-3370200", nil},
		{cfg, "tcc-0.9.27/tests", blacklictTCC},
		{cfgGame, "benchmarksgame-team.pages.debian.net", blacklistGame},
	} {
		t.Run(v.dir, func(t *testing.T) {
			f, o, s, n := testParse(t, v.cfg, "/"+v.dir, v.blacklist)
			files += f
			ok += o
			skip += s
			fails += n
		})
	}
	t.Logf("TOTAL: files %v, skip %v, ok %v, fails %v", files, skip, ok, fails)
}

func testParse(t *testing.T, cfg *Config, dir string, blacklist map[string]struct{}) (files, ok, skip, nfails int) {
	var fails []string
	p := newParallel()
	err := walk(dir, func(pth string, fi os.FileInfo) error {
		if fi.IsDir() {
			return nil
		}

		if filepath.Ext(pth) != ".c" {
			return nil
		}

		files++
		switch {
		case re != nil:
			if !re.MatchString(pth) {
				skip++
				return nil
			}
		default:
			if _, ok := blacklist[filepath.Base(pth)]; ok {
				skip++
				return nil
			}
		}
		p.exec(func() {
			if *oTrace {
				fmt.Fprintln(os.Stderr, pth)
			}

			var err error

			func() {
				defer func() {
					if e := recover(); e != nil && err == nil {
						err = fmt.Errorf("%v: PANIC: %v", pth, e)
					}
				}()

				_, err = Parse(
					cfg,
					[]Source{
						{Name: "<predefined>", Value: predefined},
						{Name: "<builtin>", Value: builtin},
						{Name: pth, FS: cFS},
					},
				)
			}()
			p.Lock()

			defer p.Unlock()

			if err != nil {
				fails = append(fails, pth)
				t.Errorf("%v: %v", pth, err)
			} else {
				ok++
			}
		})
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	p.wg.Wait()
	for _, v := range fails {
		t.Log(v)
	}
	// fmt.Fprintf(os.Stderr, "%v: files %v, skip %v, ok %v, fails %v\n", dir, files, skip, ok, len(fails))
	t.Logf("files %v, skip %v, ok %v, fails %v", files, skip, ok, len(fails))
	return files, ok, skip, len(fails)
}
