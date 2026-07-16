package inifiles

import (
	"strings"
	"testing"
)

// translated from config-inifiles/t/09case.t
func TestCaseSensitive(t *testing.T) {
	t.Run("newval_and_val_correct_case", func(t *testing.T) {
		cfg := New()
		if err := cfg.NewVal("Section", "PaRaMeTeR", "Mixed Case"); err != nil {
			t.Fatalf("NewVal: %v", err)
		}
		if err := cfg.NewVal("Section", "Parameter", "Title Case"); err != nil {
			t.Fatalf("NewVal: %v", err)
		}
		if v, ok := cfg.Val("Section", "PaRaMeTeR"); !ok || v != "Mixed Case" {
			t.Errorf(`Val(Section,PaRaMeTeR) = %q ok=%v, want "Mixed Case"`, v, ok)
		}
		if v, ok := cfg.Val("Section", "Parameter"); !ok || v != "Title Case" {
			t.Errorf(`Val(Section,Parameter) = %q ok=%v, want "Title Case"`, v, ok)
		}
	})

	t.Run("sections_distinct_by_case", func(t *testing.T) {
		cfg := New()
		_ = cfg.NewVal("Section", "Parameter", "Value")
		_ = cfg.NewVal("section", "parameter", "value")
		if got := len(cfg.Sections()); got != 2 {
			t.Errorf("Sections() count = %d, want 2", got)
		}
	})

	t.Run("delete_value_case_sensitive", func(t *testing.T) {
		cfg := New()
		_ = cfg.NewVal("Section", "Parameter", "Title Case")
		_ = cfg.NewVal("Section", "parameter", "lower case")
		_ = cfg.NewVal("Section", "PARAMETER", "UPPER CASE")

		members := cfg.Parameters("Section")
		if got := len(members); got != 3 {
			t.Errorf("Parameters before delete = %d, want 3 (%v)", got, members)
		}
		if err := cfg.DelVal("Section", "PARAMETER"); err != nil {
			t.Fatalf("DelVal: %v", err)
		}
		members = cfg.Parameters("Section")
		if got := len(members); got != 2 {
			t.Errorf("Parameters after delete = %d, want 2 (%v)", got, members)
		}
		if !containsExact(members, "Parameter") {
			t.Errorf("Parameter exists; members=%v", members)
		}
		if !containsExact(members, "parameter") {
			t.Errorf("parameter exists; members=%v", members)
		}
	})

	t.Run("parameters_case_check", func(t *testing.T) {
		cfg := New()
		_ = cfg.NewVal("Section", "PaRaMeTeR", "Mixed Case")
		_ = cfg.NewVal("Section", "Parameter", "Title Case")
		_ = cfg.NewVal("SECTION", "Parameter", "N/A")

		if list := cfg.Parameters("SECTION"); len(list) != 1 || list[0] != "Parameter" {
			t.Errorf(`Parameters(SECTION) = %v, want ["Parameter"]`, list)
		}
		list := cfg.Parameters("Section")
		if len(list) != 2 {
			t.Errorf("Parameters(Section) = %v, want 2 entries", list)
		}
		joined := strings.Join(list, " ")
		if !strings.Contains(joined, "PaRaMeTeR") {
			t.Errorf("Parameters(Section)=%v missing PaRaMeTeR", list)
		}
		if !strings.Contains(joined, "Parameter") {
			t.Errorf("Parameters(Section)=%v missing Parameter", list)
		}
	})

	t.Run("groups_case_sensitive", func(t *testing.T) {
		cfg := New()
		_ = cfg.NewVal("interface foo", "parameter", "foo")
		_ = cfg.NewVal("interface bar", "parameter", "bar")
		_ = cfg.NewVal("INTERFACE blurgle", "parameter", "flurgle")

		if got := len(cfg.Groups()); got != 2 {
			t.Errorf("Groups() = %d, want 2 (%v)", got, cfg.Groups())
		}
		if members := cfg.GroupMembers("Interface"); len(members) != 0 {
			t.Errorf(`GroupMembers("Interface") = %v, want empty (wrong case)`, members)
		}
		members := cfg.GroupMembers("interface")
		if len(members) != 2 {
			t.Errorf("GroupMembers(interface) = %v, want 2 members", members)
		}
		if !containsExact(members, "interface foo") {
			t.Errorf("GroupMembers(interface)=%v missing interface foo", members)
		}
		if !containsExact(members, "interface bar") {
			t.Errorf("GroupMembers(interface)=%v missing interface bar", members)
		}
	})
}

