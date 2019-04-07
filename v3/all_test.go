// Copyright 2019 The CC Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cc // import "modernc.org/cc/v3"

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/dustin/go-humanize"
	"modernc.org/mathutil"
)

func caller(s string, va ...interface{}) {
	if s == "" {
		s = strings.Repeat("%v ", len(va))
	}
	_, fn, fl, _ := runtime.Caller(2)
	fmt.Fprintf(os.Stderr, "# caller: %s:%d: ", path.Base(fn), fl)
	fmt.Fprintf(os.Stderr, s, va...)
	fmt.Fprintln(os.Stderr)
	_, fn, fl, _ = runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "# \tcallee: %s:%d: ", path.Base(fn), fl)
	fmt.Fprintln(os.Stderr)
	os.Stderr.Sync()
}

func dbg(s string, va ...interface{}) {
	if s == "" {
		s = strings.Repeat("%v ", len(va))
	}
	_, fn, fl, _ := runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "# dbg %s:%d: ", path.Base(fn), fl)
	fmt.Fprintf(os.Stderr, s, va...)
	fmt.Fprintln(os.Stderr)
	os.Stderr.Sync()
}

func TODO(...interface{}) string { //TODOOK
	_, fn, fl, _ := runtime.Caller(1)
	return fmt.Sprintf("# TODO: %s:%d:\n", path.Base(fn), fl) //TODOOK
}

func stack() []byte { return debug.Stack() }

func use(...interface{}) {}

func init() {
	use(caller, dbg, TODO, stack, tokStr, shift) //TODOOK
}

// ----------------------------------------------------------------------------

const (
	maxFiles = 2000

	parserTestBuiltin = `
#define __DI__
#define __HI__
#define __PRETTY_FUNCTION__ __func__
#define __QI__
#define __SI__
#define __builtin_add_overflow(...) 0
#define __builtin_mul_overflow(...) 0
#define __builtin_offsetof(type, member) ((__SIZE_TYPE__)&(((type*)0)->m))
#define __builtin_sub_overflow(...) 0
#define __builtin_va_arg(ap, type) (type)__builtin_va_arg_impl(ap)
#define __builtin_va_end(ap)
#define __builtin_va_list void*
#define __builtin_va_start(ap, v)
#define __declspec(...)
#define __extension__
#define __func__ NULL
#define __sync_synchronize(...)
#define __word__
__UINT16_TYPE__ __builtin_bswap16 (__UINT16_TYPE__ x);
__UINT32_TYPE__ __builtin_bswap32 (__UINT32_TYPE__ x);
__UINT64_TYPE__ __builtin_bswap64 (__UINT64_TYPE__ x);
int __builtin_clzll (unsigned long long);
int __printf__ ( const char * format, ... );
int __scanf__ ( const char *format, ... );
typedef __int128 __int128_t;
typedef char __builtin_neon_qi;
typedef char __builtin_neon_qi;
typedef int __builtin_neon_si;
typedef short __builtin_neon_hi;
typedef unsigned __int128 __uint128_t;
typedef unsigned char __builtin_neon_uqi;
void __builtin_va_arg_impl(void* ap);
`
)

var (
	oDev      = flag.Bool("dev", false, "Enable developer tests/downloads.")
	oDownload = flag.Bool("download", false, "Download missing testdata. Add -dev to download also 100+ MB of developer resources.")
	oMaxFiles = flag.Int("maxFiles", maxFiles, "")
	oRE       = flag.String("re", "", "")
	oTrace    = flag.Bool("trc", false, "Print tested paths.")
	oWalkDir  = flag.String("walkDir", "testdata", "")

	gccDir    = filepath.FromSlash("testdata/gcc-8.3.0")
	sqliteDir = filepath.FromSlash("testdata/sqlite-amalgamation-3270200")

	downloads = []struct {
		dir, url string
		sz       int
		dev      bool
	}{
		{gccDir, "http://mirrors-usa.go-parts.com/gcc/releases/gcc-8.3.0/gcc-8.3.0.tar.gz", 114000, true},
		{sqliteDir, "https://www.sqlite.org/2019/sqlite-amalgamation-3270200.zip", 2200, false},
	}

	testBuiltinSource   *cachedPPFile
	testIncludes        []string
	testPredef          string
	testPredefGNU       string
	testPredefGNUSource *cachedPPFile
	testPredefSource    *cachedPPFile
	testSQLiteSource    *cachedPPFile
	testShellSource     *cachedPPFile
	testSysIncludes     []string
	testWD              string
)

