// Copyright 2019 The CC Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cc // import "modernc.org/cc/v3"

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"testing"
	"time"
)

func TestTranslateSQLite(t *testing.T) {
	cfg := &Config{ABI: testABI}
	root := filepath.Join(testWD, filepath.FromSlash(sqliteDir))
	t.Run("shell.c", func(t *testing.T) { testTranslate(t, cfg, testPredef, filepath.Join(root, "shell.c")) })
	t.Run("shell.c/gnu", func(t *testing.T) { testTranslate(t, cfg, testPredefGNU, filepath.Join(root, "shell.c")) })
	t.Run("sqlite3.c", func(t *testing.T) { testTranslate(t, cfg, testPredef, filepath.Join(root, "sqlite3.c")) })
	t.Run("sqlite3.c/gnu", func(t *testing.T) { testTranslate(t, cfg, testPredefGNU, filepath.Join(root, "sqlite3.c")) })
}

var (
	benchmarkTranslateSQLiteAST *AST
	testTranslateSQLiteAST      *AST
)

func testTranslate(t *testing.T, cfg *Config, predef string, files ...string) {
	testTranslateSQLiteAST = nil
	sources := []Source{
		{Name: "<predefined>", Value: predef},
		{Name: "<built-in>", Value: parserTestBuiltin},
	}
	for _, v := range files {
		sources = append(sources, Source{Name: v})
	}
	ctx := newContext(cfg)
	var m0, m1 runtime.MemStats
	var err error
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m0)
	t0 := time.Now()
	if testTranslateSQLiteAST, err = translate(ctx, testIncludes, testSysIncludes, sources); err != nil {
		t.Error(err)
	}
	d := time.Since(t0)
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m1)
	t.Logf("sources %v, bytes %v, %v, %v B/s, mem %v",
		h(ctx.tuSources), h(ctx.tuSize), d, h(float64(time.Second)*float64(ctx.tuSize)/float64(d)), h(m1.Alloc-m0.Alloc))
}

func BenchmarkTranslateSQLite(b *testing.B) {
	cfg := &Config{ABI: testABI}
	root := filepath.Join(testWD, filepath.FromSlash(sqliteDir))
	b.Run("shell.c", func(b *testing.B) { benchmarkTranslateSQLite(b, cfg, testPredef, filepath.Join(root, "shell.c")) })
	b.Run("shell.c/gnu", func(b *testing.B) { benchmarkTranslateSQLite(b, cfg, testPredefGNU, filepath.Join(root, "shell.c")) })
	b.Run("sqlite3.c", func(b *testing.B) { benchmarkTranslateSQLite(b, cfg, testPredef, filepath.Join(root, "sqlite3.c")) })
	b.Run("sqlite3.c/gnu", func(b *testing.B) { benchmarkTranslateSQLite(b, cfg, testPredefGNU, filepath.Join(root, "sqlite3.c")) })
}

func benchmarkTranslateSQLite(b *testing.B, cfg *Config, predef string, files ...string) {
	sources := []Source{
		{Name: "<predefined>", Value: predef},
		{Name: "<built-in>", Value: parserTestBuiltin},
	}
	for _, v := range files {
		sources = append(sources, Source{Name: v})
	}
	ctx := newContext(cfg)
	// Warm up the cache
	if _, err := parse(ctx, testIncludes, testSysIncludes, sources); err != nil {
		b.Error(err)
		return
	}

	sz := ctx.tuSize
	var err error
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if benchmarkTranslateSQLiteAST, err = Translate(cfg, testIncludes, testSysIncludes, sources); err != nil {
			b.Error(err)
		}
	}
	b.SetBytes(sz)
}

var (
	benchmarkTranslateGCCAST *AST
	testTranslateGCCAST      *AST
)

