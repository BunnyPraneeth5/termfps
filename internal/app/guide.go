package app

import (
	"fmt"
	"math"
	"strings"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// The in-game guide (M18a, D51), opened from the title and pause menus.
// Every number on its pages comes from the game code, so balancing can't
// make the guide wrong.

// guideEntry is one guide page.
type guideEntry struct {
	name   string
	lines  []string  // paragraphs; "" is an empty line
	view   guideView // shown in 3D (M18b); viewNone = none
	kind   pageKind  // how the page is laid out (M18c)
	mapIdx int       // pageMap: index into App.maps
}

// guideSection is a group of pages, one item in the guide menu.
type guideSection struct {
	name    string
	entries []guideEntry
}

// guideSections builds every page from the current game numbers, maps and
// progress (locked maps stay a surprise).
func (a *App) guideSections() []guideSection {
	return []guideSection{
		{"Controls and HUD", []guideEntry{controlsPage(), hudPage()}},
		{"Enemies", a.enemyPages()},
		{"Weapons", a.gunPages()},
		{"Pickups", pickupPages()},
		{"Hazards", hazardPages()},
		{"Survival and waves", survivalPages()},
		{"Maps", a.mapPages()},
		{"Settings", []guideEntry{settingsPage()}},
		{"Tips", []guideEntry{tipsPage()}},
	}
}

// num formats a number without trailing zeros: 1.5, 3, 0.25.
func num(v float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}

func controlsPage() guideEntry {
	return guideEntry{name: "Controls", kind: pageKeys, lines: []string{
		"In the pause menu: Resume, Settings, Restart, Guide, Quit to menu.",
		"There is no mouse-look: terminals can't capture the mouse. On Windows you can move and turn at the same time (the async key mode).",
	}}
}

func hudPage() guideEntry {
	return guideEntry{name: "HUD", kind: pageHUD, lines: []string{
		"The parts on the left always show; the ones on the right only when the window is wide enough. STAMINA shows while it isn't full; in waves mode WAVE 3/10 takes the tier's place.",
		"Notices show just above it: a new tier, a wave, a new map, the hunt.",
		"The red edge means you were hit; the red arc around the crosshair points to where the hit came from. Ticks around the crosshair mean your shot hit (red: a kill).",
		"The minimap (top right): walls, doors, props, green acid, and dots for pickups: yellow ammo, red medkit, blue armor, white gun.",
	}}
}

// shotsToKill says how many hits of each gun kill an enemy with hp at tier
// 1 (the shotgun with every pellet, up close).
func shotsToKill(hp int) string {
	var parts []string
	for k := range game.NumGuns {
		st := k.Stats()
		n := int(math.Ceil(float64(hp) / float64(st.Damage*st.Pellets)))
		parts = append(parts, fmt.Sprintf("%s %d", strings.ToLower(st.Name), n))
	}
	return "Shots to kill at tier 1: " + strings.Join(parts, ", ") + " (shotgun up close)."
}

// arrives says when enemy kind k first comes.
func arrives(k game.EnemyKind) string {
	t := game.FirstTier(k)
	if t <= 1 {
		return "Comes from the start."
	}
	return fmt.Sprintf("Comes from tier %d (in waves mode, from wave %d).", t, 2*t-1)
}

func (a *App) enemyPages() []guideEntry {
	sprite := func(k game.EnemyKind) guideView { return spriteView(render.EnemySprite(game.Enemy{Kind: k})) }
	walk := fmt.Sprintf("(you walk %s)", num(game.MoveSpeed))
	pages := []guideEntry{
		{name: "Zombie", view: sprite(game.KindZombie), lines: []string{
			"Green. Walks at you and hits up close.",
			fmt.Sprintf("HP %d. Speed %s cells per second %s.", game.ZombieHP, num(game.ZombieSpeed), walk),
			fmt.Sprintf("Hits for %d-%d every %s s.", game.ZombieMeleeDamageMin, game.ZombieMeleeDamageMax, num(game.ZombieMeleeInterval)),
			shotsToKill(game.ZombieHP), arrives(game.KindZombie),
			fmt.Sprintf("From tier %d zombies split up and come from your sides and from behind.", game.FlankTier),
			"Tip: back away while you shoot; it can't catch you walking backwards.",
		}},
		{name: "Shooter", view: sprite(game.KindShooter), lines: []string{
			"Red. Keeps its distance and shoots when it can see you.",
			fmt.Sprintf("HP %d. Speed %s cells per second. Shoots from %s cells or closer, every %s s, %d damage; backs off if you get closer than %s cells.",
				game.ShooterHP, num(game.ShooterSpeed), num(game.ShooterRange), num(game.ShooterInterval), game.ShooterDamage, num(game.ShooterMinRange)),
			fmt.Sprintf("Hits %d%% of the time up close, down to %d%% from far away.",
				int(math.Round(game.ShooterHitChance(0)*100)), int(math.Round(game.ShooterAccMin*100))),
			shotsToKill(game.ShooterHP), arrives(game.KindShooter),
			fmt.Sprintf("From tier %d it hides after each shot, then comes out again.", game.CoverTier),
			"Tip: break its line of sight; it can't hit what it can't see.",
		}},
		{name: "Runner", view: sprite(game.KindRunner), lines: []string{
			"Blue. Fast and weak.",
			fmt.Sprintf("HP %d. Speed %s cells per second %s.", game.RunnerHP, num(game.RunnerSpeed), walk),
			fmt.Sprintf("Hits for %d-%d every %s s.", game.RunnerMeleeDamageMin, game.RunnerMeleeDamageMax, num(game.RunnerMeleeInterval)),
			shotsToKill(game.RunnerHP), arrives(game.KindRunner),
			fmt.Sprintf("Tip: you can't outwalk it; sprint (F, %sx) or stand and shoot.", num(game.SprintMul)),
		}},
		{name: "Tank", view: sprite(game.KindTank), lines: []string{
			"Big and gray. Slow, very tough, hits hard and shakes the screen.",
			fmt.Sprintf("HP %d. Speed %s cells per second %s.", game.TankHP, num(game.TankSpeed), walk),
			fmt.Sprintf("Hits for %d-%d every %s s.", game.TankMeleeDamageMin, game.TankMeleeDamageMax, num(game.TankMeleeInterval)),
			shotsToKill(game.TankHP), arrives(game.KindTank),
			"Tip: keep your distance and use the rifle; lure it next to a red barrel.",
		}},
		{name: "Exploder", view: sprite(game.KindExploder), lines: []string{
			"Yellow. Runs at you and blows up.",
			fmt.Sprintf("HP %d. Speed %s cells per second. Within %s cells it lights a %s s fuse (it blinks and beeps), then explodes for up to %d damage.",
				game.ExploderHP, num(game.ExploderSpeed), num(game.ExploderFuseRange), num(game.ExploderFuse), game.ExploderDamage),
			"Shot dead, it blows up where it stands and hurts the enemies around it (they count as your kills). If it blows itself up, it isn't a kill.",
			shotsToKill(game.ExploderHP), arrives(game.KindExploder),
			"Tip: shoot it early, when it's next to other enemies and far from you.",
		}},
	}
	for k := range pages {
		kind := game.EnemyKind(k)
		if a.met(kind) {
			continue
		}
		// Not met yet (M18c): a dark silhouette until the player meets one.
		shadow := render.EnemySprite(game.Enemy{Kind: kind})
		shadow.Shadow = true
		pages[k] = guideEntry{name: "???", view: spriteView(shadow), lines: []string{
			"You haven't met this enemy yet.", arrives(kind),
		}}
	}
	return pages
}

func (a *App) gunPages() []guideEntry {
	pages := make([]guideEntry, 0, game.NumGuns)
	use := [game.NumGuns]string{
		"Good all-round: quick shots, lots of ammo.",
		"Deadly up close, weak far away: the pellets spread.",
		"For far and tough targets: one shot kills most enemies, but it's slow.",
	}
	for k := range game.NumGuns {
		st := k.Stats()
		found := "You start with it."
		if st.Arrives > 0 {
			found = fmt.Sprintf("Shows up on the map after %s; walk over it to take it. It comes with %d spare rounds.",
				render.Clock(st.Arrives), st.StartReserve)
		}
		dmg := fmt.Sprintf("%d damage", st.Damage)
		if st.Pellets > 1 {
			dmg = fmt.Sprintf("%d pellets of %d damage (%d if all hit)", st.Pellets, st.Damage, st.Pellets*st.Damage)
		}
		pages = append(pages, guideEntry{name: st.Name[:1] + strings.ToLower(st.Name[1:]), view: guideView{kind: viewGun, gun: k}, lines: []string{
			fmt.Sprintf("Key %d. %s", int(k)+1, use[k]),
			fmt.Sprintf("%s a shot, %s shots per second, %d rounds in the gun, %s s to reload.",
				dmg, num(1/st.Interval), st.Mag, num(st.Reload)),
			fmt.Sprintf("Spare rounds: up to %d; an ammo box gives %d.", st.MaxReserve, st.BoxAmmo),
			found,
		}})
		if !a.found(k) { // not found yet (M18c): a dark silhouette
			shadow := render.PickupSprite(game.Pickup{Kind: game.GunPickup, Gun: k})
			shadow.Shadow = true
			pages[k] = guideEntry{name: "???", view: spriteView(shadow), lines: []string{
				"You haven't found this gun yet.",
				fmt.Sprintf("It shows up on the map after %s.", render.Clock(st.Arrives)),
			}}
		}
	}
	return pages
}

func pickupPages() []guideEntry {
	spawn := fmt.Sprintf("Pickups appear every %s-%s s, at most %d at once, never right next to you. Walk over one to take it; one you can't use stays.",
		num(game.PickupEveryMin), num(game.PickupEveryMax), game.MaxPickups)
	var ammo []string
	for k := range game.NumGuns {
		st := k.Stats()
		ammo = append(ammo, fmt.Sprintf("%s +%d (up to %d)", strings.ToLower(st.Name), st.BoxAmmo, st.MaxReserve))
	}
	return []guideEntry{
		{name: "Ammo", view: spriteView(render.PickupSprite(game.Pickup{Kind: game.AmmoBox})), lines: []string{
			"Each ammo box is for one of the guns you have: " + strings.Join(ammo, ", ") + ".", spawn,
		}},
		{name: "Medkit", view: spriteView(render.PickupSprite(game.Pickup{Kind: game.Medkit})), lines: []string{
			fmt.Sprintf("White with a red cross: +%d HP, up to %d.", game.MedkitHeal, game.PlayerMaxHP),
			fmt.Sprintf("You also heal %s HP per second after %s s without being hit, but only up to %d: above that only medkits heal.",
				num(game.RegenPerSec), num(game.RegenDelay), game.RegenMaxHP),
			spawn,
		}},
		{name: "Armor", view: spriteView(render.PickupSprite(game.Pickup{Kind: game.ArmorVest})), lines: []string{
			fmt.Sprintf("Blue vest: +%d armor, up to %d.", game.ArmorPickup, game.MaxArmor),
			"Armor takes half of every hit (rounded up) until it runs out.", spawn,
		}},
		{name: "Guns on the map", view: spriteView(render.PickupSprite(game.Pickup{Kind: game.GunPickup, Gun: game.Shotgun})), lines: []string{
			fmt.Sprintf("The shotgun appears after %s and the rifle after %s, one at a time, as a white dot on the minimap. Taking one puts it in your hands.",
				render.Clock(game.Shotgun.Stats().Arrives), render.Clock(game.Rifle.Stats().Arrives)),
		}},
	}
}

func hazardPages() []guideEntry {
	return []guideEntry{
		{name: "Red barrels", view: spriteView(render.PropSprite(world.ExplosiveBarrel, 0, 0)), lines: []string{
			fmt.Sprintf("Shoot one and it explodes for up to %d damage. Other red barrels in reach go off a moment later.", game.BarrelDamage),
			"Gray barrels, crates and pillars are plain cover: they stop bullets.",
			"Tip: wait until enemies crowd around a red barrel, and stay back yourself.",
		}},
		{name: "Acid", view: guideView{kind: viewAcid}, lines: []string{
			fmt.Sprintf("Green floor. It burns %d HP every %s s while you stand in it; enemies don't mind it.", game.AcidDamage, num(game.AcidInterval)),
			"Nothing spawns on it. It shows green on the minimap too.",
		}},
		{name: "Explosions", view: boomView(), lines: []string{
			fmt.Sprintf("An explosion does full damage within %s cell, less farther out, nothing past %s cells.", num(game.ExplosionCore), num(game.ExplosionRadius)),
			"It hurts you and every enemy in reach. Walls and closed doors shield; crates don't.",
			"Explosions shake the screen.",
		}},
	}
}

func survivalPages() []guideEntry {
	tiers := []string{
		fmt.Sprintf("Survival is endless. Every %d kills or %s, whichever comes first, the tier goes up.", game.TierKills, render.Clock(game.TierTime)),
	}
	for t := 1; t <= 6; t++ {
		st := game.TierFor(t)
		tiers = append(tiers, fmt.Sprintf("Tier %d: enemy HP x%s, damage x%s, speed x%s; a new enemy every %s s, up to %d alive.",
			t, num(st.HPMul), num(st.DamageMul), num(st.SpeedMul), num(st.SpawnEvery), st.MaxAlive))
	}
	tiers = append(tiers, "And so on, up to the limits.")
	return []guideEntry{
		{name: "Tiers", lines: tiers},
		{name: "Enemies learn", lines: []string{
			"They start simple: every enemy walks the shortest way to you.",
			fmt.Sprintf("Tier %d: they hear your gunshots within %d steps and rush to you.", game.HearTier, game.HearingSteps),
			fmt.Sprintf("Tier %d: shooters hide after they shoot.", game.CoverTier),
			fmt.Sprintf("Tier %d: zombies come from your sides and from behind.", game.FlankTier),
			fmt.Sprintf("Don't camp: after %s without a kill every enemy comes straight for you, faster, until your next kill. A horn warns you.", render.Clock(game.CampTime)),
		}},
		{name: "Waves mode", lines: []string{
			fmt.Sprintf("Choose Waves after Play: %d waves, a %s s break between them; clear the last one to win.", game.NumWaves, num(game.WaveBreak)),
			fmt.Sprintf("Wave 1 has %d enemies, wave %d has %d. The tier goes up every two waves, so every enemy kind shows up.",
				game.WaveSize(1), game.NumWaves, game.WaveSize(game.NumWaves)),
			"Use the breaks to reload and pick things up. Your best wave and fastest win are kept per map.",
		}},
		{name: "Health and stamina", lines: []string{
			fmt.Sprintf("You have %d HP. After %s s without being hit you heal %s HP per second, up to %d; medkits heal above that.",
				game.PlayerMaxHP, num(game.RegenDelay), num(game.RegenPerSec), game.RegenMaxHP),
			fmt.Sprintf("Sprinting (F) is %sx speed. Full stamina lasts %s s and refills in %s s; run out and you wait until it's back to %d%%.",
				num(game.SprintMul), num(1/game.StaminaDrain), num(1/game.StaminaRegen), int(game.StaminaResume*100)),
		}},
	}
}

// mapTips are short tips for the built-in maps, by name.
var mapTips = map[string]string{
	"The Arena":     "Small and open: keep moving and use the pillars to break line of sight.",
	"The Warehouse": "Shelves make long lanes: good for the rifle. Red barrels sit near the crates.",
	"The Yard":      "Wide open ground: shooters see you from far, so move between containers.",
	"The Tunnel":    "Long and narrow: enemies come in a line. Mind the acid puddles.",
	"The Office":    "Many rooms and doors: listen for where enemies come from.",
	"The Subway":    "Two platforms and the tracks: acid on the track bed, red barrels on the platforms.",
}

// mapPages describes every map; a locked one only says it's locked.
func (a *App) mapPages() []guideEntry {
	var pages []guideEntry
	for i, m := range a.maps {
		name := m.Name
		if name == "" {
			name = fmt.Sprintf("Map %d", i+1)
		}
		if a.mapLocked(i) {
			pages = append(pages, guideEntry{name: name + " (locked)", kind: pageMap, mapIdx: i, lines: []string{
				fmt.Sprintf("Locked. Survive %s on %s to open it.", render.Clock(unlockAfter(a.cfg.UnlockRule)), a.maps[i-1].Name),
			}})
			continue
		}
		barrels, acid := 0, 0
		for _, c := range m.Cells {
			switch c {
			case world.ExplosiveBarrel:
				barrels++
			case world.Acid:
				acid++
			}
		}
		lines := []string{
			fmt.Sprintf("%d x %d cells. %d doors, %d lamps, %d starting enemies.", m.W, m.H, len(m.Doors), len(m.Lamps), len(m.EnemyStarts)),
			fmt.Sprintf("Red barrels: %d. Acid cells: %d.", barrels, acid),
		}
		view := guideView{kind: viewMap, mapIdx: i}
		if seen, played := a.explored(i); played {
			lines = append([]string{fmt.Sprintf("Explored: %d%%.", int(math.Round(100*exploredShare(m, seen))))}, lines...)
		} else {
			view = guideView{} // the 360° turn only after you've played it (M18c)
			lines = append([]string{"Not played yet: everything's dark until you explore it."}, lines...)
		}
		if tip, ok := mapTips[m.Name]; ok {
			lines = append(lines, "Tip: "+tip)
		}
		if i >= a.builtinMaps() {
			lines = append(lines, "Your own map, from the maps folder or -map.")
		}
		pages = append(pages, guideEntry{name: name, lines: lines, view: view, kind: pageMap, mapIdx: i})
	}
	return pages
}

func settingsPage() guideEntry {
	return guideEntry{name: "Settings", lines: []string{
		"Every setting takes effect at once and is saved.",
		"Render mode: half-block (default), shade, ASCII or mixed. Mixed background: how bright the fill behind mixed-mode characters is.",
		"Color threshold: reuse a cell's color when the next one is close, so slow terminals get less to draw (off by default).",
		"Render scale: auto draws the view at most 160 columns wide in big windows; full; half.",
		"FPS: 30 or 60. Minimap at start. Wall textures. Sound and volume.",
		"Key mode: async (Windows: real key state, move and turn at once) or events.",
		"Pixel aspect: tune if walls look too tall or too flat.",
		"Map unlocks: survive 5:00, survive 2:00, or all maps open. Minimap fog of war: show only what you've seen.",
	}}
}

func tipsPage() guideEntry {
	return guideEntry{name: "Tips", lines: []string{
		"Keep moving: standing still lets enemies surround you, and camping starts the hunt.",
		"Walk backwards while you shoot zombies and tanks: they can't catch you.",
		"Reload in quiet moments; switching guns cancels a reload.",
		"Save the rifle for tanks and shooters far away, the shotgun for close calls.",
		"Grab armor when you can: it halves the damage you take.",
		"Use corners and cover against shooters; listen for where sounds come from.",
		"Lure groups to red barrels, and shoot exploders before they reach you.",
	}}
}

// guideMenu is the guide's first page: its sections. back is the menu to
// return to (title or pause).
func (a *App) guideMenu(back *menuScreen) *menuScreen {
	sections := a.guideSections()
	items := make([]ui.Item, 0, len(sections)+1)
	for _, s := range sections {
		items = append(items, ui.Item{Label: s.name})
	}
	items = append(items, ui.Item{Label: "Back"})
	s := &menuScreen{overGame: back.overGame, menu: ui.New("GUIDE", nil, items)}
	s.act = func(a *App, r ui.Result) {
		i := s.menu.Selected
		switch {
		case r == ui.Back || i == len(sections):
			a.switchTo(back)
		case r == ui.Chosen && len(sections[i].entries) == 1:
			a.switchTo(&guidePage{back: s, section: sections[i], shown: -1})
		case r == ui.Chosen:
			a.switchTo(a.guideSectionMenu(s, sections[i]))
		}
	}
	return s
}

// guideSectionMenu lists a section's pages.
func (a *App) guideSectionMenu(back *menuScreen, sec guideSection) *menuScreen {
	items := make([]ui.Item, 0, len(sec.entries)+1)
	for _, e := range sec.entries {
		items = append(items, ui.Item{Label: e.name})
	}
	items = append(items, ui.Item{Label: "Back"})
	s := &menuScreen{overGame: back.overGame, menu: ui.New("GUIDE: "+strings.ToUpper(sec.name), nil, items)}
	s.act = func(a *App, r ui.Result) {
		i := s.menu.Selected
		switch {
		case r == ui.Back || i == len(sec.entries):
			a.switchTo(back)
		case r == ui.Chosen:
			a.switchTo(&guidePage{back: s, section: sec, index: i, shown: -1})
		}
	}
	return s
}
