package atf

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bitbucket.org/SlothNinja/slothninja-games/sn"
	"bitbucket.org/SlothNinja/slothninja-games/sn/codec"
	"bitbucket.org/SlothNinja/slothninja-games/sn/color"
	"bitbucket.org/SlothNinja/slothninja-games/sn/contest"
	"bitbucket.org/SlothNinja/slothninja-games/sn/game"
	"bitbucket.org/SlothNinja/slothninja-games/sn/log"
	"bitbucket.org/SlothNinja/slothninja-games/sn/mlog"
	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"bitbucket.org/SlothNinja/slothninja-games/sn/type"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user"
	"bitbucket.org/SlothNinja/slothninja-games/sn/user/stats"
	"github.com/gin-gonic/gin"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/gae/service/info"
	"go.chromium.org/gae/service/memcache"
	"golang.org/x/net/context"
)

const (
	gameKey   = "Game"
	homePath  = "/"
	jsonKey   = "JSON"
	statusKey = "Status"
	hParam    = "hid"
)

func gameFrom(ctx context.Context) (g *Game) {
	g, _ = ctx.Value(gameKey).(*Game)
	return
}

func withGame(c *gin.Context, g *Game) *gin.Context {
	c.Set(gameKey, g)
	return c
}

func jsonFrom(ctx context.Context) (g *Game) {
	g, _ = ctx.Value(jsonKey).(*Game)
	return
}

func withJSON(c *gin.Context, g *Game) *gin.Context {
	c.Set(jsonKey, g)
	return c
}

func (g *Game) Update(ctx context.Context) (tmpl string, t game.ActionType, err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	c := restful.GinFrom(ctx)
	switch a := c.PostForm("action"); a {
	case "select-area":
		return g.selectArea(ctx)
	case "build-city":
		return g.buildCity(ctx)
	case "buy-armies":
		return g.buyArmies(ctx)
	case "equip-army":
		return g.equipArmy(ctx)
	case "place-armies":
		return g.placeArmies(ctx)
	case "place-workers":
		return g.placeWorkers(ctx)
	case "trade-resource":
		return g.tradeResource(ctx)
	case "use-scribe":
		return g.useScribe(ctx)
	case "from-stock":
		return g.fromStock(ctx)
	case "make-tool":
		return g.makeTool(ctx)
	case "start-empire":
		return g.startEmpire(ctx)
	case "cancel-start-empire":
		return g.cancelStartEmpire(ctx)
	case "confirm-start-empire":
		return g.confirmStartEmpire(ctx)
	case "invade-area":
		return g.invadeArea(ctx)
	case "invade-area-warning":
		return g.invadeAreaWarning(ctx)
	case "cancel-invasion":
		return g.cancelInvasion(ctx)
	case "confirm-invasion":
		return g.confirmInvasion(ctx)
	case "reinforce-army":
		return g.reinforceArmy(ctx)
	case "destroy-city":
		return g.destroyCity(ctx)
	case "pass":
		return g.pass(ctx)
	case "pay-action-cost":
		return g.payActionCost(ctx)
	case "expand-city":
		return g.expandCity(ctx)
	case "abandon-city":
		return g.abandonCity(ctx)
	case "admin-header":
		return g.adminHeader(ctx)
	case "admin-sumer-area":
		return g.adminSumerArea(ctx)
	case "admin-non-sumer-area":
		return g.adminNonSumerArea(ctx)
	case "admin-worker-box":
		return g.adminWorkerBox(ctx)
	case "admin-player":
		return g.adminPlayer(ctx)
	case "admin-supply-table":
		return g.adminSupplyTable(ctx)
	default:
		return "atf/flash_notice", game.None, sn.NewVError("%v is not a valid action.", a)
	}
}

func newGamer(ctx context.Context) game.Gamer {
	return New(ctx)
}

func Show(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		g := gameFrom(ctx)
		cu := user.CurrentFrom(ctx)
		c.HTML(http.StatusOK, prefix+"/show", gin.H{
			"Context":    ctx,
			"VersionID":  info.VersionID(ctx),
			"CUser":      cu,
			"Game":       g,
			"IsAdmin":    user.IsAdmin(ctx),
			"Admin":      game.AdminFrom(ctx),
			"MessageLog": mlog.From(ctx),
			"ColorMap":   color.MapFrom(ctx),
			"Notices":    restful.NoticesFrom(ctx),
			"Errors":     restful.ErrorsFrom(ctx),
		})
	}
}