func TestTranslateGCC(t *testing.T) {
	if testing.Short() {
		t.Skip("-short")
		return
	}

	cfg := &Config{
		ABI:                        testABI,
		ignoreUndefinedIdentifiers: true,
	}
	root := filepath.Join(testWD, filepath.FromSlash(gccDir))
	if _, err := os.Stat(root); err != nil {
		t.Skipf("Missing resources in %s. Please run 'go test -download -dev' to fix.", root)
	}

	ok := 0
	for _, v := range []string{
		"gcc/testsuite/gcc.c-torture/compile",
		"gcc/testsuite/gcc.c-torture/execute",
	} {
		t.Run(v, func(t *testing.T) {
			ok += testTranslateDir(t, cfg, testPredef, filepath.Join(root, filepath.FromSlash(v)))
		})
		t.Run(v+"/gnu", func(t *testing.T) {
			ok += testTranslateDir(t, cfg, testPredefGNU, filepath.Join(root, filepath.FromSlash(v)))
		})
	}
	t.Logf("ok %v", h(ok))
}

func testTranslateDir(t *testing.T, cfg *Config, predef, dir string) (ok int) {
	var re *regexp.Regexp
	if s := *oRE; s != "" {
		re = regexp.MustCompile(s)
	}

	var files, psources int
	var bytes int64
	var m0, m1 runtime.MemStats
	testTranslateGCCAST = nil
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m0)
	t0 := time.Now()
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return skipDir(path)
		}

		if filepath.Ext(path) != ".c" && filepath.Ext(path) != ".h" || info.Mode()&os.ModeType != 0 {
			return nil
		}

		files++

		if re != nil && !re.MatchString(path) {
			return nil
		}

		sources := []Source{
			{Name: "<predefined>", Value: predef},
			{Name: "<built-in>", Value: parserTestBuiltin},
			{Name: path},
		}
		ctx := newContext(cfg)

		defer func() {
			psources += ctx.tuSources
			bytes += ctx.tuSize
		}()

		if *oTrace {
			fmt.Fprintln(os.Stderr, files, path)
		}
		if testTranslateGCCAST, err = parse(ctx, testIncludes, testSysIncludes, sources); err != nil {
			if predef == testPredefGNU {
				t.Error(err)
			}
			return nil
		}

		if err = testTranslateGCCAST.Typecheck(); err != nil {
			t.Error(err)
			return nil
		}

		ok++
		return nil
	}); err != nil {
		t.Error(err)
	}
	d := time.Since(t0)
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m1)
	t.Logf("files %v, sources %v, bytes %v, ok %v, %v, %v B/s, mem %v",
		h(files), h(psources), h(bytes), h(ok), d, h(float64(time.Second)*float64(bytes)/float64(d)), h(m1.Alloc-m0.Alloc))
	if files != ok && predef == testPredefGNU {
		t.Errorf("files %v, bytes %v, ok %v", files, bytes, ok)
	}
	return ok
}

func BenchmarkTranslateGCC(b *testing.B) {
	root := filepath.Join(testWD, filepath.FromSlash(gccDir))
	if _, err := os.Stat(root); err != nil {
		b.Skipf("Missing resources in %s. Please run 'go test -download -dev' to fix.", root)
	}

	cfg := &Config{
		ABI:                        testABI,
		ignoreUndefinedIdentifiers: true,
	}
	for _, v := range []string{
		"gcc/testsuite/gcc.c-torture/compile",
		"gcc/testsuite/gcc.c-torture/execute",
	} {
		b.Run(v+"/gnu", func(b *testing.B) {
			benchmarkTranslateDir(b, cfg, testPredefGNU, filepath.Join(root, filepath.FromSlash(v)))
		})
	}
}

func benchmarkTranslateDir(b *testing.B, cfg *Config, predef, dir string) {
	var bytes int64
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytes = 0
		if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return skipDir(path)
			}

			if filepath.Ext(path) != ".c" && filepath.Ext(path) != ".h" || info.Mode()&os.ModeType != 0 {
				return nil
			}

			sources := []Source{
				{Name: "<predefined>", Value: predef},
				{Name: "<built-in>", Value: parserTestBuiltin},
				{Name: path},
			}
			ctx := newContext(cfg)
			if benchmarkTranslateGCCAST, err = parse(ctx, testIncludes, testSysIncludes, sources); err != nil {
				if predef == testPredefGNU {
					b.Error(err)
				}
				return nil
			}

			if err = benchmarkTranslateGCCAST.Typecheck(); err != nil {
				b.Error(err)
				return nil
			}
			bytes += ctx.tuSize
			return nil
		}); err != nil {
			b.Error(err)
		}
	}
	b.SetBytes(bytes)
}

