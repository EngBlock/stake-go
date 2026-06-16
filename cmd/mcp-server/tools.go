package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nathanbeddoewebdev/stake-go/pkg/stake"
)

type serverConfig struct {
	EnableWatchlistMutations bool
	EnableOrderCancel        bool
}

func newMCPServer(auth *stakeAuth, cfg serverConfig) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "stake-go",
		Title:   "Stake Go",
		Version: "0.1.0",
	}, &mcp.ServerOptions{
		Instructions: instructions(cfg),
		SchemaCache:  mcp.NewSchemaCache(),
	})

	registerReadOnlyTools(server, auth)
	if cfg.EnableWatchlistMutations {
		registerWatchlistMutationTools(server, auth)
	}
	if cfg.EnableOrderCancel {
		registerOrderCancelTools(server, auth)
	}
	return server
}

func instructions(cfg serverConfig) string {
	instructions := "Use this server to inspect the authenticated Stake account and market data. Trading tools are intentionally not available in this build."
	if cfg.EnableWatchlistMutations {
		instructions += " Watchlist mutation tools are enabled and change Stake watchlists."
	}
	if cfg.EnableOrderCancel {
		instructions += " Order cancellation tools are enabled and cancel pending Stake orders."
	}
	return instructions
}

func addStakeTool[In any](server *mcp.Server, auth *stakeAuth, name, description string, handler func(context.Context, *stake.Client, In) (any, error)) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        name,
		Description: description,
	}, withStakeClient(auth, func(ctx context.Context, req *mcp.CallToolRequest, client *stake.Client, args In) (*mcp.CallToolResult, any, error) {
		result, err := handler(ctx, client, args)
		return nil, result, err
	}))
}

