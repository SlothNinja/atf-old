package atf

import (
	"encoding/gob"
	"html/template"
	"strconv"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"golang.org/x/net/context"
)

func init() {
	gob.Register(new(payActionCostEntry))
}

func (g *Game) payActionCost(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	var r Resource

	if r, err = g.validatePayActionCost(ctx); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	switch r {
	case Army:
		cp.Army -= 1
		cp.ArmySupply += 1
	case Worker:
		cp.Worker -= 1
		cp.WorkerSupply += 1
	default:
		cp.Resources[r] -= 1
		g.Resources[r] += 1
	}

	cp.PaidActionCost = true

	// Log Placement
	e := cp.newPayActionCostEntry(r)
	restful.AddNoticef(ctx, string(e.HTML()))

	tmpl, act = "atf/paid_action_cost_update", game.Cache
	return
}

func (g *Game) validatePayActionCost(ctx context.Context) (r Resource, err error) {
	if err = g.validatePlayerAction(ctx); err != nil {
		return
	}

	var rInt int

	c := restful.GinFrom(ctx)
	if rInt, err = strconv.Atoi(c.PostForm("Resource")); err != nil {
		return
	}

	r = Resource(rInt)
	cp := g.CurrentPlayer()

	switch {
	case cp.PaidActionCost:
		err = sn.NewVError("You have already paid action cost.")
	case r == Army:
		if cp.Army < 1 {
			err = sn.NewVError("You do not have an army to pay action cost.")
		}
	case r == Worker:
		if cp.Worker < 1 {
			err = sn.NewVError("You do not have an worker to pay action cost.")
		}
	default:
		if cp.Resources[r] < 0 {
			err = sn.NewVError("You do not have a %v to pay action cost.", r)
		}
	}
	return
}

type payActionCostEntry struct {
	*Entry
	Resource Resource
}

func (p *Player) newPayActionCostEntry(r Resource) *payActionCostEntry {
	g := p.Game()
	e := &payActionCostEntry{
		Entry:    p.newEntry(),
		Resource: r,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *payActionCostEntry) HTML() template.HTML {
	return restful.HTML("%s paid %s to perform an action.", e.Player().Name(), e.Resource)
}
