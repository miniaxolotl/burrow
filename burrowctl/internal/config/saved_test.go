package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSavedAliasPersistsAttemptBeforeAdmissionAndScopesIdentity(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	alias, err := OpenAlias("localhost:25701", "owner", "webhook")
	if err != nil {
		t.Fatal(err)
	}
	id := alias.Record().TunnelID
	attempt, err := alias.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenAlias("localhost:25701", "owner", "webhook"); err == nil {
		t.Fatal("simultaneous local alias use")
	}
	alias.Close()
	alias, err = OpenAlias("localhost:25701", "owner", "webhook")
	if err != nil {
		t.Fatal(err)
	}
	if alias.Record().TunnelID != id {
		t.Fatal("restart replaced candidate")
	}
	recovered, err := alias.Prepare()
	if err != nil || recovered.Nonce != attempt.Nonce {
		t.Fatal("uncertain attempt not reconciled")
	}
	if err := alias.Acknowledge("g1", "http://saved.example"); err != nil {
		t.Fatal(err)
	}
	next, err := alias.Prepare()
	if err != nil || next.Expected != "g1" || next.Nonce == attempt.Nonce {
		t.Fatal("next generation not conditional")
	}
	alias.Close()
	for _, scope := range [][2]string{{"localhost:25702", "owner"}, {"localhost:25701", "other"}} {
		other, err := OpenAlias(scope[0], scope[1], "webhook")
		if err != nil {
			t.Fatal(err)
		}
		if other.Record().TunnelID == id {
			t.Fatal("cross-scope reuse")
		}
		other.Close()
	}
}

func TestSavedAliasRejectsUnsafeNameAndCorruption(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, name := range []string{"../escape", "", "a/b", "UPPER", " spaces "} {
		if a, err := OpenAlias("localhost", "owner", name); err == nil {
			a.Close()
			t.Fatalf("accepted %q", name)
		}
	}
	a, err := OpenAlias("localhost", "owner", "saved")
	if err != nil {
		t.Fatal(err)
	}
	a.Close()
	files, _ := filepath.Glob(filepath.Join(os.Getenv("HOME"), ".config", "burrow", "saved", "*.json"))
	if len(files) != 1 {
		t.Fatal(files)
	}
	if err := os.WriteFile(files[0], []byte(`{"version":999}`), 0600); err != nil {
		t.Fatal(err)
	}
	if a, err := OpenAlias("localhost", "owner", "saved"); err == nil {
		a.Close()
		t.Fatal("rewrote incompatible state")
	}
	data, _ := os.ReadFile(files[0])
	if string(data) != `{"version":999}` {
		t.Fatal("corruption discarded")
	}
}

func TestSavedClosePreventsStaleAcknowledgementAndNeedsConfirmedReplacement(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a, err := OpenAlias("localhost", "owner", "saved")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	id := a.Record().TunnelID
	if _, err := a.Prepare(); err != nil {
		t.Fatal(err)
	}
	if err := MarkSaved("localhost", "owner", id, "uncertain"); err != nil {
		t.Fatal(err)
	}
	if err := a.Acknowledge("g1", "url"); err == nil {
		t.Fatal("stale writer restored uncertain close")
	}
	if err := a.Replace(true); err == nil {
		t.Fatal("uncertain close replaced")
	}
	if err := MarkSaved("localhost", "owner", id, "closed"); err != nil {
		t.Fatal(err)
	}
	if err := a.Replace(false); err == nil {
		t.Fatal("unapproved replacement")
	}
	if err := a.Replace(true); err != nil {
		t.Fatal(err)
	}
	if a.Record().TunnelID == id {
		t.Fatal("replacement resurrected revoked ID")
	}
	if err := MarkSaved("localhost", "owner", id, "closed"); err != nil {
		t.Fatal(err)
	}
	if a.Record().State != "saved" {
		t.Fatal("stale closure overwrote replacement")
	}
}

func TestSavedAliasRejectsSymlinkAndPublicPermissions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a, err := OpenAlias("localhost", "owner", "saved")
	if err != nil {
		t.Fatal(err)
	}
	a.Close()
	files, _ := filepath.Glob(filepath.Join(os.Getenv("HOME"), ".config", "burrow", "saved", "*.json"))
	path := files[0]
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if a, err := OpenAlias("localhost", "owner", "saved"); err == nil {
		a.Close()
		t.Fatal("public state accepted")
	}
	if err := os.Rename(path, path+".original"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path+".original", path); err != nil {
		t.Fatal(err)
	}
	if a, err := OpenAlias("localhost", "owner", "saved"); err == nil {
		a.Close()
		t.Fatal("symlink accepted")
	}
}
