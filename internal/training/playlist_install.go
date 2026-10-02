package training

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type playlistOwner struct {
	Version int    `json:"version"`
	PlanID  string `json:"planId"`
	Hash    string `json:"sha256"`
}

func playlistHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func atomicPlaylistFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".refleks-playlist-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Auto-install has one owned slot. Explicit user exports remain unrestricted.
// Old per-plan exports and user playlists are never deleted by name matching.
func (s *Service) Install(dir string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.state.Plan
	if p == nil {
		return "", fmt.Errorf("请先生成计划")
	}
	path := filepath.Join(dir, "Refleks-Adaptive-Current.json")
	// Ownership lives outside the game's playlist JSON schema.
	ownerPath := path + ".owner"
	var previous []byte
	if old, err := os.ReadFile(path); err == nil {
		previous = old
		var owner playlistOwner
		marker, e := os.ReadFile(ownerPath)
		if e != nil || json.Unmarshal(marker, &owner) != nil || owner.Version != 1 || owner.Hash != playlistHash(old) {
			return "", fmt.Errorf("固定列表槽位已有非本应用管理文件，未覆盖")
		}
		if owner.PlanID == p.ID {
			return path, nil
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if p.Status == "running" || p.Status == "ready" || p.Status == "paused" || p.Status == "waiting" {
		return "", fmt.Errorf("进行中的列表保持固定；请先结束本次训练再安装")
	}
	b, err := s.exportLocked()
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err = json.Unmarshal(b, &payload); err != nil {
		return "", err
	}
	payload["playlistName"] = "Refleks Adaptive Current"
	b, err = json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	if err = atomicPlaylistFile(path, b); err != nil {
		return "", err
	}
	marker, _ := json.Marshal(playlistOwner{Version: 1, PlanID: p.ID, Hash: playlistHash(b)})
	if err = atomicPlaylistFile(ownerPath, marker); err != nil {
		if previous != nil {
			_ = atomicPlaylistFile(path, previous)
		} else {
			_ = os.Remove(path)
		}
		return "", err
	}
	return path, nil
}
