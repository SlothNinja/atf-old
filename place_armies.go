package atf

import (
	"encoding/gob"
	"html/template"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"golang.org/x/net/context"
)

func init() {
	gob.Register(new(placeArmiesEntry))
	gob.Register(new(removeWorkersEntry))
}

func (g *Game) placeArmies(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	var armies int
	if armies, err = g.validatePlaceArmies(ctx); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.PerformedAction = true
	cp.Army -= armies
	area := g.SelectedArea()
	area.Armies = armies
	area.ArmyOwnerID = cp.ID()
	g.MultiAction = placedArmiesMA

	// Remove workers
	if w := cp.WorkersIn(area); w > 0 {
		cp.WorkerSupply += cp.WorkersIn(area)
		cp.setWorkersIn(area, 0)

		// Log removal
		e1 := cp.newRemoveWorkersEntry(w, area)
		restful.AddNoticef(ctx, string(e1.HTML()))
	}

	// Log Placed Armies
	e2 := cp.newPlaceArmiesEntry(armies, area)
	restful.AddNoticef(ctx, string(e2.HTML()))

	tmpl, act = "atf/place_armies_update", game.Cache
	return
}

func (g *Game) validatePlaceArmies(ctx context.Context) (armies int, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	var cp *Player
	switch cp, armies, err = g.CurrentPlayer(), getPlacedArmies(ctx), g.validatePlayerAction(ctx); {
	case err != nil:
	case cp.PerformedAction:
		err = sn.NewVError("You have already performed an action.")
	case armies < 1 || armies > 2:
		err = sn.NewVError("You can't place %d armies in %s.", armies, g.SelectedArea().Name())
	}
	return
}

type placeArmiesEntry struct {
	*Entry
	Armies   int
	AreaName string
}

func (p *Player) newPlaceArmiesEntry(armies int, area *Area) *placeArmiesEntry {
	g := p.Game()
	e := &placeArmiesEntry{
		Entry:    p.newEntry(),
		Armies:   armies,
		AreaName: area.Name(),
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *placeArmiesEntry) HTML() template.HTML {
	if e.Armies == 1 {
		return restful.HTML("%s placed 1 army in %s.", e.Player().Name(), e.AreaName)
	}
	return restful.HTML("%s placed 2 armies in %s.", e.Player().Name(), e.AreaName)
}

type removeWorkersEntry struct {
	*Entry
	Workers  int
	AreaName string
}

func (p *Player) newRemoveWorkersEntry(workers int, area *Area) *removeWorkersEntry {
	g := p.Game()
	e := &removeWorkersEntry{
		Entry:    p.newEntry(),
		Workers:  workers,
		AreaName: area.Name(),
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *removeWorkersEntry) HTML() template.HTML {
	return restful.HTML("%s removed %d %s from %s.",
		e.Player().Name(), e.Workers, restful.Pluralize("worker", e.Workers), e.AreaName)
}
