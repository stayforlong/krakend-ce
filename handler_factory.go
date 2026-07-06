package krakend

import (
	"fmt"

	"github.com/gin-gonic/gin"
	botdetector "github.com/krakend/krakend-botdetector/v2/gin"
	jose "github.com/krakend/krakend-jose/v2"
	ginjose "github.com/krakend/krakend-jose/v2/gin"
	lua "github.com/krakend/krakend-lua/v2/router/gin"
	metrics "github.com/krakend/krakend-metrics/v2/gin"
	ratelimit "github.com/krakend/krakend-ratelimit/v3/router/gin"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	router "github.com/luraproject/lura/v2/router/gin"
	auth "github.com/stayforlong/krakend-auth"
	ipfilter "github.com/stayforlong/krakend-ipfilter"
	mcpgateway "github.com/stayforlong/krakend-mcp-gateway"
)

// NewHandlerFactory returns a HandlerFactory with a rate-limit and a metrics collector middleware injected.
//
// mcpGateway is deliberately wrapped around the innermost base handler, before
// every other middleware (rate-limit, lua, jose, auth, ipfilter, metrics,
// bot-detector) is layered on top. krakend-mcp-gateway's own HandlerFactory
// fully replaces an endpoint's handler — and never calls its wrapped hf —
// whenever the endpoint carries MCP gateway config, which every real MCP
// endpoint does. Wrapping it around the *whole* chain (as this used to do,
// from the executor) meant that chain, auth included, was silently never
// invoked for any MCP endpoint. Placing it innermost instead means every
// other middleware still runs first, and only the final base-handler step
// is replaced for MCP-gateway-namespaced endpoints.
func NewHandlerFactory(logger logging.Logger, metricCollector *metrics.Metrics, rejecter jose.RejecterFactory, a auth.Authenticator, mcpGateway mcpgateway.MCPGateway) router.HandlerFactory {
	handlerFactory := router.CustomErrorEndpointHandler(logger, ErrorToHTTPError)
	handlerFactory = router.HandlerFactory(mcpGateway.NewHandlerFactory(mcpgateway.HandlerFactory(handlerFactory), logger))
	handlerFactory = ratelimit.NewRateLimiterMw(logger, handlerFactory)
	handlerFactory = lua.HandlerFactory(logger, handlerFactory)
	handlerFactory = ginjose.HandlerFactory(handlerFactory, logger, rejecter)
	handlerFactory = a.NewHandlerFactory(handlerFactory, logger)
	handlerFactory = ipfilter.NewHandlerFactory(handlerFactory, logger)
	handlerFactory = metricCollector.NewHTTPHandlerFactory(handlerFactory)
	handlerFactory = botdetector.New(handlerFactory, logger)

	return func(cfg *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		logger.Debug(fmt.Sprintf("[ENDPOINT: %s] Building the http handler", cfg.Endpoint))
		return handlerFactory(cfg, p)
	}
}

type handlerFactory struct{}

func (handlerFactory) NewHandlerFactory(l logging.Logger, m *metrics.Metrics, r jose.RejecterFactory, a auth.Authenticator, mcpGateway mcpgateway.MCPGateway) router.HandlerFactory {
	return NewHandlerFactory(l, m, r, a, mcpGateway)
}
