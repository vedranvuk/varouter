// Copyright 2020 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package varouter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vedranvuk/randomex"
)

// DebugPrintElements prints Varouter internals.
func DebugPrintElements(vr *Varouter) {
	fmt.Printf(`Varouter:	
Template count:     '%d'
Override char:      '%s'
Separator char:     '%s'
Variable char:      '%s'
Prefix char:        '%s'
Wildcard One char : '%s'
Wildcard Many char: '%s'
`, vr.count, string(vr.override), string(vr.separator), string(vr.variable),
		string(vr.prefix), string(vr.wildcardone), string(vr.wildcardmany))
	fmt.Println()
	fmt.Println("Root:")
	fmt.Println()
	debugPrintElement(vr.root, 0)
	fmt.Println()
	fmt.Println("Subs:")
	fmt.Println()
	debugPrintElements(vr.root, 1)
	fmt.Println(vr.DefinedTemplates())
	fmt.Println()
}

func debugPrintElement(e *element, indent int) {
	ind := strings.Repeat("\t", indent)
	fmt.Printf("%sNum subs:           '%d'\n", ind, len(e.subs))
	fmt.Printf("%sTemplate:           '%s'\n", ind, e.template)
	fmt.Printf("%sIs Override:        '%t'\n", ind, e.isoverride)
	fmt.Printf("%sIs Prefix:          '%t'\n", ind, e.isprefix)
	fmt.Printf("%sIs Wildcard:        '%t'\n", ind, e.iswildcard)
	fmt.Printf("%sHas Variable:       '%s'\n", ind, e.hasvariable)
	fmt.Printf("%sHas Prefixes:       '%t'\n", ind, e.hasprefixes)
	fmt.Printf("%sHas Wildcards:      '%t'\n", ind, e.haswildcards)
}

func debugPrintElements(e *element, indent int) {
	for key, val := range e.subs {
		ind := strings.Repeat("\t", indent)
		fmt.Printf("%sElement:            '%s'\n", ind, key)
		debugPrintElement(val, indent)
		fmt.Println()
		debugPrintElements(val, indent+1)
	}
}

// RegistrationData is a registration test data.
type RegistrationData struct {
	Pattern     string // Pattern to register.
	ExpectErr   bool   // ExpectErr denotes if an error is expected.
	FailMessage string // FailMessage is displayed if the test fails.
}

var RegistrationTests = []RegistrationData{
	{"", true, "Failed detecting invalid pattern."},
	{"no", true, "Failed detecting invalid pattern."},
	{"/++", true, "Failed detecting invalid pattern."},
	{"/:", true, "Failed detecting invalid pattern."},
	{"/:/", true, "Failed detecting invalid pattern."},
	{"/:+", true, "Failed detecting invalid pattern."},
	{"/:+/", true, "Failed detecting invalid pattern."},
	{"/+:", true, "Failed detecting invalid pattern."},
	{"/+:/", true, "Failed detecting invalid pattern."},
	{":no", true, "Failed detecting invalid variable name."},
	{":no:", true, "Failed detecting invalid variable name."},
	{":no/", true, "Failed detecting invalid variable name."},
	{":no/+", true, "Failed detecting invalid variable name."},
	{":no+", true, "Failed detecting invalid variable name."},
	{":no++", true, "Failed detecting invalid variable name."},
	{":no+/", true, "Failed detecting invalid variable name."},
	{":no++/", true, "Failed detecting invalid variable name."},
	{"/:no:", true, "Failed detecting invalid variable name."},
	{"!/:no:", true, "Failed detecting invalid variable name."},
	{"/:no:+", true, "Failed detecting invalid variable name."},
	{"!/:no:+", true, "Failed detecting invalid variable name."},
	{"/:no*", true, "Failed detecting wildcard in variable name."},
	{"!/:no*", true, "Failed detecting wildcard in variable name."},
	{"/:no*+", true, "Failed detecting wildcard in variable name."},
	{"!/:no*+", true, "Failed detecting wildcard in variable name."},
	{"/", false, ""},
	{"/+", true, "Failed detecting existing template."},
	{"/a+", false, ""},
	{"/a", true, "Failed detecting existing template."},
	{"!/a", true, "Failed detecting existing template."},
	{"/a*", false, ""},
	{"/a/*", false, ""},
	{"/a/b", false, ""},
	{"/a/b/:c", false, ""},
	{"/a/b/:d", true, "Failed detecting variable being registered on a path level with registered elements."},
	{"/a/b/:c/:d", false, ""},
	{"!/b", false, ""},
	{"!/b/c", false, ""},
}

