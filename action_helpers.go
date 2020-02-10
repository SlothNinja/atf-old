package atf

import (
	"strconv"
	"strings"

	"bitbucket.org/SlothNinja/slothninja-games/sn/restful"
	"golang.org/x/net/context"
)

func getPaidResource(ctx context.Context) (rv Resource) {
	if rv := restful.GinFrom(ctx).PostForm("paid-resource"); rv == "" {
		return noResource
	} else {
		return toResource(rv)
	}
}

func getPlacedArmies(ctx context.Context) int {
	if a, err := strconv.Atoi(restful.GinFrom(ctx).PostForm("placed-armies")); err == nil {
		return a
	}
	return 0
}

func getPlaceWorkers(ctx context.Context) int {
	if w, err := strconv.Atoi(restful.GinFrom(ctx).PostForm("place-workers")); err == nil {
		return w
	}
	return 0
}

func getAreaID(ctx context.Context) AreaID {
	return toAreaID(restful.GinFrom(ctx).PostForm("area"))
}

func getTrades(ctx context.Context) (gave, received Resources) {
	gave = make(Resources, 8)
	received = make(Resources, 8)
	for i, s := range resourceStrings {
		key := strings.ToLower(s) + "-traded-resource"
		if res := restful.GinFrom(ctx).PostForm(key); res != "" && res != "none" {
			gave[toResource(res)] += 1
			received[i] += 1
		}
	}
	return
}
