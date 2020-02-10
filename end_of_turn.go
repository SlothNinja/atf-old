package atf

import (
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"golang.org/x/net/context"
)

func (g *Game) endOfTurn(ctx context.Context) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	g.Phase = EndOfTurn
	g.returnArmies(ctx)
	g.returnWorkers(ctx)
	g.resetPassboxes(ctx)
	g.resetArmyBoxes(ctx)
}

func (g *Game) returnArmies(ctx context.Context) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	for _, a := range g.Areas {
		a.Armies = 0
		a.ArmyOwnerID = NoPlayerID
	}
	for _, p := range g.Players() {
		p.ArmySupply = 20
		p.Army = 0
	}
}

func (g *Game) returnWorkers(ctx context.Context) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	for _, p := range g.Players() {
		p.WorkerSupply += p.Worker
		p.Worker = 0
	}
}

func (g *Game) resetPassboxes(ctx context.Context) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	for _, p := range g.Players() {
		for r, count := range p.PassedResources {
			p.PassedResources[r] = 0
			g.Resources[r] += count
		}
	}
}

func (g *Game) resetArmyBoxes(ctx context.Context) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	for _, p := range g.Players() {
		if emp := p.empire(); emp != nil {
			for r, count := range emp.Equipment {
				emp.Equipment[r] = 0
				g.Resources[r] += count
			}
		}
	}
}