func registerReadOnlyTools(server *mcp.Server, auth *stakeAuth) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "me",
		Description: "Return the authenticated Stake user profile.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args noInput) (*mcp.CallToolResult, any, error) {
		user, err := auth.Login(ctx)
		if err != nil {
			return nil, nil, err
		}
		return nil, output("user", user), nil
	})

	addStakeTool(server, auth, "fx.convert", "Return a Stake FX conversion quote between AUD and USD.", func(ctx context.Context, client *stake.Client, args fxConvertInput) (any, error) {
		request, err := args.toStake()
		if err != nil {
			return nil, err
		}
		conversion, err := client.NYSE.FX.Convert(ctx, request)
		return output("conversion", conversion), err
	})

	addStakeTool(server, auth, "nyse.market.status", "Return the current NYSE market status from Stake.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		status, err := client.NYSE.Market.Get(ctx)
		return output("marketStatus", status), err
	})

	addStakeTool(server, auth, "asx.market.status", "Return the current ASX market status from Stake.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		status, err := client.ASX.Market.Get(ctx)
		return output("marketStatus", status), err
	})

	addStakeTool(server, auth, "nyse.market.is_open", "Report whether Stake considers the NYSE market open.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		open, err := client.NYSE.Market.IsOpen(ctx)
		return output("open", open), err
	})

	addStakeTool(server, auth, "asx.market.is_open", "Report whether Stake considers the ASX market open.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		open, err := client.ASX.Market.IsOpen(ctx)
		return output("open", open), err
	})

	addStakeTool(server, auth, "nyse.positions.list", "List US-market equity positions for the authenticated Stake account.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		positions, err := client.NYSE.Equities.List(ctx)
		return output("positions", positions), err
	})

	addStakeTool(server, auth, "asx.positions.list", "List ASX equity positions for the authenticated Stake account.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		positions, err := client.ASX.Equities.List(ctx)
		return output("positions", positions), err
	})

	addStakeTool(server, auth, "nyse.cash_available", "Return US-market cash availability for the authenticated Stake account.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		cash, err := client.NYSE.Fundings.CashAvailable(ctx)
		return output("cashAvailable", cash), err
	})

	addStakeTool(server, auth, "asx.cash_available", "Return ASX cash availability for the authenticated Stake account.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		cash, err := client.ASX.Fundings.CashAvailable(ctx)
		return output("cashAvailable", cash), err
	})

	addStakeTool(server, auth, "nyse.funds.in_flight", "List US-market funds currently in flight.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		funds, err := client.NYSE.Fundings.InFlight(ctx)
		return output("fundsInFlight", funds), err
	})

	addStakeTool(server, auth, "asx.funds.in_flight", "List ASX funding transactions currently pending or awaiting approval.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		funds, err := client.ASX.Fundings.InFlight(ctx)
		return output("fundsInFlight", funds), err
	})

	addStakeTool(server, auth, "nyse.fundings.list", "List US-market funding transactions.", func(ctx context.Context, client *stake.Client, args nyseTransactionsInput) (any, error) {
		request, err := args.toStake()
		if err != nil {
			return nil, err
		}
		fundings, err := client.NYSE.Fundings.List(ctx, request)
		return output("fundings", fundings), err
	})

	addStakeTool(server, auth, "asx.fundings.list", "List ASX funding transactions.", func(ctx context.Context, client *stake.Client, args asxFundingsInput) (any, error) {
		request, err := args.toStake()
		if err != nil {
			return nil, err
		}
		fundings, err := client.ASX.Fundings.List(ctx, request)
		return output("fundings", fundings), err
	})

	addStakeTool(server, auth, "nyse.transactions.list", "List US-market account transactions.", func(ctx context.Context, client *stake.Client, args nyseTransactionsInput) (any, error) {
		request, err := args.toStake()
		if err != nil {
			return nil, err
		}
		transactions, err := client.NYSE.Transactions.List(ctx, request)
		return output("transactions", transactions), err
	})

	addStakeTool(server, auth, "asx.transactions.list", "List ASX trade activity records.", func(ctx context.Context, client *stake.Client, args asxTransactionsInput) (any, error) {
		request, err := args.toStake()
		if err != nil {
			return nil, err
		}
		transactions, err := client.ASX.Transactions.List(ctx, request)
		return output("transactions", transactions), err
	})

	addStakeTool(server, auth, "nyse.orders.list", "List pending US-market orders.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		orders, err := client.NYSE.Orders.List(ctx)
		return output("orders", orders), err
	})

	addStakeTool(server, auth, "asx.orders.list", "List pending ASX orders.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		orders, err := client.ASX.Orders.List(ctx)
		return output("orders", orders), err
	})

	addStakeTool(server, auth, "nyse.orders.brokerage", "Estimate US-market brokerage for an order amount.", func(ctx context.Context, client *stake.Client, args brokerageInput) (any, error) {
		if args.OrderAmount <= 0 {
			return nil, fmt.Errorf("orderAmount must be greater than zero")
		}
		brokerage, err := client.NYSE.Orders.Brokerage(ctx, args.OrderAmount)
		return output("brokerage", brokerage), err
	})

	addStakeTool(server, auth, "asx.orders.brokerage", "Estimate ASX brokerage for an order amount.", func(ctx context.Context, client *stake.Client, args brokerageInput) (any, error) {
		if args.OrderAmount <= 0 {
			return nil, fmt.Errorf("orderAmount must be greater than zero")
		}
		brokerage, err := client.ASX.Orders.Brokerage(ctx, args.OrderAmount)
		return output("brokerage", brokerage), err
	})

	addStakeTool(server, auth, "nyse.product.get", "Get a US-market product by ticker symbol, including live market data and pre-market pricing.", func(ctx context.Context, client *stake.Client, args symbolInput) (any, error) {
		symbol, err := requireNonEmpty(args.Symbol, "symbol")
		if err != nil {
			return nil, err
		}
		product, err := client.NYSE.Products.GetWithQuote(ctx, symbol)
		return output("product", product), err
	})

	addStakeTool(server, auth, "asx.product.get", "Get an ASX product quote by ticker symbol.", func(ctx context.Context, client *stake.Client, args symbolInput) (any, error) {
		symbol, err := requireNonEmpty(args.Symbol, "symbol")
		if err != nil {
			return nil, err
		}
		product, err := client.ASX.Products.Get(ctx, symbol)
		return output("product", product), err
	})

	addStakeTool(server, auth, "nyse.products.search", "Search US-market instruments by keyword.", func(ctx context.Context, client *stake.Client, args keywordInput) (any, error) {
		keyword, err := requireNonEmpty(args.Keyword, "keyword")
		if err != nil {
			return nil, err
		}
		products, err := client.NYSE.Products.Search(ctx, stake.ProductSearchByName{Keyword: keyword})
		return output("instruments", products), err
	})

	addStakeTool(server, auth, "asx.products.search", "Search ASX instruments by keyword.", func(ctx context.Context, client *stake.Client, args keywordInput) (any, error) {
		keyword, err := requireNonEmpty(args.Keyword, "keyword")
		if err != nil {
			return nil, err
		}
		products, err := client.ASX.Products.Search(ctx, stake.ProductSearchByName{Keyword: keyword})
		return output("instruments", products), err
	})

	addStakeTool(server, auth, "asx.product.depth", "Get ASX aggregated market depth by ticker symbol.", func(ctx context.Context, client *stake.Client, args symbolInput) (any, error) {
		symbol, err := requireNonEmpty(args.Symbol, "symbol")
		if err != nil {
			return nil, err
		}
		depth, err := client.ASX.Products.Depth(ctx, symbol)
		return output("depth", depth), err
	})

	addStakeTool(server, auth, "asx.product.course_of_sales", "Get ASX course-of-sales data by ticker symbol.", func(ctx context.Context, client *stake.Client, args symbolInput) (any, error) {
		symbol, err := requireNonEmpty(args.Symbol, "symbol")
		if err != nil {
			return nil, err
		}
		sales, err := client.ASX.Products.CourseOfSales(ctx, symbol)
		return output("courseOfSales", sales), err
	})

	addStakeTool(server, auth, "nyse.ratings.list", "List analyst ratings for US-market ticker symbols.", func(ctx context.Context, client *stake.Client, args ratingsInput) (any, error) {
		symbols := cleanStrings(args.Symbols)
		if len(symbols) == 0 {
			return nil, fmt.Errorf("symbols is required")
		}
		ratings, err := client.NYSE.Ratings.List(ctx, stake.RatingsRequest{Symbols: symbols, Limit: args.Limit})
		return output("ratings", ratings), err
	})

	addStakeTool(server, auth, "nyse.statements.list", "List US-market fundamentals statements for a ticker symbol.", func(ctx context.Context, client *stake.Client, args statementInput) (any, error) {
		symbol, err := requireNonEmpty(args.Symbol, "symbol")
		if err != nil {
			return nil, err
		}
		startDate, err := parseOptionalTime(args.StartDate, "startDate")
		if err != nil {
			return nil, err
		}
		statements, err := client.NYSE.Statements.List(ctx, stake.StatementRequest{Symbol: symbol, StartDate: startDate})
		return output("statements", statements), err
	})

	addStakeTool(server, auth, "nyse.watchlists.list", "List US-market watchlists.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		watchlists, err := client.NYSE.Watchlists.List(ctx)
		return output("watchlists", watchlists), err
	})

	addStakeTool(server, auth, "asx.watchlists.list", "List ASX watchlists.", func(ctx context.Context, client *stake.Client, args noInput) (any, error) {
		watchlists, err := client.ASX.Watchlists.List(ctx)
		return output("watchlists", watchlists), err
	})

	addStakeTool(server, auth, "nyse.watchlists.get", "Get a US-market watchlist by ID.", func(ctx context.Context, client *stake.Client, args watchlistIDInput) (any, error) {
		id, err := requireNonEmpty(args.ID, "id")
		if err != nil {
			return nil, err
		}
		watchlist, err := client.NYSE.Watchlists.Get(ctx, stake.GetWatchlistRequest{ID: id})
		return output("watchlist", watchlist), err
	})

	addStakeTool(server, auth, "asx.watchlists.get", "Get an ASX watchlist by ID.", func(ctx context.Context, client *stake.Client, args watchlistIDInput) (any, error) {
		id, err := requireNonEmpty(args.ID, "id")
		if err != nil {
			return nil, err
		}
		watchlist, err := client.ASX.Watchlists.Get(ctx, stake.GetWatchlistRequest{ID: id})
		return output("watchlist", watchlist), err
	})
}

