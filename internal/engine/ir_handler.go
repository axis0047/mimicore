package engine

type IRHandler struct {
	Router *Router
}

func (h *IRHandler) Serve(ctx *Context) {
	h.Router.ServeHTTP(ctx.W, ctx.R)
}