func Update(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		var (
			g          *Game
			template   string
			actionType game.ActionType
			err        error
		)

		if g = gameFrom(ctx); g == nil {
			log.Errorf(ctx, "Controller#Update Game Not Found")
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}

		switch template, actionType, err = g.Update(ctx); {
		case err != nil && sn.IsVError(err):
			restful.AddErrorf(ctx, "%v", err)
			withJSON(c, g)
		case err != nil:
			log.Errorf(ctx, err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		case actionType == game.Cache:
			mkey := g.UndoKey(ctx)
			item := memcache.NewItem(ctx, mkey).SetExpiration(time.Minute * 30)

			var v []byte
			if v, err = codec.Encode(g); err != nil {
				log.Errorf(ctx, "codec.Encode error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
			item.SetValue(v)
			if err = memcache.Set(ctx, item); err != nil {
				log.Errorf(ctx, "memcache.Set error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		case actionType == game.Save:
			if err = g.save(ctx); err != nil {
				log.Errorf(ctx, "save error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		case actionType == game.Undo:
			mkey := g.UndoKey(ctx)
			if err := memcache.Delete(ctx, mkey); err != nil && err != memcache.ErrCacheMiss {
				log.Errorf(ctx, "memcache.Delete error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		}

		switch jData := jsonFrom(ctx); {
		case jData != nil && template == "json":
			log.Debugf(ctx, "jData: %v", jData)
			log.Debugf(ctx, "template: %v", template)
			c.JSON(http.StatusOK, jData)
		case template == "":
			log.Debugf(ctx, "template: %v", template)
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
		default:
			log.Debugf(ctx, "template: %v", template)
			log.Debugf(ctx, "notices: %v", restful.NoticesFrom(ctx))
			cu := user.CurrentFrom(ctx)
			c.HTML(http.StatusOK, template, gin.H{
				"Context":   ctx,
				"VersionID": info.VersionID(ctx),
				"CUser":     cu,
				"Game":      g,
				"Admin":     game.AdminFrom(ctx),
				"IsAdmin":   user.IsAdmin(ctx),
				"Notices":   restful.NoticesFrom(ctx),
				"Errors":    restful.ErrorsFrom(ctx),
			})
		}
	}
}
func NewAction(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		g := New(ctx)
		withGame(c, g)
		if err := g.FromParams(ctx, gType.GOT); err != nil {
			log.Errorf(ctx, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		c.HTML(http.StatusOK, prefix+"/new", gin.H{
			"Context":   ctx,
			"VersionID": info.VersionID(ctx),
			"CUser":     user.CurrentFrom(ctx),
			"Game":      g,
		})
	}
}

func Create(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)

		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := New(ctx)
		withGame(c, g)

		var err error
		if err = g.FromParams(ctx, g.Type); err == nil {
			g.NumPlayers = 3
			err = g.encode(ctx)
		}

		if err == nil {
			err = datastore.RunInTransaction(ctx, func(tc context.Context) (err error) {
				if err = datastore.Put(tc, g.Header); err != nil {
					return
				}

				m := mlog.New()
				m.ID = g.ID
				return datastore.Put(tc, m)

			}, &datastore.TransactionOptions{XG: true})
		}

		if err == nil {
			restful.AddNoticef(ctx, "<div>%s created.</div>", g.Title)
		} else {
			log.Errorf(ctx, err.Error())
		}
	}
}

func Accept(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "game not found")
			return
		}

		var (
			start bool
			err   error
		)

		u := user.CurrentFrom(ctx)
		if start, err = g.Accept(ctx, u); err == nil && start {
			err = g.Start(ctx)
		}

		if err == nil {
			err = g.save(ctx)
		}

		if err == nil && start {
			g.SendTurnNotificationsTo(ctx, g.CurrentPlayer())
		}

		if err != nil {
			log.Errorf(ctx, err.Error())
		}

	}
}

func Drop(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "game not found")
			return
		}

		var err error

		u := user.CurrentFrom(ctx)
		if err = g.Drop(u); err == nil {
			err = g.save(ctx)
		}

		if err != nil {
			log.Errorf(ctx, err.Error())
			restful.AddErrorf(ctx, err.Error())
		}

	}
}

func Fetch(c *gin.Context) {
	ctx := restful.ContextFrom(c)
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	// create Gamer
	log.Debugf(ctx, "hid: %v", c.Param("hid"))
	id, err := strconv.ParseInt(c.Param("hid"), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	log.Debugf(ctx, "id: %v", id)
	g := New(ctx)
	g.ID = id

	switch action := c.PostForm("action"); {
	case action == "reset":
		// same as undo & !MultiUndo
		fallthrough
	case action == "undo":
		// pull from memcache/datastore
		if err := dsGet(ctx, g); err != nil {
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
	default:
		if user.CurrentFrom(c) != nil {
			// pull from memcache and return if successful; otherwise pull from datastore
			if err := mcGet(ctx, g); err == nil {
				return
			}
		}

		log.Debugf(ctx, "g: %#v", g)
		log.Debugf(ctx, "k: %v", datastore.KeyForObj(ctx, g.Header))
		if err := dsGet(ctx, g); err != nil {
			log.Debugf(ctx, "dsGet error: %v", err)
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
	}
}

// pull temporary game state from memcache.  Note may be different from value stored in datastore.
func mcGet(ctx context.Context, g *Game) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	var item memcache.Item

	mkey := g.GetHeader().UndoKey(ctx)
	if item, err = memcache.GetKey(ctx, mkey); err != nil {
		return
	}

	if err = codec.Decode(g, item.Value()); err != nil {
		return
	}

	if err = g.AfterCache(); err != nil {
		return
	}

	color.WithMap(withGame(restful.GinFrom(ctx), g), g.ColorMapFor(user.CurrentFrom(ctx)))
	return
}

// pull game state from memcache/datastore.  returned memcache should be same as datastore.
func dsGet(ctx context.Context, g *Game) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	switch err = datastore.Get(ctx, g.Header); {
	case err != nil:
		restful.AddErrorf(ctx, err.Error())
		return
	case g == nil:
		err = fmt.Errorf("Unable to get game for id: %v", g.ID)
		restful.AddErrorf(ctx, err.Error())
		return
	}

	s := newState()
	if err = codec.Decode(&s, g.SavedState); err != nil {
		restful.AddErrorf(ctx, err.Error())
		return
	} else {
		g.State = s
	}

	if err = g.init(ctx); err != nil {
		log.Debugf(ctx, "g.init error: %v", err)
		restful.AddErrorf(ctx, err.Error())
		return
	}

	cm := g.ColorMapFor(user.CurrentFrom(ctx))
	log.Debugf(ctx, "cm: %#v", cm)
	color.WithMap(withGame(restful.GinFrom(ctx), g), cm)
	return
}

func JSON(c *gin.Context) {
	c.JSON(http.StatusOK, gameFrom(c))
}

func JSONIndexAction(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		game.JSONIndexAction(c)
	}
}

//func Fetch(ctx *restful.Context, render render.Render, routes martini.Routes, params martini.Params, form url.Values) {
//	ctx.Debugf("Entering Fetch")
//	defer ctx.Debugf("Exiting Fetch")
//	// create Gamer
//	id, err := strconv.ParseInt(params["gid"], 10, 64)
//	if err != nil {
//		render.Redirect(routes.URLFor("home"), http.StatusSeeOther)
//	}
//
//	g := New(ctx)
//	g.ID = id
//
//	switch action := form.Get("action"); {
//	case action == "reset":
//		// pull from memcache/datastore
//		// same as undo & !MultiUndo
//		fallthrough
//	case action == "undo":
//		// pull from memcache/datastore
//		if err := dsGet(ctx, g); err != nil {
//			render.Redirect(routes.URLFor("home"), http.StatusSeeOther)
//			return
//		}
//	default:
//		// pull from memcache and return if successful; otherwise pull from datastore
//		if err := mcGet(ctx, g); err == nil {
//			ctx.Debugf("mcGet header:%#v\nstate:%#v\n", g.Header, g.State)
//			return
//		}
//		if err := dsGet(ctx, g); err != nil {
//			render.Redirect(routes.URLFor("home"), http.StatusSeeOther)
//			return
//		}
//	}
//}
//
//// pull temporary game state from memcache.  Note may be different from value stored in datastore.
//func mcGet(ctx *restful.Context, g *Game) error {
//	ctx.Debugf("Entering got#mcGet")
//	defer ctx.Debugf("Exiting got#mcGet")
//
//	mkey := g.UndoKey(ctx)
//	item, err := memcache.GetKey(ctx, mkey)
//	if err != nil {
//		return err
//	}
//
//	if err := codec.Decode(g, item.Value()); err != nil {
//		return err
//	}
//
//	if err := g.AfterCache(); err != nil {
//		return err
//	}
//
//	ctx.Data["Game"] = g
//	ctx.Data["ColorMap"] = g.ColorMapFor(user.Current(ctx))
//	ctx.Debugf("Data: %#v", ctx.Data)
//	return nil
//}
//
//// pull game state from memcache/datastore.  returned memcache should be same as datastore.
//func dsGet(ctx *restful.Context, g *Game) error {
//	ctx.Debugf("Entering got#dsGet")
//	defer ctx.Debugf("Exiting got#dsGet")
//
//	switch err := datastore.Get(ctx, g.Header); {
//	case err != nil:
//		ctx.AddErrorf(err.Error())
//		return err
//	case g == nil:
//		err := fmt.Errorf("Unable to get game for id: %v", g.ID)
//		ctx.AddErrorf(err.Error())
//		return err
//	}
//
//	ctx.Debugf("len(g.SavedState): %v", len(g.SavedState))
//
//	s := newState()
//	if err := codec.Decode(&s, g.SavedState); err != nil {
//		ctx.AddErrorf(err.Error())
//		return err
//	} else {
//		ctx.Debugf("State: %#v", s)
//		g.State = s
//	}
//
//	if err := g.init(ctx); err != nil {
//		ctx.AddErrorf(err.Error())
//		return err
//	}
//
//	ctx.Data["Game"] = g
//	ctx.Data["ColorMap"] = g.ColorMapFor(user.Current(ctx))
//	ctx.Debugf("Data: %#v", ctx.Data)
//	return nil
//}
//
//func JSON(ctx *restful.Context, render render.Render) {
//	render.JSON(http.StatusOK, ctx.Data["Game"])
//}
//
//// playback command stack up to current level but adjusted by adj
//func playBack(ctx *restful.Context, g *Game, adj int) error {
//	ctx.Debugf("Entering playBack")
//	defer ctx.Debugf("Exiting playBack")
//
//	stack := new(undo.Stack)
//	ctx.Data["Undo"] = stack
//	mkey := g.UndoKey(ctx)
//	item, err := memcache.GetKey(ctx, mkey)
//	if err != nil {
//		return err
//	}
//	if err := codec.Decode(stack, item.Value()); err == nil {
//		stop := stack.Current + adj
//		switch {
//		case stop < 0:
//			stop = 0
//		case stop > stack.Count():
//			stop = stack.Count()
//		}
//		for i := 0; i < stop; i++ {
//			entry := stack.Entries[i]
//			if _, _, err := g.Update(ctx, entry.Values); err != nil {
//				ctx.AddErrorf("Unexpected error.  Reset turn and try again.")
//				ctx.Errorf("Fetch Error: %#v", err)
//				return err
//			}
//		}
//	}
//	return nil
//}

func showPath(prefix, sid string) string {
	return fmt.Sprintf("/%s/game/show/%s", prefix, sid)
}

func recruitingPath(prefix string) string {
	return fmt.Sprintf("/%s/games/recruiting", prefix)
}

func newPath(prefix string) string {
	return fmt.Sprintf("/%s/game/new", prefix)
}

func (g *Game) save(ctx context.Context, es ...interface{}) (err error) {
	err = datastore.RunInTransaction(ctx, func(tc context.Context) (terr error) {
		oldG := New(tc)
		if ok := datastore.PopulateKey(oldG.Header, datastore.KeyForObj(tc, g.Header)); !ok {
			terr = fmt.Errorf("Unable to populate game with key.")
			return
		}

		if terr = datastore.Get(tc, oldG.Header); terr != nil {
			return
		}

		if oldG.UpdatedAt != g.UpdatedAt {
			terr = fmt.Errorf("Game state changed unexpectantly.  Try again.")
			return
		}

		if terr = g.encode(ctx); terr != nil {
			return
		}

		if terr = datastore.Put(tc, append(es, g.Header)); terr != nil {
			return
		}

		if terr = memcache.Delete(tc, g.UndoKey(tc)); terr == memcache.ErrCacheMiss {
			terr = nil
		}
		return
	}, &datastore.TransactionOptions{XG: true})
	return
}

func (g *Game) encode(ctx context.Context) (err error) {
	log.Debugf(ctx, "Entering")
	defer log.Debugf(ctx, "Exiting")

	var encoded []byte
	if encoded, err = codec.Encode(g.State); err != nil {
		return
	}
	g.SavedState = encoded
	g.updateHeader()

	return
}

func wrap(s *stats.Stats, cs contest.Contests) (es []interface{}) {
	es = make([]interface{}, len(cs)+1)
	es[0] = s
	for i, c := range cs {
		es[i+1] = c
	}
	return
}

//func (g *Game) saveAndUpdateStats(c *gin.Context) error {
//	ctx := restful.ContextFrom(c)
//	cu := user.CurrentFrom(c)
//	s, err := stats.ByUser(c, cu)
//	if err != nil {
//		return err
//	}
//
//	return datastore.RunInTransaction(ctx, func(tc context.Context) error {
//		c = restful.WithContext(c, tc)
//		oldG := New(c)
//		if ok := datastore.PopulateKey(oldG.Header, datastore.KeyForObj(tc, g.Header)); !ok {
//			return fmt.Errorf("Unable to populate game with key.")
//		}
//		if err := datastore.Get(tc, oldG.Header); err != nil {
//			return err
//		}
//
//		if oldG.UpdatedAt != g.UpdatedAt {
//			return fmt.Errorf("Game state changed unexpectantly.  Try again.")
//		}
//
//		//g.TempData = nil
//		if encoded, err := codec.Encode(g.State); err != nil {
//			return err
//		} else {
//			g.SavedState = encoded
//		}
//
//		es := []interface{}{s, g.Header}
//		if err := datastore.Put(tc, es); err != nil {
//			return err
//		}
//		if err := memcache.Delete(tc, g.UndoKey(c)); err != nil && err != memcache.ErrCacheMiss {
//			return err
//		}
//		return nil
//	}, &datastore.TransactionOptions{XG: true})
//}

func Undo(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))

		g := gameFrom(ctx)
		if g == nil {
			log.Errorf(ctx, "game not found")
			return
		}
		mkey := g.UndoKey(ctx)
		if err := memcache.Delete(ctx, mkey); err != nil && err != memcache.ErrCacheMiss {
			log.Errorf(ctx, "memcache.Delete error: %s", err)
		}
		restful.AddNoticef(ctx, "%s undid turn.", user.CurrentFrom(ctx))
	}
}