// translated from config-inifiles/t/09case.t (case-insensitive section)
func TestCaseInsensitive_NewValOverwrites(t *testing.T) {
	cfg := New(WithNoCase(true))
	if err := cfg.NewVal("Section", "PaRaMeTeR", "Mixed Case"); err != nil {
		t.Fatalf("NewVal: %v", err)
	}
	if err := cfg.NewVal("Section", "Parameter", "Title Case"); err != nil {
		t.Fatalf("NewVal: %v", err)
	}
	vs, ok := cfg.Vals("Section", "parameter")
	if !ok {
		t.Fatalf("Vals(Section,parameter) not found")
	}
	want := []string{"Title Case"}
	if len(vs) != 1 || vs[0] != want[0] {
		t.Errorf("Vals(Section,parameter) = %v, want %v (second newval overwrites first)", vs, want)
	}
}

// translated from config-inifiles/t/09case.t test 7
func TestCaseInsensitive_GroupMembersInTestIni(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"), WithNoCase(true))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	got := cfg.GroupMembers("GrOuP")
	want := []string{"group member one", "group member two", "group member three"}
	if len(got) != len(want) {
		t.Fatalf("GroupMembers(GrOuP) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("GroupMembers(GrOuP)[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// translated from config-inifiles/t/09case.t tests 8 & 9
func TestCaseInsensitive_ParameterAndSectionLookup(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"), WithNoCase(true), WithDefault("test1"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	if v, ok := cfg.Val("test2", "FOUR"); !ok || v != "value4" {
		t.Errorf(`Val(test2,FOUR) = %q ok=%v, want "value4" (case-insensitive param)`, v, ok)
	}
	if v, ok := cfg.Val("TEST2", "four"); !ok || v != "value4" {
		t.Errorf(`Val(TEST2,four) = %q ok=%v, want "value4" (case-insensitive section)`, v, ok)
	}
	if v, ok := cfg.Val("mixedcasesect", "mixedcaseparam"); !ok || v != "MixedCaseVal" {
		t.Errorf(`Val(mixedcasesect,mixedcaseparam) = %q ok=%v, want "MixedCaseVal"`, v, ok)
	}
}

// translated from config-inifiles/t/24case-sensitive-exists.t
func TestCaseSensitiveExists_NocaseMatches(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "case-sensitive.ini"), WithNoCase(true))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	if !cfg.Exists("FOO", "BAR") {
		t.Errorf(`Exists(FOO,BAR) = false under nocase, want true`)
	}
	if v, ok := cfg.Val("FOO", "BAR"); !ok || v != "goodness" {
		t.Errorf(`Val(FOO,BAR) = %q ok=%v, want "goodness"`, v, ok)
	}
}

// translated from config-inifiles/t/16case-sensitive-default.t
func TestCaseSensitiveDefault_Nocase(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "case-sensitive-default.ini"), WithDefault("Common"), WithNoCase(true))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	got := cfg.ValWithDefault("MyScript", "stopfile", "not defined")
	if got != "myscript-stop" {
		t.Errorf(`ValWithDefault(MyScript,stopfile) = %q, want "myscript-stop"`, got)
	}
}

// containsExact reports whether s is present as an exact element of list.
func containsExact(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}
