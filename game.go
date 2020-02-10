package atf

import (
	"encoding/gob"
	"errors"
	"html/template"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/color"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/type"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"golang.org/x/net/context"
)

func init() {
	gob.Register(new(setupEntry))
	gob.Register(new(startEntry))
	gob.Register(new(startTurnEntry))
	//	gob.Register(new(Game))
	//	game.Register(gType.ATF, newGamer, PhaseNames, nil)
}

//func Register(m *martini.Martini) {
//func Register(r martini.Router) {
//	gob.Register(new(Game))
//	game.Register(gType.ATF, newGamer, PhaseNames, nil)
//	AddRoutes(gType.ATF.Prefix(), r)
//	//game.AddDefaultRoutes(gType.ATF.Prefix(), r)
//	//m.Use(game.AddDefaultRoutes(gType.ATF.Prefix()))
//}
func Register(t gType.Type, r *gin.Engine) {
	//func Register(m *martini.Martini) {
	gob.Register(new(Game))
	game.Register(t, newGamer, PhaseNames, nil)
	AddRoutes(t.Prefix(), r)
	//m.Use(game.AddDefaultRoutes(gType.GOT.Prefix()))
}

var ErrMustBeGame = errors.New("Resource must have type *Game.")

const NoPlayerID = game.NoPlayerID

type Game struct {
	*game.Header
	*State

	// Non-persistent values
	// They are memcached but ignored by datastore
	// NewLog          sn.GameLog `datastore:"-"`
	BuiltCityAreaID AreaID  `datastore:"-"`
	PlacedWorkers   bool    `datastore:"-"`
	From            string  `datastore:"-"`
	To              string  `datastore:"-"`
	OtherPlayer     *Player `datastore:"-"`
	ExpandedCity    bool    `datastore:"-"`
}

type State struct {
	Playerers      game.Playerers
	Log            game.GameLog
	Resources      Resources `form:"resources"`
	Areas          Areas
	EmpireTable    EmpireTable
	Continue       bool
	MultiAction    MultiActionID
	SelectedAreaID AreaID
}

//type Game struct {
//	*game.Header
//	*State
//}
//
//type State struct {
//	Playerers      game.Playerers
//	Log            game.GameLog
//	Resources      Resources
//	Areas          Areas
//	EmpireTable    EmpireTable
//	Continue       bool
//	MultiAction    MultiActionID
//	SelectedAreaID AreaID
//	*TempData
//}
//
//// Non-persistent values
//// They are memcached but ignored by datastore
//// NewLog          sn.GameLog `datastore:"-"`
//type TempData struct {
//	BuiltCityAreaID AreaID
//	PlacedWorkers   bool
//	From            string
//	To              string
//	OtherPlayer     *Player
//	ExpandedCity    bool
//}

func (g *Game) GetPlayerers() game.Playerers {
	return g.Playerers
}

func (g *Game) Players() (players Players) {
	ps := g.GetPlayerers()
	length := len(ps)
	if length > 0 {
		players = make(Players, length)
		for i, p := range ps {
			players[i] = p.(*Player)
		}
	}
	return
}

func (g *Game) setPlayers(players Players) {
	length := len(players)
	if length > 0 {
		ps := make(game.Playerers, length)
		for i, p := range players {
			ps[i] = p
		}
		g.Playerers = ps
	}
}

type Games []*Game

func (g *Game) Start(ctx context.Context) error {
	g.Status = game.Running
	g.setupPhase(ctx)
	return nil
}

func (g *Game) addNewPlayers() {
	for _, u := range g.Users {
		g.addNewPlayer(u)
	}
}

func (g *Game) setupPhase(ctx context.Context) {
	g.Phase = Setup
	g.addNewPlayers()
	g.createAreas()
	g.EmpireTable = defaultEmpireTable()
	g.initEmpireTable()
	g.Resources = Resources{0, 9, 9, 0, 9, 4, 4, 7}
	g.RandomTurnOrder()
	for _, p := range g.Players() {
		p.newSetupEntry()
	}
	g.start(ctx)
}

type setupEntry struct {
	*Entry
}

func (p *Player) newSetupEntry() *setupEntry {
	g := p.Game()
	e := new(setupEntry)
	e.Entry = p.newEntry()
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *setupEntry) HTML() template.HTML {
	return restful.HTML("%s received 1 wood, 1 metal, 1 tool, 1 oil, 1 gold, and 2 workers.", e.Player().Name())
}

func (g *Game) start(ctx context.Context) {
	g.Phase = StartGame
	g.newStartEntry()
	g.startTurn(ctx)
}

type startEntry struct {
	*Entry
}

