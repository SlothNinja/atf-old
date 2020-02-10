package atf

import (
	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"golang.org/x/net/context"
)

func (g *Game) fromStock(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err = g.validateFromStock(ctx); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	g.CurrentPlayer().Worker -= 1
	g.From = "Stock"
	g.MultiAction = selectedWorkerMA
	tmpl, act = "atf/select_worker_from_stock_update", game.Cache
	return
}

func (g *Game) validateFromStock(ctx context.Context) (err error) {
	switch err = g.validatePlayerAction(ctx); {
	case err != nil:
	case g.MultiAction != usedScribeMA:
		err = sn.NewVError("You cannot chose 'From Stock' at this time.")
	case g.CurrentPlayer().Worker < 1:
		err = sn.NewVError("You have no available workers to place.")
	}
	return
}

func (g *Game) selectWorker(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err = g.validateSelectWorker(ctx); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	a := g.SelectedArea()

	switch {
	case a.ID == UsedScribes:
		cp.incWorkersIn(a, -1)
		g.From = "Scribes"
	default:
		cp.incWorkersIn(a, -1)
		g.From = a.Name()
	}

	g.MultiAction = selectedWorkerMA

	// Log
	restful.AddNoticef(ctx, "Select area to place worker.")
	tmpl, act = "atf/select_worker_update", game.Cache
	return
}

func (g *Game) validateSelectWorker(ctx context.Context) (err error) {
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
	case g.SelectedAreaID == WorkerStock:
		if cp.Worker < 1 {
			err = sn.NewVError("You have no available workers to place.")
		}
	case a.IsSumer():
		err = sn.NewVError("You have no workers in %s.", a.Name())
	case cp.WorkersIn(a) < 1:
		err = sn.NewVError("You have no workers in %s.", a.Name())
	}
	return
}
