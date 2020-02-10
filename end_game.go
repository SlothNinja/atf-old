package atf

import (
	"encoding/gob"
	"fmt"
	"html/template"

	"bitbucket.org/SlothNinja/slothninja-games/sn/contest"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/send"
	"go.chromium.org/gae/service/mail"
	"golang.org/x/net/context"
)

func init() {
	gob.Register(new(endGameEntry))
	gob.Register(new(announceTHWinnersEntry))
}

func (g *Game) endGame(ctx context.Context) contest.Contests {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	g.Phase = EndGame
	places := g.determinePlaces(ctx)
	g.SetWinners(places[0])
	g.newEndGameEntry()
	return contest.GenContests(ctx, places)
}

type endGameEntry struct {
	*Entry
}

func (g *Game) newEndGameEntry() {
	e := &endGameEntry{
		Entry: g.newEntry(),
	}
	g.Log = append(g.Log, e)
}

func (e *endGameEntry) HTML() template.HTML {
	return restful.HTML("")
}

func (g *Game) SetWinners(rmap contest.ResultsMap) {
	g.Phase = AnnounceWinners
	g.Status = game.Completed

	g.setCurrentPlayers()
	for key := range rmap {
		p := g.PlayerByUserID(key.IntID())
		g.WinnerIDS = append(g.WinnerIDS, p.ID())
	}

	g.newAnnounceWinnersEntry()
}

func (g *Game) SendEndGameNotifications(ctx context.Context) error {
	g.Phase = GameOver
	g.Status = game.Completed

	ms := make([]*mail.Message, len(g.Players()))
	sender := "webmaster@slothninja.com"
	subject := fmt.Sprintf("SlothNinja Games: After The Flood #%d Has Ended", g.ID)

	var body string
	for _, p := range g.Players() {
		body += fmt.Sprintf("%s scored %d points.\n", g.NameFor(p), p.Score)
	}

	var names []string
	for _, p := range g.Winners() {
		names = append(names, g.NameFor(p))
	}
	body += fmt.Sprintf("\nCongratulations to: %s.", restful.ToSentence(names))

	for i, p := range g.Players() {
		ms[i] = &mail.Message{
			To:      []string{p.User().Email},
			Sender:  sender,
			Subject: subject,
			Body:    body,
		}
	}

	return send.Message(ctx, ms...)
}

type announceTHWinnersEntry struct {
	*Entry
}

func (g *Game) newAnnounceWinnersEntry() *announceTHWinnersEntry {
	e := new(announceTHWinnersEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return e
}

func (e *announceTHWinnersEntry) HTML() template.HTML {
	names := make([]string, len(e.Winners()))
	for i, winner := range e.Winners() {
		names[i] = winner.Name()
	}
	return restful.HTML("Congratulations to: %s.", restful.ToSentence(names))
}

func (g *Game) Winners() Players {
	length := len(g.WinnerIDS)
	if length == 0 {
		return nil
	}
	ps := make(Players, length)
	for i, pid := range g.WinnerIDS {
		ps[i] = g.PlayerByID(pid)
	}
	return ps
}