func (g *Game) newStartEntry() *startEntry {
	e := new(startEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return e
}

func (e *startEntry) HTML() template.HTML {
	g := e.Game()
	return restful.HTML("Good luck %s, %s, and %s.  Have fun.",
		g.NameFor(g.Players()[0]), g.NameFor(g.Players()[1]), g.NameFor(g.Players()[2]))
}

func (g *Game) startTurn(ctx context.Context) {
	g.Turn += 1
	g.Phase = StartTurn
	g.Round = 1
	cp := g.Players()[0]
	g.setCurrentPlayers(cp)
	g.beginningOfPhaseReset()
	g.newStartTurnEntry()
	g.collectGrainPhase(ctx)
	g.collectTextilePhase(ctx)
	g.collectWorkersPhase(ctx)
	g.resetScribesPhase(ctx)
	g.resetToolMakersPhase(ctx)
	g.declinePhase(ctx)
	g.actionsPhase(ctx)
}

type startTurnEntry struct {
	*Entry
}

func (g *Game) newStartTurnEntry() *startTurnEntry {
	e := new(startTurnEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return e
}

func (e *startTurnEntry) HTML() template.HTML {
	return restful.HTML("Starting Turn %d", e.Turn())
}

func (g *Game) setCurrentPlayers(players ...*Player) {
	var playerers game.Playerers

	switch length := len(players); {
	case length == 0:
		playerers = nil
	case length == 1:
		playerers = game.Playerers{players[0]}
	default:
		playerers = make(game.Playerers, length)
		for i, player := range players {
			playerers[i] = player
		}
	}
	g.SetCurrentPlayerers(playerers...)
}

func (g *Game) PlayerByID(id int) *Player {
	if p := g.PlayererByID(id); p != nil {
		return p.(*Player)
	} else {
		return nil
	}
}

func (g *Game) PlayerBySID(sid string) *Player {
	if p := g.Header.PlayerBySID(sid); p != nil {
		return p.(*Player)
	} else {
		return nil
	}
}

func (g *Game) PlayerByUserID(id int64) *Player {
	if p := g.PlayererByUserID(id); p != nil {
		return p.(*Player)
	} else {
		return nil
	}
}

func (g *Game) PlayerByIndex(index int) *Player {
	if p := g.PlayererByIndex(index); p != nil {
		return p.(*Player)
	} else {
		return nil
	}
}

func (g *Game) PlayerByColor(c color.Color) *Player {
	if p := g.PlayererByColor(c); p != nil {
		return p.(*Player)
	} else {
		return nil
	}
}

func (g *Game) undoAction(ctx context.Context) (tmpl string, err error) {
	return g.undoRedoReset(ctx, "%s undid action.")
}

func (g *Game) redoAction(ctx context.Context) (tmpl string, err error) {
	return g.undoRedoReset(ctx, "%s redid action.")
}

func (g *Game) resetTurn(ctx context.Context) (tmpl string, err error) {
	return g.undoRedoReset(ctx, "%s reset turn.")
}

func (g *Game) undoRedoReset(ctx context.Context, fmt string) (tmpl string, err error) {
	cp := g.CurrentPlayer()
	if !g.CUserIsCPlayerOrAdmin(ctx) {
		return "", sn.NewVError("Only the current player may perform this action.")
	}

	restful.AddNoticef(ctx, fmt, g.NameFor(cp))
	return "", nil
}

func (g *Game) CurrentPlayer() *Player {
	if p := g.CurrentPlayerer(); p != nil {
		return p.(*Player)
	}
	return nil
}

func (g *Game) adminSupplyTable(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	ns := new(State)
	ns.Resources = Resources{0, 9, 9, 0, 9, 4, 4, 7}
	if err = restful.BindWith(ctx, ns, binding.FormPost); err != nil {
		act = game.None
	} else {
		log.Debugf(ctx, "ns: %#v", ns)

		g.Resources = ns.Resources
		act = game.Save
	}
	return
}

func (g *Game) SelectedPlayer() *Player {
	switch g.SelectedAreaID {
	case Player0:
		return g.PlayerByID(0)
	case Player1:
		return g.PlayerByID(1)
	case Player2:
		return g.PlayerByID(2)
	case RedPass:
		return g.PlayerByColor(color.Red)
	case GreenPass:
		return g.PlayerByColor(color.Green)
	case PurplePass:
		return g.PlayerByColor(color.Purple)
	default:
		return nil
	}
}

func (g *Game) anyPassed() bool {
	return g.Players()[0].Passed || g.Players()[1].Passed || g.Players()[2].Passed
}

func (g *Game) adminHeader(ctx context.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	h := game.NewHeader(ctx, nil)
	if err = restful.BindWith(ctx, h, binding.FormPost); err != nil {
		act = game.None
		return
	}

	log.Debugf(ctx, "h: %#v", h)
	g.Title = h.Title
	g.Turn = h.Turn
	g.Phase = h.Phase
	g.SubPhase = h.SubPhase
	g.Round = h.Round
	g.NumPlayers = h.NumPlayers
	g.Password = h.Password
	g.CreatorID = h.CreatorID
	g.UserIDS = h.UserIDS
	g.OrderIDS = h.OrderIDS
	g.CPUserIndices = h.CPUserIndices
	g.WinnerIDS = h.WinnerIDS
	g.Status = h.Status
	act = game.Save
	game.WithAdmin(restful.GinFrom(ctx), true)
	return
}
