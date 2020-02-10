package atf

import (
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/type"
	"go.chromium.org/gae/service/datastore"
	"golang.org/x/net/context"
)

const kind = "Game"

func New(ctx context.Context) *Game {
	g := new(Game)
	g.Header = game.NewHeader(ctx, g)
	g.State = newState()
	g.Parent = pk(ctx)
	g.Type = gType.ATF
	return g
}

func newState() *State {
	return new(State)
	//return &State{TempData: new(TempData)}
}

func pk(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, gType.ATF.SString(), "root", 0, game.GamesRoot(ctx))
}

func newKey(ctx context.Context, id int64) *datastore.Key {
	return datastore.NewKey(ctx, kind, "", id, pk(ctx))
}

func (g *Game) NewKey(ctx context.Context, id int64) *datastore.Key {
	return newKey(ctx, id)
}

func (g *Game) init(ctx context.Context) error {
	if err := g.Header.AfterLoad(g); err != nil {
		return err
	}

	for _, player := range g.Players() {
		player.Init(g)
	}

	g.initAreas()
	g.initEmpireTable()

	for _, entry := range g.Log {
		entry.Init(g)
	}
	return nil
}

//func (g *Game) Load(props datastore.PropertyMap) error {
//	h := g.GetHeader()
//	if err := datastore.GetPLS(h).Load(props); err != nil {
//		return err
//	}
//	if err := codec.Decode(g.State, g.SavedState); err != nil {
//		return err
//	}
//	return g.init(g.CTX())
//}
//
//func (g *Game) Save(withMeta bool) (datastore.PropertyMap, error) {
//	g.CTX().Debugf("Entering Save")
//	defer g.CTX().Debugf("Exiting Save")
//
//	g.OrderIDS = make(game.UserIndices, len(g.Players()))
//	for i, p := range g.Players() {
//		g.OrderIDS[i] = p.ID()
//	}
//
//	if saved, err := codec.Encode(g.State); err != nil {
//		return nil, err
//	} else {
//		g.SavedState = saved
//		return datastore.GetPLS(g).Save(withMeta)
//	}
//}

//func (g *Game) Save(c chan<- datastore.Property) error {
//	// Time stamp
//	t := time.Now()
//	if g.CreatedAt.IsZero() {
//		g.CreatedAt = t
//	}
//	g.UpdatedAt = t
//
//	// Set turn order in header
//	g.OrderIDS = make(game.UserIndices, len(g.Players()))
//	for i, p := range g.Players() {
//		g.OrderIDS[i] = p.ID()
//	}
//
//	// Clear TempData
//	// g.TempData = nil
//
//	// Encode and save game state in header
//	if saved, err := codec.Encode(g.State); err != nil {
//		return err
//	} else {
//		g.SavedState = saved
//		return datastore.SaveStruct(g.GetHeader(), c)
//	}
//}

//func (g *Game) Load(c <-chan datastore.Property) error {
//	h := g.GetHeader()
//	if err := datastore.LoadStruct(h, c); err != nil {
//		return err
//	}
//	if err := codec.Decode(g.State, g.SavedState); err != nil {
//		return err
//	}
//	return g.init(g.CTX())
//}

func (g *Game) AfterCache() error {
	return g.init(g.CTX())
}

func copyGame(g Game) *Game {
	g1 := g
	return &g1
}