func registerWatchlistMutationTools(server *mcp.Server, auth *stakeAuth) {
	addStakeTool(server, auth, "nyse.watchlists.create", "Create a US-market watchlist. This changes Stake account state.", func(ctx context.Context, client *stake.Client, args watchlistCreateInput) (any, error) {
		name, err := requireNonEmpty(args.Name, "name")
		if err != nil {
			return nil, err
		}
		watchlist, err := client.NYSE.Watchlists.Create(ctx, stake.CreateWatchlistRequest{Name: name, Tickers: cleanStrings(args.Tickers)})
		return output("watchlist", watchlist), err
	})

	addStakeTool(server, auth, "asx.watchlists.create", "Create an ASX watchlist. This changes Stake account state.", func(ctx context.Context, client *stake.Client, args watchlistCreateInput) (any, error) {
		name, err := requireNonEmpty(args.Name, "name")
		if err != nil {
			return nil, err
		}
		watchlist, err := client.ASX.Watchlists.Create(ctx, stake.CreateWatchlistRequest{Name: name, Tickers: cleanStrings(args.Tickers)})
		return output("watchlist", watchlist), err
	})

	registerWatchlistUpdateTool(server, auth, "nyse.watchlists.add", "Add tickers to a US-market watchlist. This changes Stake account state.", func(client *stake.Client, ctx context.Context, request stake.UpdateWatchlistRequest) (*stake.Watchlist, error) {
		return client.NYSE.Watchlists.Add(ctx, request)
	})
	registerWatchlistUpdateTool(server, auth, "asx.watchlists.add", "Add tickers to an ASX watchlist. This changes Stake account state.", func(client *stake.Client, ctx context.Context, request stake.UpdateWatchlistRequest) (*stake.Watchlist, error) {
		return client.ASX.Watchlists.Add(ctx, request)
	})
	registerWatchlistUpdateTool(server, auth, "nyse.watchlists.remove", "Remove tickers from a US-market watchlist. This changes Stake account state.", func(client *stake.Client, ctx context.Context, request stake.UpdateWatchlistRequest) (*stake.Watchlist, error) {
		return client.NYSE.Watchlists.Remove(ctx, request)
	})
	registerWatchlistUpdateTool(server, auth, "asx.watchlists.remove", "Remove tickers from an ASX watchlist. This changes Stake account state.", func(client *stake.Client, ctx context.Context, request stake.UpdateWatchlistRequest) (*stake.Watchlist, error) {
		return client.ASX.Watchlists.Remove(ctx, request)
	})

	addStakeTool(server, auth, "nyse.watchlists.delete", "Delete a US-market watchlist. This changes Stake account state.", func(ctx context.Context, client *stake.Client, args watchlistIDInput) (any, error) {
		id, err := requireNonEmpty(args.ID, "id")
		if err != nil {
			return nil, err
		}
		deleted, err := client.NYSE.Watchlists.Delete(ctx, stake.DeleteWatchlistRequest{ID: id})
		return output("deleted", deleted), err
	})

	addStakeTool(server, auth, "asx.watchlists.delete", "Delete an ASX watchlist. This changes Stake account state.", func(ctx context.Context, client *stake.Client, args watchlistIDInput) (any, error) {
		id, err := requireNonEmpty(args.ID, "id")
		if err != nil {
			return nil, err
		}
		deleted, err := client.ASX.Watchlists.Delete(ctx, stake.DeleteWatchlistRequest{ID: id})
		return output("deleted", deleted), err
	})
}

