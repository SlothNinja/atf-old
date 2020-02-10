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
	gob.Register(new(placeWorkersEntry))
}

func (g *Game) placeWorkers(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	res, workers, err := g.validatePlaceWorkers(ctx)
	if err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.PerformedAction = true
	cp.Resources[res] -= 1
	g.Resources[res] += 1
	cp.Worker -= workers
	area := g.SelectedArea()
	if area.ID == Scribes {
		area = g.Areas[NewScribes]
	}
	cp.incWorkersIn(area, workers)
	g.PlacedWorkers = true

	// Log
	e := cp.newPlaceWorkersEntry(res, workers)
	restful.AddNoticef(ctx, string(e.HTML()))
	tmpl, act = "atf/place_workers_update", game.Cache
	return
}

func (g *Game) validatePlaceWorkers(ctx context.Context) (rs Resource, ws int, err error) {
	if err = g.validatePlayerAction(ctx); err != nil {
		return
	}

	rs = getPaidResource(ctx)
	ws = getPlaceWorkers(ctx)
	cp := g.CurrentPlayer()
	a := g.SelectedArea()

	switch err = cp.canPlaceWorkersIn(a); {
	case err != nil:
	case rs == noResource:
		err = sn.NewVError("You must spend a resource to place workers in %s.", a.Name())
	case cp.Resources[rs] < 1:
		err = sn.NewVError("You do not have a %s to spend.", rs)
	case ws < 1:
		err = sn.NewVError("You must place at least 1 worker.")
	case ws > resourceValueMap[rs]:
		err = sn.NewVError("You tried to place %d workers, but a %s permits only up to %d workers", ws, rs, resourceValueMap[rs])
	case ws > cp.Worker:
		err = sn.NewVError("You tried to place %d workers, but have only %d workers available.", ws, cp.Worker)
	case (a.ID == Scribes) && (cp.totalScribes()+ws > 2):
		err = sn.NewVError("You tried to place %d workers in Scribes box, which would give you %d scribes.  You can have no more than two scribes.", ws, cp.totalScribes()+ws)
	}
	return
}

type placeWorkersEntry struct {
	*Entry
	AreaName string
	Resource Resource
	Workers  int
}

func (p *Player) newPlaceWorkersEntry(res Resource, workers int) *placeWorkersEntry {
	g := p.Game()
	e := &placeWorkersEntry{
		Entry:    p.newEntry(),
		AreaName: g.SelectedArea().Name(),
		Resource: res,
		Workers:  workers,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *placeWorkersEntry) HTML() template.HTML {
	return restful.HTML("%s spent %s to place %d workers in %s.", e.Player().Name(), e.Resource, e.Workers, e.AreaName)
}
