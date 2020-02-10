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
	gob.Register(new(useScribeEntry))
}

func (g *Game) useScribe(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err = g.validateUseScribe(ctx); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.incWorkersIn(g.Areas[Scribes], -1)
	cp.incWorkersIn(g.Areas[UsedScribes], 1)
	g.MultiAction = usedScribeMA
	cp.PerformedAction = false

	restful.AddNoticef(ctx, "Select worker to move.")
	tmpl, act = "atf/use_scribe_update", game.Cache
	return
}

func (g *Game) validateUseScribe(ctx context.Context) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	var (
		a  *Area
		cp *Player
	)

	switch cp, a, err = g.CurrentPlayer(), g.SelectedArea(), g.validatePlayerAction(ctx); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("No area selected.")
	case a.ID != Scribes:
		err = sn.NewVError("You must chose Scribes area in order to use scribe.")
	case cp.PerformedAction && g.MultiAction != placedWorkerMA && !g.PlacedWorkers:
		err = sn.NewVError("You have already performed an action.")
	case cp.WorkersIn(a) < 1:
		err = sn.NewVError("You don't have a scribe to use.")
	}
	return
}

type useScribeEntry struct {
	*Entry
	From string
	To   string
}

func (p *Player) newUseScribeEntry() *useScribeEntry {
	g := p.Game()
	e := &useScribeEntry{
		Entry: p.newEntry(),
		From:  g.From,
		To:    g.To,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *useScribeEntry) HTML() template.HTML {
	return restful.HTML("%s used scribe to move worker from %s to %s.", e.Player().Name(), e.From, e.To)
}