// jnml@4670:~/src/modernc.org/cc/v3$ date ; go test -timeout 24h -v -dev -run DevTranslate -maxFiles -1 | tee log
// Sun Apr 14 23:14:06 CEST 2019
// === RUN   TestDevTranslate
// === RUN   TestDevTranslate/.c/gnu
// --- PASS: TestDevTranslate (361.82s)
//     --- PASS: TestDevTranslate/.c/gnu (361.82s)
//         ---- pass at least 1000 files
//           5666/5713   99.18% gcc-8.3.0/gcc/testsuite/gcc.target/i386
//           3969/4293   92.45% gcc-8.3.0/gcc/testsuite/gcc.dg
//           1759/1759  100.00% gcc-8.3.0/gcc/testsuite/gcc.c-torture/compile
//           1558/1564   99.62% gcc-8.3.0/gcc/testsuite/gcc.dg/tree-ssa
//           1475/1475  100.00% gcc-8.3.0/gcc/testsuite/gcc.c-torture/execute
//           1050/1094   95.98% gcc-8.3.0/gcc/testsuite/gcc.dg/torture
//           1041/1041  100.00% gcc-8.3.0/gcc/testsuite/gcc.dg/vect
//         files 32,435, sources 932,448, bytes 11,434,496,182, ok 25,586, 6m1.329721096s, 31,645,600 B/s, mem 2,350,776,816
// PASS
// ok  	modernc.org/cc/v3	362.214s
// jnml@4670:~/src/modernc.org/cc/v3$

// ==== jnml@e5-1650:~/src/modernc.org/cc/v3> date |& tee log ; go test -timeout 24h -v -dev -run DevTranslate -maxFiles -1 |& tee -a log
// Pá dub 12 16:49:30 CEST 2019
// === RUN   TestDevTranslate
// === RUN   TestDevTranslate/.c
// === RUN   TestDevTranslate/.c/gnu
// --- PASS: TestDevTranslate (778.43s)
//     --- PASS: TestDevTranslate/.c (389.08s)
//         ---- pass at least 1000 files
//           5670/5713   99.25% gcc-8.3.0/gcc/testsuite/gcc.target/i386
//           3967/4293   92.41% gcc-8.3.0/gcc/testsuite/gcc.dg
//           1759/1759  100.00% gcc-8.3.0/gcc/testsuite/gcc.c-torture/compile
//           1560/1564   99.74% gcc-8.3.0/gcc/testsuite/gcc.dg/tree-ssa
//           1475/1475  100.00% gcc-8.3.0/gcc/testsuite/gcc.c-torture/execute
//           1041/1041  100.00% gcc-8.3.0/gcc/testsuite/gcc.dg/vect
//           1018/1094   93.05% gcc-8.3.0/gcc/testsuite/gcc.dg/torture
//         files 32,435, sources 916,614, bytes 11,159,743,585, ok 25,493, 6m28.113484053s, 28,753,815 B/s, mem 2,348,989,736
//     --- PASS: TestDevTranslate/.c/gnu (389.34s)
//         ---- pass at least 1000 files
//           5670/5713   99.25% gcc-8.3.0/gcc/testsuite/gcc.target/i386
//           3972/4293   92.52% gcc-8.3.0/gcc/testsuite/gcc.dg
//           1759/1759  100.00% gcc-8.3.0/gcc/testsuite/gcc.c-torture/compile
//           1560/1564   99.74% gcc-8.3.0/gcc/testsuite/gcc.dg/tree-ssa
//           1475/1475  100.00% gcc-8.3.0/gcc/testsuite/gcc.c-torture/execute
//           1041/1041  100.00% gcc-8.3.0/gcc/testsuite/gcc.dg/vect
//           1018/1094   93.05% gcc-8.3.0/gcc/testsuite/gcc.dg/torture
//         files 32,435, sources 908,727, bytes 11,153,976,670, ok 25,581, 6m28.394236689s, 28,718,182 B/s, mem 23,613,264
// PASS
// ok  	modernc.org/cc/v3	778.868s
// ==== jnml@e5-1650:~/src/modernc.org/cc/v3>

