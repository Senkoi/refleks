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
	tmp, err := os.CreateTemp(filepath.Dir(path), ".aimmeow-playlist-*")
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
		return "", fmt.Errorf("先让我安排一份训练列表喵。")
	}
	path := filepath.Join(dir, "AimMeow-Current.json")
	legacyPath := filepath.Join(dir, "Refleks-Adaptive-Current.json")
	active := p.Status == "running" || p.Status == "ready" || p.Status == "paused" || p.Status == "waiting"
	// A running legacy plan must keep its installed playlist unchanged.
	if active {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if old, err := os.ReadFile(legacyPath); err == nil && ownsPlaylist(legacyPath, old, p.ID) {
				return legacyPath, nil
			}
		}
	}
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
	if active {
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
	payload["playlistName"] = "AimMeow Current"
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
	// Remove only an intact slot whose ownership hash we can verify, after
	// the new slot and marker have both been published successfully.
	if old, err := os.ReadFile(legacyPath); err == nil && ownsPlaylist(legacyPath, old, "") {
		if err := os.Remove(legacyPath); err != nil {
			return path, fmt.Errorf("新列表已安装，但旧列表清理失败：%w", err)
		}
		_ = os.Remove(legacyPath + ".owner")
	}
	return path, nil
}

func ownsPlaylist(path string, content []byte, planID string) bool {
	marker, err := os.ReadFile(path + ".owner")
	var owner playlistOwner
	return err == nil && json.Unmarshal(marker, &owner) == nil && owner.Version == 1 &&
		owner.Hash == playlistHash(content) && (planID == "" || owner.PlanID == planID)
}
