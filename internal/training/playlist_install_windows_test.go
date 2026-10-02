package training

import (
	"bytes"
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsLockedPlaylistPreservesOwnedSlot(t *testing.T) {
	s, _ := templateFixture(t)
	dir := t.TempDir()
	if _, err := s.Generate(defaults(), nil); err != nil {
		t.Fatal(err)
	}
	path, err := s.Install(dir)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	marker, _ := os.ReadFile(path + ".owner")
	if _, err = s.Generate(defaults(), nil); err != nil {
		t.Fatal(err)
	}
	// Simulate a game retaining the playlist without delete sharing.
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Install(dir); err == nil {
		windows.CloseHandle(handle)
		t.Fatal("locked playlist replacement unexpectedly succeeded")
	}
	if err = windows.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	ownerAfter, _ := os.ReadFile(path + ".owner")
	if !bytes.Equal(before, after) || !bytes.Equal(marker, ownerAfter) {
		t.Fatal("failed replacement changed playlist or owner")
	}
	if _, err = s.Install(dir); err != nil {
		t.Fatal("slot cannot be replaced after game releases it", err)
	}
}
