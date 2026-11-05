package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"burrow/burrowctl/internal"
	"burrow/burrowctl/internal/localstate"
	"burrow/protocol"
)

var aliasPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func ValidateAlias(name string) error {
	if !aliasPattern.MatchString(name) {
		return errors.New("name must contain 1-64 lowercase letters, numbers, underscores or hyphens")
	}
	return nil
}

type SavedAlias struct {
	Version    int       `json:"version"`
	Scope      string    `json:"scope"`
	Name       string    `json:"name"`
	TunnelID   string    `json:"tunnel_id"`
	Attachment string    `json:"attachment"`
	Generation string    `json:"generation"`
	Nonce      string    `json:"nonce"`
	Pending    bool      `json:"pending"`
	URL        string    `json:"url"`
	State      string    `json:"state"`
	ObservedAt time.Time `json:"observed_at"`
	Checksum   string    `json:"checksum"`
}

type Alias struct {
	mu                sync.Mutex
	path, scope, name string
	use               *os.File
}
type AttachmentAttempt struct{ ID, Expected, Nonce string }

func savedScope(server, owner string) (string, string, error) {
	if owner == "" {
		return "", "", errors.New("existing owner credential required")
	}
	origin, err := internal.ServerURL(server)
	if err != nil {
		return "", "", err
	}
	h := sha256.Sum256([]byte(origin.String() + "\x00" + owner))
	scope := hex.EncodeToString(h[:])
	base, err := configDir()
	if err != nil {
		return "", "", err
	}
	// Reject symlinks for each Burrow-controlled directory rather than chmodding
	// a symlink target. The user's home and .config are not feature-owned.
	if err := localstate.Directory(base); err != nil {
		return "", "", err
	}
	dir := filepath.Join(base, "saved")
	if err := localstate.Directory(dir); err != nil {
		return "", "", err
	}
	return dir, scope, nil
}

func OpenAlias(server, owner, name string) (*Alias, error) {
	if err := ValidateAlias(name); err != nil {
		return nil, err
	}
	dir, scope, err := savedScope(server, owner)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256([]byte(name))
	path := filepath.Join(dir, scope+"-"+hex.EncodeToString(h[:])+".json")
	lock, err := localstate.Lock(path+".use.lock", false)
	if err != nil {
		return nil, err
	}
	a := &Alias{path: path, scope: scope, name: name, use: lock}
	err = a.update(func(r *SavedAlias) error {
		if r.Version != 0 {
			return nil
		}
		*r = SavedAlias{Version: 1, Scope: scope, Name: name, TunnelID: protocol.RandomTunnelID(), Attachment: rand.Text(), State: "saved", ObservedAt: time.Now().UTC()}
		return nil
	})
	if err != nil {
		lock.Close()
		return nil, err
	}
	return a, nil
}

func checksum(r SavedAlias) string {
	r.Checksum = ""
	data, _ := json.Marshal(r)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
func (a *Alias) read() (SavedAlias, error) {
	var r SavedAlias
	data, err := localstate.Read(a.path, 16384)
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return r, errors.New("corrupt saved alias; preserve the file for recovery")
	}
	if r.Version != 1 || r.Scope != a.scope || r.Name != a.name || r.TunnelID == "" || r.Attachment == "" || r.Checksum != checksum(r) {
		return r, errors.New("incompatible or corrupt saved alias; preserve the file for recovery")
	}
	return r, nil
}
func (a *Alias) update(fn func(*SavedAlias) error) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	lock, err := localstate.Lock(a.path+".state.lock", true)
	if err != nil {
		return err
	}
	defer lock.Close()
	r, err := a.read()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := fn(&r); err != nil {
		return err
	}
	r.Checksum = checksum(r)
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return localstate.Write(a.path, data)
}
func (a *Alias) Record() SavedAlias { a.mu.Lock(); defer a.mu.Unlock(); r, _ := a.read(); return r }
func (a *Alias) Prepare() (AttachmentAttempt, error) {
	var attempt AttachmentAttempt
	err := a.update(func(r *SavedAlias) error {
		if r.State != "saved" {
			return errors.New("saved alias is closed, conflicted or uncertain; reconcile before reuse")
		}
		if !r.Pending {
			r.Nonce = rand.Text()
			r.Pending = true
		}
		attempt = AttachmentAttempt{r.Attachment, r.Generation, r.Nonce}
		return nil
	})
	return attempt, err
}
func (a *Alias) Acknowledge(generation, url string) error {
	return a.update(func(r *SavedAlias) error {
		if r.State != "saved" || !r.Pending || generation == "" {
			return errors.New("alias changed during admission")
		}
		r.Generation = generation
		r.URL = url
		r.Pending = false
		r.ObservedAt = time.Now().UTC()
		return nil
	})
}
func (a *Alias) Close() error { return a.use.Close() }

func (a *Alias) SetState(state string) error {
	return a.update(func(r *SavedAlias) error { r.State = state; r.ObservedAt = time.Now().UTC(); return nil })
}

func (a *Alias) Replace(yes bool) error {
	return a.update(func(r *SavedAlias) error {
		if !yes || r.State != "closed" {
			return errors.New("replacement requires a confirmed closed alias and --yes")
		}
		r.TunnelID = protocol.RandomTunnelID()
		r.Attachment = rand.Text()
		r.Generation = ""
		r.Nonce = ""
		r.Pending = false
		r.URL = ""
		r.State = "saved"
		r.ObservedAt = time.Now().UTC()
		return nil
	})
}

type SavedView struct {
	Name       string    `json:"name"`
	TunnelID   string    `json:"tunnel_id"`
	URL        string    `json:"url"`
	State      string    `json:"state"`
	ObservedAt time.Time `json:"observed_at"`
}

func ListSaved(server, owner string) ([]SavedView, error) {
	dir, scope, err := savedScope(server, owner)
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(dir, scope+"-*.json"))
	if err != nil {
		return nil, err
	}
	items := make([]SavedView, 0, len(files))
	for _, path := range files {
		data, err := localstate.Read(path, 16384)
		if err != nil {
			return nil, err
		}
		var candidate SavedAlias
		if json.Unmarshal(data, &candidate) != nil {
			return nil, errors.New("corrupt saved registry")
		}
		a := &Alias{path: path, scope: scope, name: candidate.Name}
		r, err := a.read()
		if err != nil {
			return nil, err
		}
		items = append(items, SavedView{r.Name, r.TunnelID, r.URL, r.State, r.ObservedAt})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

// Close updates do not acquire the usage lock: another terminal must be able to
// revoke an active alias. The per-record state lock fences acknowledgement writes.
func MarkSaved(server, owner, id, state string) error {
	items, err := ListSaved(server, owner)
	if err != nil {
		return err
	}
	dir, scope, err := savedScope(server, owner)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.TunnelID != id {
			continue
		}
		h := sha256.Sum256([]byte(item.Name))
		a := &Alias{path: filepath.Join(dir, scope+"-"+hex.EncodeToString(h[:])+".json"), scope: scope, name: item.Name}
		if err := a.update(func(r *SavedAlias) error {
			if r.TunnelID == id {
				r.State = state
				r.ObservedAt = time.Now().UTC()
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