func init() {
	isTesting = true
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	flag.BoolVar(&panicOnParserError, "panicOnParserError", false, "Panic on parser error.") //TODOOK

	flag.Parse()
	var err error
	if testWD, err = os.Getwd(); err != nil {
		log.Fatalf("Cannot determine working dir: %v", err)
	}

	if s := *oWalkDir; !filepath.IsAbs(s) {
		*oWalkDir = filepath.Join(testWD, s)
	}

	if testPredefGNU, testIncludes, testSysIncludes, err = HostConfig(""); err != nil {
		log.Fatal("Cannot acquire host cpp configuration.")
		return
	}

	cfg := &Config{}
	if testPredefGNUSource, err = cache.getValue(newContext(cfg), "<predefined>", testPredefGNU); err != nil {
		log.Fatal(err)
	}

	a := strings.Split(testPredefGNU, "\n")
	w := 0
	for _, v0 := range a {
		v := strings.TrimSpace(strings.ToLower(v0))
		if !strings.HasPrefix(v, "#define __gnu") && !strings.HasPrefix(v, "#define __gcc") {
			a[w] = v0
			w++
		}
	}
	testPredef = strings.Join(a[:w], "\n")
	if testPredefSource, err = cache.getValue(newContext(cfg), "<predefined>", testPredef); err != nil {
		log.Fatal(err)
	}

	if testBuiltinSource, err = cache.getValue(newContext(cfg), "<built-in>", parserTestBuiltin); err != nil {
		log.Fatal(err)
	}

	testSysIncludes = testSysIncludes[:len(testSysIncludes):len(testSysIncludes)]
	testIncludes = testIncludes[:len(testIncludes):len(testIncludes)]
	testIncludes = append(testIncludes, "@")
	testIncludes = append(testIncludes, testSysIncludes...)

	if *oDownload {
		download()
	}

	path := filepath.Join(testWD, sqliteDir)
	if _, err := os.Stat(path); err != nil {
		log.Fatalf("Missing resources in %s. Please run 'go test -download' to fix.", path)
		return
	}

	path = filepath.Join(testWD, sqliteDir, "shell.c")
	if testShellSource, err = cache.getFile(newContext(cfg), path); err != nil {
		log.Fatal(err)
	}

	path = filepath.Join(testWD, sqliteDir, "sqlite3.c")
	if testSQLiteSource, err = cache.getFile(newContext(cfg), path); err != nil {
		log.Fatal(err)
	}
}

