package atf

import (
	"net/http"
	"time"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user/stats"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
)

func Finish(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))

		g := gameFrom(ctx)
		switch g.Phase {
		case Actions:
			if err := g.actionsPhaseFinishTurn(ctx); err != nil {
				log.Errorf(ctx, "g.actionsPhaseFinishTurn error: %v", err)
				return
			}
		case ExpandCity:
			if err := g.expandCityPhaseFinishTurn(ctx); err != nil {
				log.Errorf(ctx, "g.expandCityPhaseFinishTurn error: %v", err)
				return
			}

		}
	}
}

func (g *Game) validateFinishTurn(ctx context.Context) (s *stats.Stats, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	var cp *Player
	switch cp, s = g.CurrentPlayer(), stats.Fetched(ctx); {
	case s == nil:
		err = sn.NewVError("missing stats for player.")
	case !g.CUserIsCPlayerOrAdmin(ctx):
		err = sn.NewVError("Only the current player may finish a turn.")
	case !cp.PerformedAction:
		err = sn.NewVError("%s has yet to perform an action.", g.NameFor(cp))
	}
	return
}

// ps is an optional parameter.
// If no player is provided, assume current player.
func (g *Game) nextPlayer(ps ...game.Playerer) *Player {
	if nper := g.NextPlayerer(ps...); nper != nil {
		return nper.(*Player)
	}
	return nil
}

func (g *Game) actionPhaseNextPlayer(pers ...game.Playerer) *Player {
	cp := g.CurrentPlayer()
	cp.endOfTurnUpdate()
	ps := g.Players()
	p := g.nextPlayer(pers...)
	for !ps.allPassed() {
		if p.Passed {
			p = g.nextPlayer(p)
		} else {
			p.beginningOfTurnReset()
			if p.canAutoPass() {
				p.autoPass()
				p = g.nextPlayer(p)
			} else {
				return p
			}
		}
	}
	return nil
}

func (g *Game) actionsPhaseFinishTurn(ctx context.Context) error {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	s, err := g.validateActionsPhaseFinishTurn(ctx)
	if err != nil {
		return err
	}

	oldCP := g.CurrentPlayer()
	np := g.actionPhaseNextPlayer()
	if np == nil {
		g.orderOfPlay(ctx)
		g.scoreEmpires(ctx)
		if completed := g.expandCityPhase(ctx); !completed {
			return g.save(ctx, s.GetUpdate(ctx, time.Time(g.UpdatedAt)))
		}

		if g.Turn == 5 {
			es := wrap(s.GetUpdate(ctx, time.Time(g.UpdatedAt)), g.endGameScoring(ctx))
			return g.save(ctx, es...)
		} else {
			g.endOfTurn(ctx)
			g.startTurn(ctx)
		}
	} else {
		g.setCurrentPlayers(np)
		if np.Equal(g.Players()[0]) {
			g.Round += 1
		}
	}

	if newCP := g.CurrentPlayer(); newCP != nil && oldCP.ID() != newCP.ID() {
		g.SendTurnNotificationsTo(ctx, newCP)
	}
	restful.AddNoticef(ctx, "%s finished turn.", g.NameFor(oldCP))

	return g.save(ctx, s.GetUpdate(ctx, time.Time(g.UpdatedAt)))
}

func (g *Game) validateActionsPhaseFinishTurn(ctx context.Context) (*stats.Stats, error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	switch s, err := g.validateFinishTurn(ctx); {
	case err != nil:
		return nil, err
	case g.Phase != Actions:
		return nil, sn.NewVError(`Expected "Actions" phase but have %q phase.`, g.Phase)
	default:
		return s, nil
	}
}

func (g *Game) expandCityPhaseNextPlayer(pers ...game.Playerer) (p *Player) {
	ps := g.Players()
	p = g.nextPlayer(pers...)
	for !ps.allVPPassed() {
		if p.VPPassed {
			p = g.nextPlayer(p)
		} else {
			p.beginningOfTurnReset()
			if !p.canAutoVPPass() {
				return
			}
			p.autoVPPass()
		}
	}
	p = nil
	return
}

func (g *Game) expandCityPhaseFinishTurn(ctx context.Context) error {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	s, err := g.validateExpandCityPhaseFinishTurn(ctx)
	if err != nil {
		return err
	}

	cp := g.CurrentPlayer()
	cp.VPPassed = true
	if !g.ExpandedCity {
		cp.newNoCityExpansionEntry()
	}
	restful.AddNoticef(ctx, "%s finished turn.", g.NameFor(cp))

	oldCP := g.CurrentPlayer()
	np := g.expandCityPhaseNextPlayer()
	if np != nil {
		g.setCurrentPlayers(np)
	} else if g.Turn == 5 {
		es := wrap(s.GetUpdate(ctx, time.Time(g.UpdatedAt)), g.endGameScoring(ctx))
		return g.save(ctx, es...)
	} else {
		g.endOfTurn(ctx)
		g.startTurn(ctx)
	}

	if newCP := g.CurrentPlayer(); newCP != nil && oldCP.ID() != newCP.ID() {
		g.SendTurnNotificationsTo(ctx, newCP)
	}
	restful.AddNoticef(ctx, "%s finished turn.", g.NameFor(oldCP))

	return g.save(ctx, s.GetUpdate(ctx, time.Time(g.UpdatedAt)))
}

func (g *Game) validateExpandCityPhaseFinishTurn(ctx context.Context) (*stats.Stats, error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	switch s := stats.Fetched(ctx); {
	case s == nil:
		return nil, sn.NewVError("missing stats for player.")
	case !g.CUserIsCPlayerOrAdmin(ctx):
		return nil, sn.NewVError("Only the current player may finish a turn.")
	case g.Phase != ExpandCity:
		return nil, sn.NewVError(`Expected "Expand City" phase but have %q phase.`, g.PhaseName())
	default:
		return s, nil
	}
}