func TestRegister(t *testing.T) {
	vr := New()
	for _, test := range RegistrationTests {
		err := vr.Register(test.Pattern)
		if test.ExpectErr != (err != nil) {
			if test.FailMessage != "" {
				t.Fatal(test.FailMessage + fmt.Sprintf(": %s", test.Pattern))
			}
			t.Fatal(err)
		}
	}
}

// FailMatchTest fails a Match test and prints error details.
func FailMatchTest(t *testing.T, match Match, result []string, ph Vars, expected bool) {
	t.Fatalf(`
	Match failed,
	Expected: 
	  MatchTest:    '%#+v'
	Got:      
	  Patterns:     '%#+v'
	  variables: '%#+v'
	  Matched:      '%#+v'`, match, result, ph, expected)
}

// RunMatchTests runs match tests.
func RunMatchTests(t *testing.T, tests []MatchTest) {
	for _, matchtest := range tests {
		vr := New()
		for _, pattern := range matchtest.RegisteredPatterns {
			if err := vr.Register(pattern); err != nil {
				t.Fatal(err)
			}
		}
		for _, match := range matchtest.Matches {
			patterns, variables, matched := vr.Match(match.Path)
			if matched != match.ExpectedMatch {
				DebugPrintElements(vr)
				FailMatchTest(t, match, patterns, variables, matched)
			}
			for _, expectedpattern := range match.ExpectedPatterns {
				found := false
				for i := 0; i < len(patterns); i++ {
					if patterns[i] == expectedpattern {
						found = true
					}
				}
				if len(match.ExpectedPatterns) != len(patterns) {
					DebugPrintElements(vr)
					FailMatchTest(t, match, patterns, variables, matched)
				}
				if !found {
					DebugPrintElements(vr)
					FailMatchTest(t, match, patterns, variables, matched)
				}
			}
			for expectedvariablekey, expectedvariableval := range match.Expectedvariables {
				if val, ok := variables[expectedvariablekey]; !ok || expectedvariableval != val {
					DebugPrintElements(vr)
					FailMatchTest(t, match, patterns, variables, matched)
				}
			}
		}
	}
}

// Match is a definition of expected match results.
type Match struct {
	Path              string   // Path to test against.
	ExpectedPatterns  []string // Expected matched patterns.
	Expectedvariables Vars     // Expected variables.
	ExpectedMatch     bool     // Expected Match result.
}

// MatchTest is a definiton of a Match test.
type MatchTest struct {
	RegisteredPatterns []string // Patterns to pre-register for the test.
	Matches            []Match  // Matches to run against registered patterns.
}

