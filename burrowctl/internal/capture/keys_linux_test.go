//go:build linux

package capture

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestOSKeysLockedBackendRefusesWithoutPrompt(t *testing.T) {
	if os.Getenv("BURROW_TEST_SECURE_BACKEND") != "1" {
		t.Skip("only run against an explicitly isolated disposable credential backend")
	}
	scope := Hash("synthetic-locked-key-fixture")
	if _, err := (OSKeys{}).Key(scope); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	bus, err := dbus.Connect(os.Getenv("DBUS_SESSION_BUS_ADDRESS"), dbus.WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()
	root := bus.Object("org.freedesktop.secrets", "/org/freedesktop/secrets")
	var collection, prompt dbus.ObjectPath
	if err := root.CallWithContext(ctx, "org.freedesktop.Secret.Service.ReadAlias", 0, "default").Store(&collection); err != nil {
		t.Fatal(err)
	}
	var locked []dbus.ObjectPath
	if err := root.CallWithContext(ctx, "org.freedesktop.Secret.Service.Lock", 0, []dbus.ObjectPath{collection}).Store(&locked, &prompt); err != nil || prompt != "/" {
		t.Fatalf("fixture could not lock: %v", err)
	}
	for _, account := range []string{scope, Hash("new-key-while-locked")} {
		started := time.Now()
		if _, err := (OSKeys{}).Key(account); err == nil {
			t.Fatal("locked key backend accepted")
		}
		if time.Since(started) > time.Second {
			t.Fatal("locked backend waited for a prompt")
		}
	}
}
