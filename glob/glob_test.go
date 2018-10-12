package glob

// Code modified from: https://github.com/bmatcuk/doublestar

// The MIT License (MIT)
//
// Copyright (c) 2014 Bob Matcuk
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"os"
	"path/filepath"
	"testing"
)

type MatchTest struct {
	pattern     string   // pattern to test
	expected    string   // expected results
	shouldMatch bool     // whether pattern should match expected
	ignores     []string // paths to ignore
	traverse    bool     // whether to traverse dir matches
}

var matchTests = []MatchTest{
	// no traversal, no ignores
	{"abc", "abc", true, nil, false},
	{"*", "abc", true, nil, false},
	{"*c", "abc", true, nil, false},
	{"a*", "axbxcxdxe", true, nil, false},
	{"a*", "ab/c", false, nil, false},
	{"a*/b", "abc/b", true, nil, false},
	{"a*/b", "a/c/b", false, nil, false},
	{"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil, false},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil, false},
	{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil, false},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil, false},
	{"a*b?c*x", "abxbbxdbxebxczzx", true, nil, false},
	{"a*b?c*x", "abxbbxdbxebxczzy", false, nil, false},
	{"ab[c]", "abc", true, nil, false},
	{"ab[b-d]", "abc", true, nil, false},
	{"ab[e-g]", "abc", false, nil, false},
	{"ab[^c]", "abc", false, nil, false},
	{"ab[^b-d]", "abc", false, nil, false},
	{"ab[^e-g]", "abc", true, nil, false},
	{"a\\*b", "ab", false, nil, false},
	{"[a-ζ]*", "α", true, nil, false},
	{"*[a-ζ]", "A", false, nil, false},
	{"a?b", "a/b", false, nil, false},
	{"a*b", "a/b", false, nil, false},
	{"[\\]a]", "]", true, nil, false},
	{"[\\-]", "-", true, nil, false},
	{"[x\\-]", "x", true, nil, false},
	{"[x\\-]", "-", true, nil, false},
	{"[x\\-]", "z", false, nil, false},
	{"[\\-x]", "x", true, nil, false},
	{"[\\-x]", "-", true, nil, false},
	{"[\\-x]", "a", false, nil, false},
	{"[]a]", "]", false, nil, false},
	{"[-]", "-", false, nil, false},
	{"[x-]", "x", false, nil, false},
	{"[x-]", "-", false, nil, false},
	{"[x-]", "z", false, nil, false},
	{"[-x]", "x", false, nil, false},
	{"[-x]", "-", false, nil, false},
	{"[-x]", "a", false, nil, false},
	{"\\", "a", false, nil, false},
	{"[a-b-c]", "a", false, nil, false},
	{"[", "a", false, nil, false},
	{"[^", "a", false, nil, false},
	{"[^bc", "a", false, nil, false},
	{"a[", "a", false, nil, false},
	{"a[", "ab", false, nil, false},
	{"*x", "xxx", true, nil, false},
	{"a/**", "a", false, nil, false},
	{"**/c", "c", true, nil, false},
	{"**/c", "b/c", true, nil, false},
	{"**/c", "a/b/c", true, nil, false},
	{"**/c", "a/b", false, nil, false},
	{"**/c", "abcd", false, nil, false},
	{"**/c", "a/abc", false, nil, false},
	{"a/**/b", "a/b", true, nil, false},
	{"a/**/c", "a/b/c", true, nil, false},
	{"a/**/d", "a/b/c/d", true, nil, false},
	{"a/\\**", "a/b/c", false, nil, false},
	{"ab{c,d}", "abc", true, nil, false},
	{"ab{c,d,*}", "abcde", true, nil, false},
	{"ab{c,d}[", "abcd", false, nil, false},
	{"abc**", "abc", true, nil, false},
	{"**abc", "abc", true, nil, false},
	{"broken-symlink", "broken-symlink", true, nil, false},
	{"working-symlink/c/*", "working-symlink/c/d", true, nil, false},
	{"working-sym*/*", "working-symlink/c", true, nil, false},
	{"b/**/f", "b/symlink-dir/f", true, nil, false},

	// traversal, no ignores
	{"abc", "abc/b", true, nil, true},
	{"*", "c", true, nil, true},
	{"**", "a/b/c/d", true, nil, true},
	{"*c", "c", true, nil, true},
	{"a*", "abcd", true, nil, true},
	{"a*", "a/b/c", false, nil, true},
	{"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil, true},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil, true},
	{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil, true},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil, true},
	{"a*b?c*x", "abxbbxdbxebxczzx", true, nil, true},
	{"a*b?c*x", "abxbbxdbxebxczzy", false, nil, true},
	{"ab[c]", "abc/b", true, nil, true},
	{"ab[b-d]", "abc/b", true, nil, true},
	{"ab[e-g]", "abc/b", false, nil, true},
	{"ab[^c]", "abc/b", false, nil, true},
	{"ab[^b-d]", "abc/b", false, nil, true},
	{"ab[^e-g]", "abc/b", true, nil, true},
	{"a\\*b", "ab", false, nil, true},
	{"[a-ζ]*", "α", true, nil, true},
	{"*[a-ζ]", "A", false, nil, true},
	{"a?b", "a/b", false, nil, true},
	{"a*b", "a/b", false, nil, true},
	{"[\\]a]", "]", true, nil, true},
	{"[\\-]", "-", true, nil, true},
	{"[x\\-]", "x", true, nil, true},
	{"[x\\-]", "-", true, nil, true},
	{"[x\\-]", "z", false, nil, true},
	{"[\\-x]", "x", true, nil, true},
	{"[\\-x]", "-", true, nil, true},
	{"[\\-x]", "a", false, nil, true},
	{"[]a]", "]", false, nil, true},
	{"[-]", "-", false, nil, true},
	{"[x-]", "x", false, nil, true},
	{"[x-]", "-", false, nil, true},
	{"[x-]", "z", false, nil, true},
	{"[-x]", "x", false, nil, true},
	{"[-x]", "-", false, nil, true},
	{"[-x]", "a", false, nil, true},
	{"\\", "a", false, nil, true},
	{"[a-b-c]", "a", false, nil, true},
	{"[", "a", false, nil, true},
	{"[^", "a", false, nil, true},
	{"[^bc", "a", false, nil, true},
	{"a[", "a", false, nil, true},
	{"a[", "ab", false, nil, true},
	{"*x", "xxx", true, nil, true},
	{"a/**", "a", false, nil, true},
	{"**/c", "c", true, nil, true},
	{"**/c", "b/c", true, nil, true},
	{"**/c", "a/b/c/d", true, nil, true},
	{"**/c", "a/b", false, nil, true},
	{"**/c", "abcd", false, nil, true},
	{"**/c", "a/abc", false, nil, true},
	{"a/**/b", "a/b/c/d", true, nil, true},
	{"a/**/c", "a/b/c/d", true, nil, true},
	{"a/**/d", "a/b/c/d", true, nil, true},
	{"a/\\**", "a/b/c", false, nil, true},
	{"ab{c,d}", "abc/b", true, nil, true},
	{"ab{c,d,*}", "abcde", true, nil, true},
	{"ab{c,d}[", "abcd", false, nil, true},
	{"abc**", "abc/b", true, nil, true},
	{"**abc", "abc/b", true, nil, true},
	{"broken-symlink", "broken-symlink", true, nil, true},
	{"working-symlink/c/*", "working-symlink/c/d", true, nil, true},
	{"working-sym*/*", "working-symlink/c/d", true, nil, true},
	{"b/**/f", "b/symlink-dir/f", true, nil, true},

	// traversal, and ignores
	{"a", "a/b/c/d", false, []string{"a/b"}, true},
	{"a", "a/b/c/d", false, []string{"a/b/c/d"}, true},
	{"a", "a/b/c/d", true, []string{"a/c/b"}, true},
	{"abc", "abc/b", false, []string{"abc"}, true},
	{"axbxcxdxe", "axbxcxdxe/xxx/f", false, []string{"axbxcxdxe"}, true},
	{"axbxcxdxe/*", "axbxcxdxe/f", false, []string{"axbxcxdxe"}, true},
	{"axbxcxdxe**", "axbxcxdxe/f", false, []string{"axbxcxdxe/f"}, true},
}

