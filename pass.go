package atf

import (
	"encoding/gob"
	"fmt"
	"html/template"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"golang.org/x/net/context"
)

func init() {
	gob.Register(new(passEntry))
}

func (g *Game) pass(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	if err = g.validatePass(ctx); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.Passed = true
	cp.PerformedAction = true

	for resource, count := range cp.PassedResources {
		cp.Resources[resource] -= count
	}

	// Log Pass
	e := cp.newPassEntry(cp.PassedResources)
	restful.AddNoticef(ctx, string(e.HTML()))

	tmpl, act = "atf/pass_update", game.Cache
	return
}

func (g *Game) validatePass(ctx context.Context) (err error) {
	if err = g.validatePlayerAction(ctx); err != nil {
		return
	}

	cp := g.CurrentPlayer()
	if cp.PerformedAction {
		err = sn.NewVError("You have already performed an action.")
		return
	}

	if cp.PassedResources, err = getResourcesFrom(ctx); err != nil {
		return
	}

	for r, cnt := range cp.PassedResources {
		if cnt > cp.Resources[r] {
			cp.PassedResources = make(Resources, 8)
			err = sn.NewVError("You do not have %d %s.", cnt, r)
			return
		}
	}

	return
}

type passEntry struct {
	*Entry
	Resources Resources
}

func (p *Player) newPassEntry(resources Resources) *passEntry {
	g := p.Game()
	e := &passEntry{
		Entry:     p.newEntry(),
		Resources: resources,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *passEntry) HTML() template.HTML {
	ss := make([]string, 0)
	for r, count := range e.Resources {
		resource := Resource(r)
		if count > 0 {
			ss = append(ss, fmt.Sprintf("%d %s", count, resource.LString()))
		}
	}
	if v := e.Resources.Value(); v == 00 {
		return template.HTML(fmt.Sprintf("%s passed with a turn order bid of 0.", e.Player().Name()))
	}
	return template.HTML(fmt.Sprintf("%s passed and spent %s for a turn order bid of %d.",
		e.Player().Name(), restful.ToSentence(ss), e.Resources.Value()))
}