func download() {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}

	defer os.RemoveAll(tmp)

	for _, v := range downloads {
		dir := filepath.FromSlash(v.dir)
		root := filepath.Dir(v.dir)
		fi, err := os.Stat(dir)
		switch {
		case err == nil:
			if !fi.IsDir() {
				fmt.Fprintf(os.Stderr, "expected %s to be a directory\n", dir)
			}
			continue
		default:
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "%s", err)
				continue
			}

			if v.dev && !*oDev {
				fmt.Printf("Not downloading (no -dev) %v MB from %s\n", float64(v.sz)/1000, v.url)
				continue
			}

		}

		if err := func() error {
			fmt.Printf("Downloading %v MB from %s\n", float64(v.sz)/1000, v.url)
			resp, err := http.Get(v.url)
			if err != nil {
				return err
			}

			defer resp.Body.Close()

			base := filepath.Base(v.url)
			name := filepath.Join(tmp, base)
			f, err := os.Create(name)
			if err != nil {
				return err
			}

			defer os.Remove(name)

			n, err := io.Copy(f, resp.Body)
			if err != nil {
				return err
			}

			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			switch {
			case strings.HasSuffix(base, ".tar.gz"):
				gr, err := gzip.NewReader(bufio.NewReader(f))
				if err != nil {
					return err
				}

				tr := tar.NewReader(gr)
				for {
					hdr, err := tr.Next()
					if err != nil {
						if err != io.EOF {
							return err
						}

						return nil
					}

					switch hdr.Typeflag {
					case tar.TypeDir:
						if err = os.MkdirAll(filepath.Join(root, hdr.Name), 0770); err != nil {
							return err
						}
					case tar.TypeReg, tar.TypeRegA:
						f, err := os.OpenFile(filepath.Join(root, hdr.Name), os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
						if err != nil {
							return err
						}

						w := bufio.NewWriter(f)
						if _, err = io.Copy(w, tr); err != nil {
							return err
						}

						if err := w.Flush(); err != nil {
							return err
						}

						if err := f.Close(); err != nil {
							return err
						}
					default:
						return fmt.Errorf("unexpected tar header typeflag %#02x", hdr.Typeflag)
					}
				}
			case strings.HasSuffix(base, ".zip"):
				r, err := zip.NewReader(f, n)
				if err != nil {
					return err
				}

				for _, f := range r.File {
					fi := f.FileInfo()
					if fi.IsDir() {
						if err := os.MkdirAll(filepath.Join(root, f.Name), 0770); err != nil {
							return err
						}

						continue
					}

					if err := func() error {
						rc, err := f.Open()
						if err != nil {
							return err
						}

						defer rc.Close()

						dname := filepath.Join(root, f.Name)
						g, err := os.Create(dname)
						if err != nil {
							return err
						}

						defer g.Close()

						n, err = io.Copy(g, rc)
						return err
					}(); err != nil {
						return err
					}
				}
				return nil
			}
			panic("internal error") //TODOOK
		}(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func h(v interface{}) string {
	switch x := v.(type) {
	case int:
		return humanize.Comma(int64(x))
	case int64:
		return humanize.Comma(x)
	case uint64:
		return humanize.Comma(int64(x))
	case float64:
		return humanize.CommafWithDigits(x, 0)
	default:
		panic(fmt.Errorf("%T", x)) //TODOOK
	}
}

func skipDir(path string) error {
	sp := filepath.ToSlash(path)
	if strings.Contains(sp, "/.") {
		return filepath.SkipDir
	}

	return nil
}

func shift(tok Token) string {
	pc, _, _, _ := runtime.Caller(2)
	caller := runtime.FuncForPC(pc - 1)
	return fmt.Sprintf("# %v: %v", caller.Name(), PrettyString(tok))
}

func tokStr(toks interface{}, sep string) string {
	var b strings.Builder
	switch x := toks.(type) {
	case []token3:
		for i, v := range x {
			if i != 0 {
				b.WriteString(sep)
			}
			b.WriteString(v.String())
		}
	case []cppToken:
		for i, v := range x {
			if i != 0 {
				b.WriteString(sep)
			}
			b.WriteString(v.String())
		}
	case []Token:
		for i, v := range x {
			if i != 0 {
				b.WriteString(sep)
			}
			b.WriteString(v.String())
		}
	default:
		panic(fmt.Errorf("%T", x)) //TODOOK
	}
	return b.String()
}

func exampleAST(rule int, src string) interface{} {
	cfg := &Config{ignoreErrors: true}
	ast, _ := Parse(cfg, nil, nil, []Source{{Name: "example.c", Value: src}})
	if ast == nil {
		return "FAIL"
	}

	// dbg("", PrettyString(ast.TranslationUnit))
	pc, _, _, _ := runtime.Caller(1)
	typ := runtime.FuncForPC(pc - 1).Name()
	i := strings.LastIndexByte(typ, '.')
	typ = typ[i+1+len("Example"):]
	i = strings.LastIndexByte(typ, '_')
	typ = typ[:i]
	var node Node
	depth := mathutil.MaxInt
	walkExample(typ, ast.TranslationUnit, 0, &node, &depth)
	return node
}

func walkExample(typ string, n Node, depth int, out *Node, pdepth *int) {
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
			walkExample(typ, x, depth+1, out, pdepth)
		}
	}
}