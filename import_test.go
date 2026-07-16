package inifiles

import (
	"os"
	"path/filepath"
	"testing"
)

// translated from config-inifiles/t/04import.t
func TestImport(t *testing.T) {
	en, err := NewFromFile(testdata(t, "en.ini"))
	if err != nil {
		t.Fatalf("NewFromFile en.ini: %v", err)
	}
	es, err := NewFromFile(testdata(t, "es.ini"), WithImport(en))
	if err != nil {
		t.Fatalf("NewFromFile es.ini: %v", err)
	}

	cases := []struct {
		name             string
		cfg              *Config
		sect, parm, want string
	}{
		{"en x.ShortName", en, "x", "ShortName", "GENERAL"},
		{"es x.ShortName (imported)", es, "x", "ShortName", "GENERAL"},
		{"en x.LongName", en, "x", "LongName", "General Summary"},
		{"es x.LongName (overrides)", es, "x", "LongName", "Resumen general"},
		{"en m.DataName", en, "m", "DataName", "Month"},
		{"es m.DataName (overrides)", es, "m", "DataName", "Mes"},
	}
	for _, c := range cases {
		v, ok := c.cfg.Val(c.sect, c.parm)
		if !ok || v != c.want {
			t.Errorf("%s: Val(%s,%s) = %q ok=%v, want %q", c.name, c.sect, c.parm, v, ok, c.want)
		}
	}

	// test 4: import another level (ca imports es which imports en).
	ca, err := NewFromFile(testdata(t, "ca.ini"), WithImport(es))
	if err != nil {
		t.Fatalf("NewFromFile ca.ini: %v", err)
	}
	for _, c := range []struct{ sect, parm, want string }{
		{"x", "ShortName", "GENERAL"},
		{"x", "LongName", "Resum general"},
		{"m", "DataName", "Mes"},
		{"m", "LongName", "Resum mensual"},
	} {
		v, ok := ca.Val(c.sect, c.parm)
		if !ok || v != c.want {
			t.Errorf("ca Val(%s,%s) = %q ok=%v, want %q", c.sect, c.parm, v, ok, c.want)
		}
	}

	// test 5/6: importing a hand-built (file-less) object into a file-based one.
	a := New()
	if err := a.AddSection("alpha"); err != nil {
		t.Fatalf("AddSection alpha: %v", err)
	}
	if err := a.AddSection("x"); err != nil {
		t.Fatalf("AddSection x: %v", err)
	}
	if err := a.NewVal("x", "x", "1"); err != nil {
		t.Fatalf("NewVal x/x: %v", err)
	}
	if err := a.NewVal("x", "LongName", "1"); err != nil {
		t.Fatalf("NewVal x/LongName: %v", err)
	}
	if err := a.NewVal("m", "z", "1"); err != nil {
		t.Fatalf("NewVal m/z: %v", err)
	}
	for _, c := range []struct{ sect, parm, want string }{
		{"x", "x", "1"},
		{"x", "LongName", "1"},
		{"m", "z", "1"},
	} {
		v, ok := a.Val(c.sect, c.parm)
		if !ok || v != c.want {
			t.Errorf("a Val(%s,%s) = %q ok=%v, want %q", c.sect, c.parm, v, ok, c.want)
		}
	}

	b, err := NewFromFile(testdata(t, "ca.ini"), WithImport(a))
	if err != nil {
		t.Fatalf("NewFromFile ca.ini with import: %v", err)
	}
	for _, c := range []struct{ sect, parm, want string }{
		{"x", "LongName", "Resum general"},
		{"x", "x", "1"},
		{"m", "z", "1"},
		{"m", "LongName", "Resum mensual"},
	} {
		v, ok := b.Val(c.sect, c.parm)
		if !ok || v != c.want {
			t.Errorf("b Val(%s,%s) = %q ok=%v, want %q", c.sect, c.parm, v, ok, c.want)
		}
	}
}

// makeEmptyFile creates an empty file in t.TempDir() and returns its path.
func makeEmptyFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "empty.ini")
	if err := os.WriteFile(p, nil, 0644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}
	return p
}

// assertImportedParam asserts cfg.Val(section,param) == want.
func assertImportedParam(t *testing.T, cfg *Config, sect, parm, want string) {
	t.Helper()
	v, ok := cfg.Val(sect, parm)
	if !ok || v != want {
		t.Errorf("Val(%s,%s) = %q ok=%v, want %q", sect, parm, v, ok, want)
	}
}

// translated from config-inifiles/t/35reload-config-no-file.t
func TestReloadConfigNoFile(t *testing.T) {
	// block 1: import + filename set after construction; two reloads.
	cfgNoFile := New(WithAllowEmpty(true))
	if err := cfgNoFile.NewVal("section", "param", "1"); err != nil {
		t.Fatalf("NewVal: %v", err)
	}
	emptyFile := makeEmptyFile(t)

	cfg := New(WithImport(cfgNoFile), WithAllowEmpty(true))
	// NOTE: Perl's new(-import=>x) applies the import immediately; this
	// port's New() only applies Import on Reload(), so trigger it once.
	if err := cfg.Reload(); err != nil {
		t.Fatalf("Reload to apply import: %v", err)
	}
	assertImportedParam(t, cfg, "section", "param", "1")
	if err := cfg.SetFileName(emptyFile); err != nil {
		t.Fatalf("SetFileName: %v", err)
	}
	if err := cfg.Reload(); err != nil {
		t.Fatalf("Reload #1: %v", err)
	}
	assertImportedParam(t, cfg, "section", "param", "1")
	if err := cfg.Reload(); err != nil {
		t.Fatalf("Reload #2: %v", err)
	}
	assertImportedParam(t, cfg, "section", "param", "1")

	// block 2: constructor with file + import.
	cfg2, err := NewFromFile(emptyFile, WithImport(cfgNoFile), WithAllowEmpty(true))
	if err != nil {
		t.Fatalf("NewFromFile empty with import: %v", err)
	}
	assertImportedParam(t, cfg2, "section", "param", "1")

	// block 3: fresh no-file config imported into file-based; reload keeps it.
	cfgNoFile2 := New(WithAllowEmpty(true))
	if err := cfgNoFile2.NewVal("section", "param", "1"); err != nil {
		t.Fatalf("NewVal: %v", err)
	}
	cfg3, err := NewFromFile(emptyFile, WithImport(cfgNoFile2), WithAllowEmpty(true))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	assertImportedParam(t, cfg3, "section", "param", "1")
	if err := cfg3.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	assertImportedParam(t, cfg3, "section", "param", "1")

	// block 4: import a config written to one file, with an unsaved param,
	// then re-read a different (empty) file; imported param survives reload.
	configFile := New(WithAllowEmpty(true))
	if err := configFile.SetFileName(filepath.Join(t.TempDir(), "written.ini")); err != nil {
		t.Fatalf("SetFileName: %v", err)
	}
	if err := configFile.NewVal("section", "param2", "1"); err != nil {
		t.Fatalf("NewVal param2: %v", err)
	}
	if err := configFile.Rewrite(); err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	if err := configFile.NewVal("section", "param", "1"); err != nil {
		t.Fatalf("NewVal param: %v", err)
	}
	cfg4, err := NewFromFile(emptyFile, WithImport(configFile), WithAllowEmpty(true))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	assertImportedParam(t, cfg4, "section", "param", "1")
	if err := cfg4.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	assertImportedParam(t, cfg4, "section", "param", "1")
}