func TestGlob(t *testing.T) {
	abspath, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting current working directory: %v", err)
		return
	}
	for i, tt := range matchTests {
		// test both relative paths and absolute paths
		testGlobWith(t, tt.pattern, tt.expected, tt.shouldMatch, tt.ignores, tt.traverse, i, "")
		testGlobWith(t, tt.pattern, tt.expected, tt.shouldMatch, tt.ignores, tt.traverse, i, abspath)
	}
}

func testGlobWith(t *testing.T, pattern string, expected string, shouldMatch bool, ignores []string, traverse bool, index int, basepath string) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Glob(%#q) panicked: %#v", index, pattern, r)
		}
	}()

	var ignoresAbs []string
	for _, ignore := range ignores {
		ignoresAbs = append(ignoresAbs, filepath.Join(basepath, "testdata", ignore))
	}

	pattern = filepath.Join(basepath, "testdata", pattern)
	expected = filepath.Join(basepath, "testdata", expected)
	matches, _ := Glob(nil, pattern, ignoresAbs, traverse)

	_, ok := matches[expected]
	if ok != shouldMatch {
		if shouldMatch {
			t.Errorf("#%v. Glob(%#q) - doesn't contain %v, but should", index, pattern, expected)
		} else {
			t.Errorf("#%v. Glob(%#q) - contains %v, but shouldn't", index, pattern, expected)
		}
	}
}