func TestDevTranslate(t *testing.T) {
	if !*oDev {
		t.Skip("-dev to enable")
		return
	}

	if testing.Short() {
		t.Skip("-short")
		return
	}

	t.Run(".c/gnu", func(t *testing.T) {
		testDevTranslate(t, testPredefGNU, func(s string) bool { return filepath.Ext(s) == ".c" }, 1000)
	})
}

func testDevTranslate(t *testing.T, predef string, filter func(string) bool, min int) {
	var re *regexp.Regexp
	if x := *oRE; x != "" {
		re = regexp.MustCompile(x)
	}

	cfg := &Config{
		ABI:                        testABI,
		ignoreUndefinedIdentifiers: true,
	}
	m := map[string]int{}
	n := map[string]int{}
	limit := *oMaxFiles
	var files, psources, ok int
	var bytes int64
	strip := len(*oWalkDir) + 1
	var m0, m1 runtime.MemStats
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m0)
	t0 := time.Now()
	if err := filepath.Walk(*oWalkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return skipDir(path)
		}

		if !filter(path) || info.Mode()&os.ModeType != 0 {
			return nil
		}

		if re != nil && !re.MatchString(path) {
			return nil
		}

		if limit == 0 {
			return nil
		}

		limit--
		files++
		key := filepath.Dir(path)[strip:]
		n[key]++
		sources := []Source{
			{Name: "<predefined>", Value: predef},
			{Name: "<built-in>", Value: parserTestBuiltin},
			{Name: path},
		}
		ctx := newContext(cfg)
		if *oTrace {
			fmt.Fprintln(os.Stderr, files, path)
		}
		if ast, err := parse(ctx, testIncludes, testSysIncludes, sources); err == nil && ast.Typecheck() == nil {
			ok++
			m[key]++
		}
		psources += ctx.tuSources
		bytes += ctx.tuSize
		return nil
	}); err != nil {
		t.Error(err)
	}
	d := time.Since(t0)
	debug.FreeOSMemory()
	runtime.ReadMemStats(&m1)
	var a []string
	for k, v := range m {
		if v >= min {
			a = append(a, fmt.Sprintf("%6d/%-5d %6.2f%% %s", v, n[k], 100*float64(v)/float64(n[k]), k))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(a)))
	if len(a) != 0 {
		t.Logf("---- pass at least %v files", min)
		for _, v := range a {
			t.Log(v)
		}
	}
	t.Logf("files %v, sources %v, bytes %v, ok %v, %v, %v B/s, mem %v",
		h(files), h(psources), h(bytes), h(ok), d, h(float64(time.Second)*float64(bytes)/float64(d)), h(m1.Alloc-m0.Alloc))
}

func BenchmarkDevTranslate(b *testing.B) {
	if !*oDev {
		b.Skip("-dev to enable")
		return
	}

	b.Run(".c/gnu", func(b *testing.B) { benchmarkDevTranslate(b, testPredefGNU) })
}

func benchmarkDevTranslate(b *testing.B, predef string) {
	var re *regexp.Regexp
	if x := *oRE; x != "" {
		re = regexp.MustCompile(x)
	}

	cfg := &Config{ABI: testABI}
	limit := *oMaxFiles
	var bytes int64
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytes = 0
		if err := filepath.Walk(*oWalkDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return skipDir(path)
			}

			if filepath.Ext(path) != ".c" || info.Mode()&os.ModeType != 0 {
				return nil
			}

			if re != nil && !re.MatchString(path) {
				return nil
			}

			if limit == 0 {
				return nil
			}

			limit--
			sources := []Source{
				{Name: "<predefined>", Value: predef},
				{Name: "<built-in>", Value: parserTestBuiltin},
				{Name: path},
			}
			ctx := newContext(cfg)
			translate(ctx, testIncludes, testSysIncludes, sources)
			bytes += ctx.tuSize
			return nil
		}); err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(bytes)
}