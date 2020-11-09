// Copyright 2020 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package varouter

import (
	"fmt"
	"testing"

	"github.com/vedranvuk/randomex"
)

// debugPrintElements recursively prints out all elements in e.
func debugPrintElements(e *element) {
	for key, val := range e.subs {
		fmt.Printf(`Template:  '%s'
Element:   '%s'
Container: '%s'
Wildcard:  '%t'
NumSubs:   '%d'

`, val.template, key, val.container, val.wildcard, len(val.subs))
		debugPrintElements(val)
	}
}

func FailDaMatchTest(whichOne *testing.T, match Match, result []string, ph PlaceholderMap, wozExpected bool) {
	whichOne.Fatalf(`
	Match failed,
	Expected: 
	  MatchTest:    '%#+v'
	Got:      
	  Patterns:     '%#+v'
	  Placeholders: '%#+v'
	  Matched:      '%#+v'`, match, result, ph, wozExpected)
}

// DebugPrintElements prints out all elements in Varouter.
func DebugPrintElements(vr *Varouter) { debugPrintElements(vr.root) }

type RegistrationTest struct {
	Pattern     string
	ExpectErr   bool
	FailMessage string
}

var RegistrationTests = []RegistrationTest{
	{"", true, "Failed detecting empty pattern."},
	{"h", true, "Failed detecting invalid pattern."},
	{"/", false, ""},
	{"/:", true, "Failed detecting invalid placeholder pattern."},
	{"/home", false, ""},
	{"/home*", false, ""},
	{"/home/*", false, ""},
	{":cantregisterthis", true, "Failed detecting Placeholder being registered on a path level with registered elements."},
	{":cantregisterthis/", true, "Failed detecting Placeholder being registered on a path level with registered elements."},
	{"/home/users", false, ""},
	{"/home/users/:user", false, ""},
	{"/home/users/:username", true, "Failed detecting Placeholder being registered on a path level with registered elements."},
	{"/home/users/vedran", true, "Failed detecting Placeholder being registered on a path level with registered elements."},
	{"/home/users/vedran/.config", true, "Failed detecting Placeholder being registered on a path level with registered elements."},
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

type Match struct {
	Path                 string
	ExpectedPatterns     []string
	ExpectedPlaceholders PlaceholderMap
	ExpectedMatch        bool
}

type MatchTest struct {
	RegisteredPatterns []string
	Matches            []Match
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
				Path:                 "/",
				ExpectedPatterns:     []string{"/"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "/home",
				ExpectedPatterns:     []string{"/home"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "/home/",
				ExpectedPatterns:     []string{"/home/"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "/home/users",
				ExpectedPatterns:     []string{"/home/users"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "/home/users/",
				ExpectedPatterns:     []string{"/home/users/"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "/home/users//",
				ExpectedPatterns:     []string{"/home/users//"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "/home//users/",
				ExpectedPatterns:     []string{"/home//users/"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "///home///users///",
				ExpectedPatterns:     []string{"///home///users///"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
		},
	},
}

func TestExactMatch(t *testing.T) {
	for _, matchtest := range MatchExactTests {
		vr := New()
		for _, pattern := range matchtest.RegisteredPatterns {
			if err := vr.Register(pattern); err != nil {
				t.Fatal(err)
			}
		}

		for _, match := range matchtest.Matches {

			patterns, placeholders, matched := vr.Match(match.Path)
			if matched != match.ExpectedMatch {
				DebugPrintElements(vr)
				FailDaMatchTest(t, match, patterns, placeholders, matched)
			}

			for _, expectedpattern := range match.ExpectedPatterns {
				found := false
				for i := 0; i < len(patterns); i++ {
					if patterns[i] == expectedpattern {
						found = true
					}
				}
				if !found {
					DebugPrintElements(vr)
					FailDaMatchTest(t, match, patterns, placeholders, matched)
				}
			}

			for expectedplaceholderkey, expectedplaceholderval := range match.ExpectedPlaceholders {

				if val, ok := placeholders[expectedplaceholderkey]; !ok || expectedplaceholderval != val {
					DebugPrintElements(vr)
					FailDaMatchTest(t, match, patterns, placeholders, matched)
				}
			}

		}
	}
}

var MatchPlaceholderTests = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/home/users/:username",
			"/home/users/:username/",
			"/home/users/:username/.config/:application",
			"/home/users/:username/.config/:application/",
		},
		Matches: []Match{
			{
				Path:                 "/home/users/vedran",
				ExpectedPatterns:     []string{"/home/users/:username"},
				ExpectedPlaceholders: PlaceholderMap{"username": "vedran"},
				ExpectedMatch:        true,
			},
			{
				Path:                 "/home/users/vedran/",
				ExpectedPatterns:     []string{"/home/users/:username/"},
				ExpectedPlaceholders: PlaceholderMap{"username": "vedran"},
				ExpectedMatch:        true,
			},
			{
				Path:                 "/home/users/vedran/.config/myapp",
				ExpectedPatterns:     []string{"/home/users/:username/.config/:application"},
				ExpectedPlaceholders: PlaceholderMap{"username": "vedran", "application": "myapp"},
				ExpectedMatch:        true,
			},
			{
				Path:                 "/home/users/vedran/.config/myapp/",
				ExpectedPatterns:     []string{"/home/users/:username/.config/:application/"},
				ExpectedPlaceholders: PlaceholderMap{"username": "vedran", "application": "myapp"},
				ExpectedMatch:        true,
			},
		},
	},
}

func TestPlaceholderMatch(t *testing.T) {
	for _, matchtest := range MatchPlaceholderTests {
		vr := New()
		for _, pattern := range matchtest.RegisteredPatterns {
			if err := vr.Register(pattern); err != nil {
				t.Fatal(err)
			}
		}

		for _, match := range matchtest.Matches {

			patterns, placeholders, matched := vr.Match(match.Path)
			if matched != match.ExpectedMatch {
				DebugPrintElements(vr)
				FailDaMatchTest(t, match, patterns, placeholders, matched)
			}

			for _, expectedpattern := range match.ExpectedPatterns {
				found := false
				for i := 0; i < len(patterns); i++ {
					if patterns[i] == expectedpattern {
						found = true
					}
				}
				if !found {
					DebugPrintElements(vr)
					FailDaMatchTest(t, match, patterns, placeholders, matched)
				}
			}

			for expectedplaceholderkey, expectedplaceholderval := range match.ExpectedPlaceholders {

				if val, ok := placeholders[expectedplaceholderkey]; !ok || expectedplaceholderval != val {
					DebugPrintElements(vr)
					FailDaMatchTest(t, match, patterns, placeholders, matched)
				}
			}

		}
	}
}

var MatchWildcardTests = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/users/+",
			"/us+++/+",
			"/users/+/.config/+",
			"/users/+/.config/+nope",
		},
		Matches: []Match{
			{
				Path:                 "/users",
				ExpectedPatterns:     []string{},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        false,
			},
			{
				Path:                 "/users/",
				ExpectedPatterns:     []string{"/users/+"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "/users/vedran",
				ExpectedPatterns:     []string{"/users/+"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "/users/vedran/",
				ExpectedPatterns:     []string{"/users/+"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "/users/vedran/.config",
				ExpectedPatterns:     []string{"/users/+"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path:                 "/users/vedran/.config/",
				ExpectedPatterns:     []string{"/users/+"},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
			{
				Path: "/users/vedran/.config/anything",
				ExpectedPatterns: []string{
					"/users/+",
					"/users/+/.config/+",
				},
				ExpectedPlaceholders: nil,
				ExpectedMatch:        true,
			},
		},
	},
}

func TestWildcardMatch(t *testing.T) {
	for _, matchtest := range MatchWildcardTests {
		vr := New()
		for _, pattern := range matchtest.RegisteredPatterns {
			if err := vr.Register(pattern); err != nil {
				t.Fatal(err)
			}
		}

		for _, match := range matchtest.Matches {

			patterns, placeholders, matched := vr.Match(match.Path)
			if matched != match.ExpectedMatch {
				DebugPrintElements(vr)
				FailDaMatchTest(t, match, patterns, placeholders, matched)
			}

			for _, expectedpattern := range match.ExpectedPatterns {
				found := false
				for i := 0; i < len(patterns); i++ {
					if patterns[i] == expectedpattern {
						found = true
					}
				}
				if !found {
					DebugPrintElements(vr)
					FailDaMatchTest(t, match, patterns, placeholders, matched)
				}
			}

			for expectedplaceholderkey, expectedplaceholderval := range match.ExpectedPlaceholders {

				if val, ok := placeholders[expectedplaceholderkey]; !ok || expectedplaceholderval != val {
					DebugPrintElements(vr)
					FailDaMatchTest(t, match, patterns, placeholders, matched)
				}
			}

		}
	}
}

var MatchCombinedTests = []MatchTest{
	{
		RegisteredPatterns: []string{
			"/home/users/:user/temp/+",
		},
		Matches: []Match{
			{
				Path:                 "/home/users/vedran/temp/file",
				ExpectedPatterns:     []string{"/home/users/:user/temp/+"},
				ExpectedPlaceholders: PlaceholderMap{"user": "vedran"},
				ExpectedMatch:        true,
			},
		},
	},
}

func TestCombined(t *testing.T) {
	for _, matchtest := range MatchCombinedTests {
		vr := New()
		for _, pattern := range matchtest.RegisteredPatterns {
			if err := vr.Register(pattern); err != nil {
				t.Fatal(err)
			}
		}

		for _, match := range matchtest.Matches {

			patterns, placeholders, matched := vr.Match(match.Path)
			if matched != match.ExpectedMatch {
				DebugPrintElements(vr)
				FailDaMatchTest(t, match, patterns, placeholders, matched)
			}

			for _, expectedpattern := range match.ExpectedPatterns {
				found := false
				for i := 0; i < len(patterns); i++ {
					if patterns[i] == expectedpattern {
						found = true
					}
				}
				if !found {
					DebugPrintElements(vr)
					FailDaMatchTest(t, match, patterns, placeholders, matched)
				}
			}

			for expectedplaceholderkey, expectedplaceholderval := range match.ExpectedPlaceholders {

				if val, ok := placeholders[expectedplaceholderkey]; !ok || expectedplaceholderval != val {
					DebugPrintElements(vr)
					FailDaMatchTest(t, match, patterns, placeholders, matched)
				}
			}

		}
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

type BenchTest struct {
	Templates []string
	Paths     []string
}

func makeRandomData(iterations int, maxPathElems, maxPathLength int) *BenchTest {

	var data *BenchTest = &BenchTest{}
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
			path += "/" + randomex.String(true, true, true, true, maxPathLength)
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
	data := makeRandomData(b.N, 8, 8)
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
	data := makeRandomData(b.N, 64, 64)
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
	data := makeRandomData(b.N, 8, 64)
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
	data := makeRandomData(b.N, 64, 8)
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
	vr.Register("/home/users/:username/+")

	templates, params, matched := vr.Match("/home/users/vedran/.config")
	fmt.Printf("Templates: '%v', Params: '%v', Matched: '%t'\n", templates, params, matched)
	// Output: Templates: '[/+ /home/users/:username/+]', Params: 'map[username:vedran]', Matched: 'true'
}