var MatchExactTests = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/",
			"/home",
			"/home/",
			"/home/users",
			"/home/users/",
			"/home/users//",
			"/home//users/",
			"///home///users///",
			"/-_-/<_</>_>/",
		},
		Matches: []Match{
			{
				Path:              "/",
				ExpectedPatterns:  []string{"/"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/home",
				ExpectedPatterns:  []string{"/home"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/",
				ExpectedPatterns:  []string{"/home/"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/users",
				ExpectedPatterns:  []string{"/home/users"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/users/",
				ExpectedPatterns:  []string{"/home/users/"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/users//",
				ExpectedPatterns:  []string{"/home/users//"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/home//users/",
				ExpectedPatterns:  []string{"/home//users/"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "///home///users///",
				ExpectedPatterns:  []string{"///home///users///"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "///home///users///",
				ExpectedPatterns:  []string{"///home///users///"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/-_-/<_</>_>/",
				ExpectedPatterns:  []string{"/-_-/<_</>_>/"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
		},
	},
}

func TestExactMatch(t *testing.T) {
	RunMatchTests(t, MatchExactTests)
}

var MatchVariableTests = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/home/users/:username",
			"/home/users/:username/",
			"/home/users/:username/.config/:application",
			"/home/users/:username/.config/:application/",
		},
		Matches: []Match{
			{
				Path:              "/home/users/vedran",
				ExpectedPatterns:  []string{"/home/users/:username"},
				Expectedvariables: Vars{"username": "vedran"},
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/users/vedran/",
				ExpectedPatterns:  []string{"/home/users/:username/"},
				Expectedvariables: Vars{"username": "vedran"},
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/users/vedran/.config/myapp",
				ExpectedPatterns:  []string{"/home/users/:username/.config/:application"},
				Expectedvariables: Vars{"username": "vedran", "application": "myapp"},
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/users/vedran/.config/myapp/",
				ExpectedPatterns:  []string{"/home/users/:username/.config/:application/"},
				Expectedvariables: Vars{"username": "vedran", "application": "myapp"},
				ExpectedMatch:     true,
			},
		},
	},
}

func TestVariableMatch(t *testing.T) {
	RunMatchTests(t, MatchVariableTests)
}

var MatchPrefixTests = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/+",
			"/home/+",
			"/home/vedran/+",
		},
		Matches: []Match{
			{
				Path:              "/",
				ExpectedPatterns:  []string{"/+"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/home",
				ExpectedPatterns:  []string{"/+"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/vedran",
				ExpectedPatterns:  []string{"/+", "/home/+"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/vedran/test",
				ExpectedPatterns:  []string{"/+", "/home/+", "/home/vedran/+"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
		},
	},
}

func TestPrefixMatch(t *testing.T) {
	RunMatchTests(t, MatchPrefixTests)
}

var MatchOverrideTests = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/+",
			"!/file",
			"/users/+",
			"/users/vedran/+",
			"!/users/vedran/.config",
			"!/users/vedran/.config/+",
			"!/users/vedran/.config/stack",
		},
		Matches: []Match{
			{
				Path:              "/",
				ExpectedPatterns:  []string{"/+"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/random",
				ExpectedPatterns:  []string{"/+"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/file",
				ExpectedPatterns:  []string{"!/file"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/users/vedran/.config",
				ExpectedPatterns:  []string{"!/users/vedran/.config"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/users/vedran/.config/stack",
				ExpectedPatterns:  []string{"!/users/vedran/.config/stack"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
		},
	},
}

func TestOverrideMatch(t *testing.T) {
	RunMatchTests(t, MatchOverrideTests)
}

var MatchWildcardTests = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/h*m?",
			"/h*e/u*s/???ran",
		},
		Matches: []Match{
			{
				Path:              "/home",
				ExpectedPatterns:  []string{"/h*m?"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/users/vedran",
				ExpectedPatterns:  []string{"/h*e/u*s/???ran"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
		},
	},
}

func TestWildcardMatch(t *testing.T) {
	RunMatchTests(t, MatchWildcardTests)
}

var MatchCombinedTests1 = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/home+",
			"/home/:user",
			"!/home/:user/+",
			"!/home/:user/.config",
		},
		Matches: []Match{
			{
				Path:              "/home/vedran/.config",
				ExpectedPatterns:  []string{"!/home/:user/.config"},
				Expectedvariables: Vars{"user": "vedran"},
				ExpectedMatch:     true,
			},
		},
	},
}

func TestCombined1(t *testing.T) {
	RunMatchTests(t, MatchCombinedTests1)
}

var MatchCombinedTests2 = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/+",
			"!/etc",
			"/usr",
		},
		Matches: []Match{
			{
				Path:              "/home/vedran/.config",
				ExpectedPatterns:  []string{"/+"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
		},
	},
}

func TestCombined2(t *testing.T) {
	RunMatchTests(t, MatchCombinedTests2)
}

var MatchCombinedTests3 = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/",
			"/home/:user+",
			"!/etc",
			"/usr",
		},
		Matches: []Match{
			{
				Path:              "/home/vedran/.config",
				ExpectedPatterns:  []string{"/home/:user+"},
				Expectedvariables: Vars{"user": "vedran"},
				ExpectedMatch:     true,
			},
		},
	},
}

func TestCombined3(t *testing.T) {
	RunMatchTests(t, MatchCombinedTests3)
}

var MatchCombinedTests4 = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/home/:user/t*p",
			"/home/:user/t*p/:file",
		},
		Matches: []Match{
			{
				Path:              "/home/vedran/temp",
				ExpectedPatterns:  []string{"/home/:user/t*p"},
				Expectedvariables: Vars{"user": "vedran"},
				ExpectedMatch:     true,
			},
			{
				Path:              "/home/vedran/temp/test",
				ExpectedPatterns:  []string{"/home/:user/t*p/:file"},
				Expectedvariables: Vars{"user": "vedran", "file": "test"},
				ExpectedMatch:     true,
			},
		},
	},
}

func TestCombined4(t *testing.T) {
	RunMatchTests(t, MatchCombinedTests4)
}

func TestWildcardMatcher(t *testing.T) {
	vr := NewVarouter(false, '!', '/', ':', '+', '?', '*')
	text := "sinferopopokatepetl"
	wildcard := "sin*p?p?k?t?p*t?"
	if vr.matchWildcard(&text, &wildcard) != true {
		t.Fatal("MatchWildcard failed.")
	}
}

func BenchmarkRegister(b *testing.B) {
	vl := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vl.Register("/home/users/vedran/Go/src/github.com/vedranvuk/varouter")
	}
}

