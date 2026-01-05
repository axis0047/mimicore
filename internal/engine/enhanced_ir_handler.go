package engine

// EnhancedIRHandler wraps the enhanced router
type EnhancedIRHandler struct {
	Router *EnhancedRouter
}

func (h *EnhancedIRHandler) Serve(ctx *Context) {
	h.Router.ServeHTTP(ctx.W, ctx.R)
}