func registerWatchlistUpdateTool(server *mcp.Server, auth *stakeAuth, name, description string, update func(*stake.Client, context.Context, stake.UpdateWatchlistRequest) (*stake.Watchlist, error)) {
	addStakeTool(server, auth, name, description, func(ctx context.Context, client *stake.Client, args watchlistUpdateInput) (any, error) {
		id, err := requireNonEmpty(args.ID, "id")
		if err != nil {
			return nil, err
		}
		tickers := cleanStrings(args.Tickers)
		if len(tickers) == 0 {
			return nil, fmt.Errorf("tickers is required")
		}
		watchlist, err := update(client, ctx, stake.UpdateWatchlistRequest{ID: id, Tickers: tickers})
		return output("watchlist", watchlist), err
	})
}

func registerOrderCancelTools(server *mcp.Server, auth *stakeAuth) {
	addStakeTool(server, auth, "nyse.orders.cancel", "Cancel a pending US-market order. This changes Stake account state.", func(ctx context.Context, client *stake.Client, args orderCancelInput) (any, error) {
		orderID, err := requireNonEmpty(args.OrderID, "orderId")
		if err != nil {
			return nil, err
		}
		if err := client.NYSE.Orders.Cancel(ctx, stake.CancelOrderRequest{OrderID: orderID}); err != nil {
			return nil, err
		}
		return output("cancelled", true), nil
	})

	addStakeTool(server, auth, "asx.orders.cancel", "Cancel a pending ASX order. This changes Stake account state.", func(ctx context.Context, client *stake.Client, args orderCancelInput) (any, error) {
		orderID, err := requireNonEmpty(args.OrderID, "orderId")
		if err != nil {
			return nil, err
		}
		if err := client.ASX.Orders.Cancel(ctx, stake.CancelOrderRequest{OrderID: orderID}); err != nil {
			return nil, err
		}
		return output("cancelled", true), nil
	})
}