func BenchmarkMatch(b *testing.B) {
	vl := New()
	vl.Register("/home/users/vedran/Go/src/github.com/vedranvuk/varouter")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vl.Match("/home/users/vedran/Go/src/github.com/vedranvuk/varouter")
	}
}

func BenchmarkWildcard(b *testing.B) {
	vr := NewVarouter(false, '!', '/', ':', '+', '?', '*')
	text := "sinferopopokatepetl"
	wildcard := "sin*p?p?k?t?p*t?"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vr.matchWildcard(&text, &wildcard)
	}
}

// BenchData is the benchmark data.
type BenchData struct {
	Templates []string // Templates to register.
	Paths     []string // Paths to test against registered templates.
}

func makeBenchData(iterations int, maxPathElems, maxPathLength int) *BenchData {
	var data *BenchData = &BenchData{}
	var NumTemplates int = iterations / maxPathElems
	if NumTemplates <= 1 {
		NumTemplates = iterations
	}
	var Extra int
	if NumTemplates > 0 {
		Extra = iterations % NumTemplates
	}
	data.Templates = make([]string, 0, NumTemplates+Extra)
	data.Paths = make([]string, 0, iterations)
	var path string
	for i := 0; i < NumTemplates; i++ {
		path = ""
		for i := 0; i < maxPathElems; i++ {
			path += "/" + randomex.String(true, true, true, false, maxPathLength)
			data.Paths = append(data.Paths, path)
		}
		data.Templates = append(data.Templates, path)
	}
	for i := 0; i < Extra; i++ {
		data.Paths = append(data.Paths, path)
	}
	return data
}

func BenchmarkMatch_8ElemNumX8Namelen(b *testing.B) {
	vr := New()
	data := makeBenchData(b.N, 8, 8)
	for _, template := range data.Templates {
		vr.Register(template)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vr.Match(data.Paths[i])
	}
}

func BenchmarkMatch_8ElemNumX8NamelenPrealloc(b *testing.B) {
	vr := New()
	vars := make(Vars)
	matches := make([]string, 0, 64)
	data := makeBenchData(b.N, 8, 8)
	for _, template := range data.Templates {
		vr.Register(template)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vr.MatchTo(&data.Paths[i], &matches, &vars)
	}
}

func BenchmarkMatch_64ElemNumX64Namelen(b *testing.B) {
	vr := New()
	data := makeBenchData(b.N, 64, 64)
	for _, template := range data.Templates {
		vr.Register(template)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vr.Match(data.Paths[i])
	}
}

func BenchmarkMatch_64ElemNumX64NamelenPrealloc(b *testing.B) {
	vr := New()
	vars := make(Vars)
	matches := make([]string, 0, 64)
	data := makeBenchData(b.N, 64, 64)
	for _, template := range data.Templates {
		vr.Register(template)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vr.MatchTo(&data.Paths[i], &matches, &vars)
	}
}

func BenchmarkMatch_8ElemNumX64Namelen(b *testing.B) {
	vr := New()
	data := makeBenchData(b.N, 8, 64)
	for _, template := range data.Templates {
		vr.Register(template)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vr.Match(data.Paths[i])
	}
}

func BenchmarkMatch_64ElemNumX8Namelen(b *testing.B) {
	vr := New()
	data := makeBenchData(b.N, 64, 8)
	for _, template := range data.Templates {
		vr.Register(template)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vr.Match(data.Paths[i])
	}
}

func ExampleVarouter() {
	vr := New()
	vr.Register("/+")
	vr.Register("/dir/:var/+")

	templates, params, matched := vr.Match("/dir/val/abc")
	fmt.Printf("Templates: '%v', Params: '%v', Matched: '%t'\n", templates, params, matched)
	// Output: Templates: '[/+ /dir/:var/+]', Params: 'map[var:val]', Matched: 'true'
}