func Index(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := restful.ContextFrom(c)
		log.Debugf(ctx, "Entering")
		defer log.Debugf(ctx, "Exiting")

		gs := game.GamersFrom(ctx)
		switch status := game.StatusFrom(ctx); status {
		case game.Recruiting:
			c.HTML(http.StatusOK, "shared/invitation_index", gin.H{
				"Context":   ctx,
				"VersionID": info.VersionID(ctx),
				"CUser":     user.CurrentFrom(ctx),
				"Games":     gs,
				"Type":      gType.ATF.String(),
			})
		default:
			c.HTML(http.StatusOK, "shared/games_index", gin.H{
				"Context":   ctx,
				"VersionID": info.VersionID(ctx),
				"CUser":     user.CurrentFrom(ctx),
				"Games":     gs,
				"Type":      gType.ATF.String(),
				"Status":    status,
			})
		}
	}
}

func (g *Game) updateHeader() {
	switch g.Phase {
	case GameOver:
		g.Progress = g.PhaseName()
	default:
		g.Progress = fmt.Sprintf("<div>Turn: %d | Round: %d</div><div>Phase: %s</div>", g.Turn, g.Round, g.PhaseName())
	}
	if u := g.Creator; u != nil {
		g.CreatorSID = user.GenID(u.GoogleID)
		g.CreatorName = u.Name
	}

	if l := len(g.Users); l > 0 {
		g.UserSIDS = make([]string, l)
		g.UserNames = make([]string, l)
		g.UserEmails = make([]string, l)
		for i, u := range g.Users {
			g.UserSIDS[i] = user.GenID(u.GoogleID)
			g.UserNames[i] = u.Name
			g.UserEmails[i] = u.Email
		}
	}

}
