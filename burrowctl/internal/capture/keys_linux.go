//go:build linux

package capture

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

// Use the supported Secret Service protocol without Unlock or Prompt calls.
// go-keyring's automatic prompt handling is deliberately not used: a locked or
// missing backend must fail, never request a password or hold startup indefinitely.
func (OSKeys) Key(scope string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	address := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if !strings.HasPrefix(address, "unix:") || strings.Contains(address, ";") {
		return nil, errors.New("an existing local OS credential session is required")
	}
	connection, err := dbus.Connect(address, dbus.WithContext(ctx))
	if err != nil {
		return nil, errors.New("secure capture backend unavailable")
	}
	defer connection.Close()
	const service = "org.freedesktop.secrets"
	const iface = "org.freedesktop.Secret.Service"
	root := connection.Object(service, "/org/freedesktop/secrets")
	var output dbus.Variant
	var session dbus.ObjectPath
	if err := root.CallWithContext(ctx, iface+".OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&output, &session); err != nil {
		return nil, errors.New("secure capture backend unavailable")
	}
	defer connection.Object(service, session).CallWithContext(ctx, "org.freedesktop.Secret.Session.Close", 0)
	attributes := map[string]string{"burrow-service": "local-capture-v1", "scope": scope}
	var unlocked, locked []dbus.ObjectPath
	if err := root.CallWithContext(ctx, iface+".SearchItems", 0, attributes).Store(&unlocked, &locked); err != nil {
		return nil, errors.New("secure key lookup unavailable")
	}
	if len(locked) > 0 || len(unlocked) > 1 {
		return nil, errors.New("capture key is locked or ambiguous; unlock it outside Burrow")
	}
	type secret struct {
		Session     dbus.ObjectPath
		Parameters  []byte
		Value       []byte
		ContentType string
	}
	read := func(item dbus.ObjectPath) ([]byte, error) {
		var value secret
		if err := connection.Object(service, item).CallWithContext(ctx, "org.freedesktop.Secret.Item.GetSecret", 0, session).Store(&value); err != nil || len(value.Value) != 32 {
			return nil, errors.New("secure capture key unavailable or locked")
		}
		return value.Value, nil
	}
	if len(unlocked) == 1 {
		return read(unlocked[0])
	}
	var collection dbus.ObjectPath
	if err := root.CallWithContext(ctx, iface+".ReadAlias", 0, "default").Store(&collection); err != nil || collection == "/" {
		return nil, errors.New("an existing unlocked OS credential collection is required")
	}
	var isLocked dbus.Variant
	if err := connection.Object(service, collection).CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.Secret.Collection", "Locked").Store(&isLocked); err != nil || isLocked.Value() != false {
		return nil, errors.New("OS credential collection is locked; unlock it outside Burrow")
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	properties := map[string]dbus.Variant{"org.freedesktop.Secret.Item.Label": dbus.MakeVariant("Burrow encrypted local captures"), "org.freedesktop.Secret.Item.Attributes": dbus.MakeVariant(attributes)}
	var item, prompt dbus.ObjectPath
	if err := connection.Object(service, collection).CallWithContext(ctx, "org.freedesktop.Secret.Collection.CreateItem", 0, properties, secret{session, []byte{}, key, "application/octet-stream"}, false).Store(&item, &prompt); err != nil {
		return nil, errors.New("secure key creation unavailable")
	}
	if prompt != "/" {
		connection.Object(service, prompt).CallWithContext(ctx, "org.freedesktop.Secret.Prompt.Dismiss", 0)
		return nil, errors.New("secure key creation requires external approval; no prompt was opened")
	}
	return read(item)
}
