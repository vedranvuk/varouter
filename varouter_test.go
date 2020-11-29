// Copyright 2020 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package varouter

import (
	"fmt"
	"sort"
	"testing"

	"github.com/vedranvuk/randomex"
)

// debugPrintElements recursively prints out all elements in e.
func debugPrintElements(e *element) {
	for key, val := range e.subs {
		fmt.Printf(`Sub:  '%s'
NumSubs:   '%d'
Template:   '%s'
IsOverride:  '%t'
IsPrefix:  '%t'
IsWildcard:  '%t'
HasVariable: '%s'
HasWildcards: '%t'

`, key, len(val.subs), val.template, val.isoverride, val.isprefix, val.iswildcard, val.hasvariable, val.haswildcards)
		debugPrintElements(val)
	}
}

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

// DebugPrintElements prints out all elements in Varouter.
func DebugPrintElements(vr *Varouter) {
	debugPrintElements(vr.root)
	tmpls := vr.DefinedTemplates()
	sort.Strings(tmpls)
	fmt.Println(tmpls)
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

type RegistrationTest struct {
	Pattern     string
	ExpectErr   bool
	FailMessage string
}

var RegistrationTests = []RegistrationTest{
	{"", true, "Failed detecting empty pattern."},
	{"h", true, "Failed detecting invalid pattern."},
	{"/", false, ""},
	{"/:", true, "Failed detecting invalid variable pattern."},
	{"/home", false, ""},
	{"/home*", false, ""},
	{"/home/*", false, ""},
	{":cantregisterthis", true, "Failed detecting variable being registered on a path level with registered elements."},
	{":cantregisterthis/", true, "Failed detecting variable being registered on a path level with registered elements."},
	{"/home/users", false, ""},
	{"/home/users/:user", false, ""},
	{"/home/users/:username", true, "Failed detecting variable being registered on a path level with registered elements."},
	{"/home/users/vedran", true, "Failed detecting variable being registered on a path level with registered elements."},
	{"/home/users/vedran/.config", true, "Failed detecting variable being registered on a path level with registered elements."},
}

func TestRegister(t *testing.T) {
	vr := New()
	for _, test := range RegistrationTests {
		err := vr.Register(test.Pattern)
		if test.ExpectErr != (err != nil) {
			if test.FailMessage != "" {
				t.Fatal(test.FailMessage)
			}
			t.Fatal(err)
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
			"/home/+",
			"/home/vedran/+",
		},
		Matches: []Match{
			{
				Path:              "/home",
				ExpectedPatterns:  []string{},
				Expectedvariables: nil,
				ExpectedMatch:     false,
			},
			{
				Path:              "/home/vedran",
				ExpectedPatterns:  []string{"/home/+"},
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
			"/",
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
			{
				Path:              "/file",
				ExpectedPatterns:  []string{"!/file"},
				Expectedvariables: nil,
				ExpectedMatch:     true,
			},
		},
	},
}

func TestOverrideMatch(t *testing.T) {
	RunMatchTests(t, MatchOverrideTests)
}

var MatchCombinedTests1 = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/home+",
			"/home/:user",
			"/home/:user/",
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
