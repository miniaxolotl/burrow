package capture

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"burrow/burrowctl/internal/routing"
)

type memoryKeys struct {
	key []byte
	err error
}

func (k memoryKeys) Key(string) ([]byte, error) { return append([]byte(nil), k.key...), k.err }
func privateDirectory(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}
func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(privateDirectory(t), "synthetic-scope", memoryKeys{key: bytes.Repeat([]byte{7}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func record(id string) Record {
	return Record{ID: id, TunnelID: "tunnel", Method: "POST", Path: "/hook", Body: []byte("synthetic-private-body"), Route: routing.Route{Path: "/", Port: 3000}, ConfigDigest: "digest"}
}

func TestEncryptedStoreScopeTTLAndDeletionFence(t *testing.T) {
	s := testStore(t)
	epoch, err := s.Epoch("tunnel")
	if err != nil {
		t.Fatal(err)
	}
	r := record("one")
	if err := s.Put(r, epoch); err != nil {
		t.Fatal(err)
	}
	got, err := s.Read("one", "tunnel")
	if err != nil || !bytes.Equal(got.Body, r.Body) {
		t.Fatal("encrypted roundtrip failed", err)
	}
	if _, err := s.Read("one", "other"); err == nil {
		t.Fatal("cross-tunnel read")
	}
	files, _ := filepath.Glob(filepath.Join(s.dir, "*"))
	for _, file := range files {
		data, _ := os.ReadFile(file)
		if bytes.Contains(data, r.Body) || bytes.Contains(data, []byte("/hook")) {
			t.Fatalf("plaintext at rest in %s", filepath.Base(file))
		}
	}
	if err := s.Delete("one", "tunnel"); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(r, epoch); err == nil {
		t.Fatal("stale writer revived tombstone")
	}
	if _, err := s.Read("one", "tunnel"); err == nil {
		t.Fatal("deleted read")
	}
	if err := s.Clear("tunnel"); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(record("two"), epoch); err == nil {
		t.Fatal("clear did not fence in-flight capture")
	}
}

func TestCaptureCannotOpenWithoutSecureKeyAndRejectsCipherSwap(t *testing.T) {
	if s, err := Open(privateDirectory(t), "scope", memoryKeys{err: errors.New("locked")}); err == nil {
		s.Close()
		t.Fatal("plaintext fallback")
	}
	s := testStore(t)
	epoch, _ := s.Epoch("tunnel")
	for _, id := range []string{"a", "b"} {
		if err := s.Put(record(id), epoch); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.Exec("UPDATE captures SET payload=(SELECT payload FROM captures WHERE id='a') WHERE id='b'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read("b", "tunnel"); err == nil {
		t.Fatal("cross-record ciphertext accepted")
	}
	if _, err := s.db.Exec("UPDATE captures SET expires_ms=0 WHERE id='a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read("a", "tunnel"); err == nil {
		t.Fatal("expired payload readable")
	}
}

func TestCaptureMetadataIsBoundedAndUnknownClearDoesNotCreateScope(t *testing.T) {
	s := testStore(t)
	if err := s.Clear("unknown"); err == nil {
		t.Fatal("unknown clear created a scope")
	}
	if _, err := s.db.Exec("WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<2048) INSERT INTO epochs SELECT printf('%d',x),0 FROM n"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Epoch("new"); err == nil {
		t.Fatal("unbounded metadata creation")
	}
}

func TestEncryptedRecordRejectsTunnelMetadataSwap(t *testing.T) {
	s := testStore(t)
	epoch, _ := s.Epoch("tunnel")
	if err := s.Put(record("one"), epoch); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("UPDATE captures SET tunnel=? WHERE id='one'", Hash("other")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read("one", "other"); err == nil {
		t.Fatal("ciphertext authorized using swapped tunnel metadata")
	}
}
