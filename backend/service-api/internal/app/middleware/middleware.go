package middleware

type MiddlewareProvider interface {
}

type middleware struct {
}

func NewMiddleware() MiddlewareProvider {
	return &middleware{}
}
