package ealthread

import (
	"context"
	"strconv"
	"time"

	"github.com/functionalfoundry/graphqlws"
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver/gqlsub"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// GraphQL types.
var (
	GqlWorkerNodeType *gqlserver.NodeType
	GqlWorkerType     *graphql.Object
	GqlLoadStatType   *graphql.Object
)

func init() {
	GqlWorkerNodeType = gqlserver.NewNodeType(eal.LCore{})
	GqlWorkerNodeType.Retrieve = func(id string) (interface{}, error) {
		nid, e := strconv.Atoi(id)
		if e != nil {
			return nil, e
		}
		for _, lc := range eal.Workers {
			if lc.ID() == nid {
				return lc, nil
			}
		}
		return nil, nil
	}

	GqlWorkerType = graphql.NewObject(GqlWorkerNodeType.Annotate(graphql.ObjectConfig{
		Name: "Worker",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Description: "Numeric LCore ID.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					lc := p.Source.(eal.LCore)
					return lc.ID(), nil
				},
			},
			"isBusy": &graphql.Field{
				Description: "Whether the LCore is running.",
				Type:        gqlserver.NonNullBoolean,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					lc := p.Source.(eal.LCore)
					return lc.IsBusy(), nil
				},
			},
			"role": &graphql.Field{
				Description: "Assigned role.",
				Type:        graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					lc := p.Source.(eal.LCore)
					return gqlserver.Optional(allocated[lc.ID()]), nil
				},
			},
			"numaSocket": eal.GqlWithNumaSocket,
		},
	}))
	GqlWorkerNodeType.Register(GqlWorkerType)

	gqlserver.AddQuery(&graphql.Field{
		Name:        "workers",
		Description: "Worker LCore allocations.",
		Type:        gqlserver.NewNonNullList(GqlWorkerType),
		Args: graphql.FieldConfigArgument{
			"role": &graphql.ArgumentConfig{
				Type:        graphql.String,
				Description: "Filter by assigned role. Empty string matches unassigned LCores.",
			},
			"numaSocket": &graphql.ArgumentConfig{
				Type:        graphql.Int,
				Description: "Filter by NUMA socket.",
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			pred := []eal.LCorePredicate{}
			if role, ok := p.Args["role"].(string); ok {
				pred = append(pred, lcAllocatedTo(role))
			}
			if numaSocket, ok := p.Args["numaSocket"].(int); ok {
				pred = append(pred, eal.LCoreOnNumaSocket(eal.NumaSocketFromID(numaSocket)))
			}
			return eal.Workers.Filter(pred...), nil
		},
	})

	GqlLoadStatType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "ThreadLoadStat",
		Fields: gqlserver.BindFields(LoadStat{}, nil),
	})

	gqlserver.AddSubscription(&graphql.Field{
		Name:        "threadLoadStat",
		Description: "Obtain thread load statistics.",
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "Worker ID.",
				Type:        gqlserver.NonNullID,
			},
			"interval": &graphql.ArgumentConfig{
				Description: "Interval between updates.",
				Type:        nnduration.GqlNanoseconds,
			},
		},
		Type: GqlLoadStatType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return p.Info.RootValue.(LoadStat), nil
		},
	}, func(ctx context.Context, sub *graphqlws.Subscription, updates chan<- interface{}) {
		defer close(updates)

		id, ok := gqlsub.GetArg(sub, "id", graphql.ID).(string)
		if !ok {
			return
		}
		var lc eal.LCore
		if e := gqlserver.RetrieveNodeOfType(GqlWorkerNodeType, id, &lc); e != nil {
			return
		}
		thObj, ok := activeThread.Load(lc)
		if !ok {
			return
		}
		th, ok := thObj.(ThreadWithLoadStat)
		if !ok {
			return
		}

		interval, ok := gqlsub.GetArg(sub, "interval", nnduration.GqlNanoseconds).(nnduration.Nanoseconds)
		if !ok {
			return
		}

		prev := th.ThreadLoadStat()
		ticker := time.NewTicker(interval.Duration())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-th.threadImpl().stopped:
				return
			case <-ticker.C:
				stat := th.ThreadLoadStat()
				updates <- stat.Sub(prev)
				prev = stat
			}
		}
	})
}

// GqlWithWorker is a GraphQL field for source object that implements Thread.
// get is a function that returns a Thread; if nil, p.Source must implement Thread.
func GqlWithWorker(get func(p graphql.ResolveParams) Thread) *graphql.Field {
	return &graphql.Field{
		Type:        GqlWorkerType,
		Name:        "worker",
		Description: "Worker lcore.",
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var thread Thread
			if get == nil {
				thread = p.Source.(Thread)
			} else {
				thread = get(p)
			}
			if thread == nil {
				return nil, nil
			}

			lc := thread.LCore()
			return gqlserver.Optional(lc, lc.Valid()), nil
		},
	}
}
