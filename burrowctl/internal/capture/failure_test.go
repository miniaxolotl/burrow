package capture

import (
	"bytes"
	"strconv"
	"testing"
)

func TestFullSQLiteAndReadOnlyOutageCannotPublishOrAcceptWork(t *testing.T) {
	s := testStore(t)
	epoch, _ := s.Epoch("tunnel")
	if err := s.Put(record("small"), epoch); err != nil {
		t.Fatal(err)
	}
	var pages int
	if err := s.db.QueryRow("PRAGMA page_count").Scan(&pages); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("PRAGMA max_page_count=" + strconv.Itoa(pages)); err != nil {
		t.Fatal(err)
	}
	large := record("large")
	large.Body = bytes.Repeat([]byte{42}, MaxBody)
	if err := s.Put(large, epoch); err == nil {
		t.Fatal("full store accepted publication")
	}
	if _, err := s.Read("large", "tunnel"); err == nil {
		t.Fatal("partial payload became replayable")
	}
	if _, err := s.db.Exec("PRAGMA query_only=ON"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Queue("small", "tunnel", "digest", "", true); err == nil {
		t.Fatal("storage outage acknowledged durable work")
	}
	if _, err := s.db.Exec("PRAGMA query_only=OFF"); err != nil {
		t.Fatal(err)
	}
	var jobs int
	if err := s.db.QueryRow("SELECT count(*) FROM jobs").Scan(&jobs); err != nil || jobs != 0 {
		t.Fatalf("failed acceptance left work: %d %v", jobs, err)
	}
}

func TestCompetingReplayWorkersAndStaleResultsAreFenced(t *testing.T) {
	s := testStore(t)
	epoch, _ := s.Epoch("tunnel")
	if err := s.Put(record("one"), epoch); err != nil {
		t.Fatal(err)
	}
	first, err := s.Queue("one", "tunnel", "digest", "", true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Queue("one", "tunnel", "digest", "", true)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := s.Claim(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Claim(second.ID); err == nil {
		t.Fatal("competing worker acquired scope")
	}
	if _, err := s.Queue("one", "tunnel", "digest", first.ID, true); err == nil {
		t.Fatal("active worker superseded")
	}
	if err := s.Intent(claimed.ID, claimed.Generation); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("UPDATE jobs SET lease_ms=0 WHERE id=?", claimed.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Status(claimed.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Claim(second.ID); err != nil {
		t.Fatal(err)
	}
	status := 200
	if _, err := s.finish(claimed, &status, ""); err == nil {
		t.Fatal("stale worker committed result")
	}
	old, _ := s.Status(claimed.ID)
	if old.State != "uncertain" {
		t.Fatal("stale result erased uncertainty")
	}
}
