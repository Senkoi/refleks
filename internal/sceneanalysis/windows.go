package sceneanalysis

import "math"

func weaponWindows(root, c sceSection, ss []sceSection) []Fact {
	out := []Fact{}
	pi, ok := resolveIndex(ss, "Character Profile", root.value("PlayerProfile"))
	if !ok {
		return out
	}
	for _, ref := range tokens(ss[pi].value("WeaponProfileNames")) {
		start := len(out)
		wi, ok := resolveIndex(ss, "Weapon Profile", ref)
		if !ok {
			continue
		}
		w := ss[wi]
		damage := numberFact(w, "DamagePerShot", "damage per shot")
		dt := numberFact(w, "TimeBetweenShots", "configured seconds")
		shots := numberFact(w, "ShotsPerClick", "shots")
		health := numberFact(c, "MaxHealth", "health")
		regen := numberFact(c, "HealthRegenPerSec", "health per configured second")
		chargeFields := source(w, "IsChargeWeapon")
		meshFields := source(c, "MeshHitDetection")
		chargeOff := len(chargeFields) == 0 || len(chargeFields) == 1 && w.value("IsChargeWeapon") == "false"
		meshOff := len(meshFields) == 0 || len(meshFields) == 1 && c.value("MeshHitDetection") == "false"
		basic := w.value("Type") == "Hitscan" && shots.Value != nil && *shots.Value == 1 && w.value("IsBurstWeapon") == "false" && chargeOff && len(source(w, "AlsoShoot")) == 1 && w.value("AlsoShoot") == ""
		allowed := root.value("InvincibleBots") == "false" && c.value("InvincibleBots") == "false"
		var hits, seconds *float64
		bodySafe := c.value("HeadshotOnly") == "false" && meshOff
		if basic && allowed && bodySafe && positive(damage) && positive(health) && positive(dt) && regen.Value != nil && *regen.Value == 0 {
			hits = num(math.Ceil(*health.Value / *damage.Value))
			seconds = num((*hits - 1) * *dt.Value)
		}
		sources := append(append(append(damage.Sources, health.Sources...), dt.Sources...), regen.Sources...)
		sources = append(sources, chargeFields...)
		sources = append(sources, meshFields...)
		out = append(out, derived("ideal_body_hits_to_kill", "hits", hits, sources, "One projectile per click, hitscan, no burst/charge/AlsoShoot; explicit non-invincibility, no regeneration", "All body hits apply full declared damage; no mitigation, buffs or external interactions"), derived("ideal_post_first_hit_kill_seconds", "configured seconds", seconds, sources, "Successful shots at declared minimum interval; excludes first acquisition, reload and reaction time"))
		m := numberFact(w, "MagazineMax", "ammo")
		cost := numberFact(w, "AmmoPerShot", "ammo")
		refund := numberFact(w, "AmmoReloadedOnKill", "ammo")
		if hits != nil && *hits == 1 && positive(m) && positive(cost) && nonnegative(refund) && *refund.Value >= *m.Value {
			for _, q := range []float64{.5, .75, .9} {
				rate, valid := FullRefundReloadRate(*m.Value, *cost.Value, q)
				if valid {
					out = append(out, derived(reloadKey(q), "reloads per 1000 shots", num(rate*1000), append(append(m.Sources, cost.Sources...), refund.Sources...), "One hit kills and fully refills magazine; independent fixed hit probability; automatic full reload before insufficient-ammo shot; no reload overlap"))
				}
			}
		}
		// Legacy formats omit these flags. A hypothetical window may still be
		// useful, but the absent flags remain explicitly unverified assumptions.
		if len(chargeFields) == 0 || len(meshFields) == 0 {
			for i := start; i < len(out); i++ {
				if out[i].Value != nil {
					out[i].Conditions = append(out[i].Conditions, "Absent legacy charge/mesh-hit flags are assumed inactive for this calculation only; engine defaults are not verified")
					out[i].Unknown = append(out[i].Unknown, "unverified_legacy_weapon_or_mesh_gate")
				}
			}
		}
	}
	return out
}
func reloadKey(q float64) string {
	switch q {
	case .5:
		return "full_refund_reload_at_q50"
	case .75:
		return "full_refund_reload_at_q75"
	}
	return "full_refund_reload_at_q90"
}

// FullRefundReloadRate models consecutive misses, not a measured player cost.
func FullRefundReloadRate(magazine, cost, q float64) (float64, bool) {
	if magazine < cost || cost <= 0 || q < 0 || q > 1 || math.IsNaN(magazine+cost+q) || math.IsInf(magazine, 0) || math.IsInf(cost, 0) {
		return 0, false
	}
	n := math.Floor(magazine / cost)
	if n < 1 {
		return 0, false
	}
	if q == 0 {
		return 1 / n, true
	}
	if q == 1 {
		return 0, true
	}
	miss := math.Pow(1-q, n)
	return q * miss / (1 - miss), true
}

type MotionEnvelope struct {
	PeakSpeed       float64  `json:"peakSpeed"`
	RMSSpeed        float64  `json:"rmsSpeed"`
	CenterExcursion float64  `json:"centerExcursion"`
	CapReached      bool     `json:"capReached"`
	Status          string   `json:"status"`
	Conditions      []string `json:"conditions"`
}

// PeriodicMotion is a 1D idealization. Shorter dwell can reduce both speed and
// excursion; 1/dwell is never used as a monotonic difficulty coefficient.
func PeriodicMotion(speed, acceleration, dwell float64) (*MotionEnvelope, bool) {
	if speed < 0 || acceleration < 0 || dwell <= 0 || math.IsNaN(speed+acceleration+dwell) || math.IsInf(speed+acceleration+dwell, 0) {
		return nil, false
	}
	e := &MotionEnvelope{Status: "calculated_under_explicit_conditions", Conditions: []string{"1D periodic command +/-speed, dwell per direction, acceleration limited, no other forces; not an engine trajectory"}}
	if speed == 0 || acceleration == 0 {
		return e, true
	}
	e.CapReached = acceleration*dwell >= 2*speed
	if e.CapReached {
		e.PeakSpeed = speed
		e.RMSSpeed = speed * math.Sqrt(1-4*speed/(3*acceleration*dwell))
		e.CenterExcursion = speed*dwell - speed*speed/acceleration
	} else {
		e.PeakSpeed = acceleration * dwell / 2
		e.RMSSpeed = acceleration * dwell / (2 * math.Sqrt(3))
		e.CenterExcursion = acceleration * dwell * dwell / 4
	}
	if math.IsInf(e.PeakSpeed, 0) || math.IsInf(e.RMSSpeed, 0) || math.IsInf(e.CenterExcursion, 0) {
		return nil, false
	}
	return e, true
}
