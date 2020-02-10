package atf

import (
	"encoding/gob"
	"html/template"

	"bitbucket.org/SlothNinja/slothninja-games/sn/contest"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"golang.org/x/net/context"
)

func init() {
	gob.Register(new(endGameScoringEntry))
}

type endGameScoringMap map[AreaID]int

func (g *Game) endGameScoring(ctx context.Context) contest.Contests {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	g.Phase = EndOfTurn
	m := make(endGameScoringMap, len(scoringIDS()))
	for _, aid := range scoringIDS() {
		a := g.Areas[aid]
		for _, p := range g.Players() {
			if p.hasMostWorkersIn(a) {
				m[a.ID] = p.ID()
				p.Score += a.Score()
			}
		}
	}
	g.newEndGameScoringEntry(m)
	return g.endGame(ctx)
}

type endGameScoringEntry struct {
	*Entry
	Map endGameScoringMap
}

func (g *Game) newEndGameScoringEntry(m endGameScoringMap) {
	e := &endGameScoringEntry{
		Entry: g.newEntry(),
		Map:   m,
	}
	g.Log = append(g.Log, e)
}

func (e *endGameScoringEntry) HTML() template.HTML {
	g := e.Game()
	s := restful.HTML("")
	rows := restful.HTML("")
	count := 0
	scores := make(map[int]int, 3)
	for _, p := range g.Players() {
		scores[p.ID()] = 0
	}
	for aid, pid := range e.Map {
		a := g.Areas[aid]
		row := restful.HTML("<tr>")
		row += restful.HTML("<td>%s</td>", aid)
		for i := range g.Players() {
			if i != pid {
				row += restful.HTML("<td></td>")
			} else {
				scores[i] += a.Score()
				row += restful.HTML("<td>%d</td>", a.Score())
			}
		}
		row += restful.HTML("</tr>")
		count += 1
		rows += row
	}
	if count == 0 {
		s += restful.HTML("No one scored points for their workers.")
	} else {
		s += restful.HTML("<div>Players scored points for their Workers as follows:</div><div>&nbsp;</div>")
		s += restful.HTML("<table class='strippedDataTable'><thead><tr><th>Area</th>")
		for _, p := range g.Players() {
			s += restful.HTML("<th>%s</th>", g.NameFor(p))
		}
		s += restful.HTML("</tr></thead><tbody>")
		s += rows
		s += restful.HTML("</tbody></table>")
	}
	s += "<div>&nbsp;</div>"
	for i, p := range g.Players() {
		s += restful.HTML("<div>%s scored %d points for Workers.</div>", g.NameFor(p), scores[i])
	}
	return s
}
