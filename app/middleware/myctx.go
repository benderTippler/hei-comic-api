package middleware

import (
	"context"
	"github.com/labstack/echo"
)

type MyCtx struct {
	EchoCtx echo.Context
	Context context.Context
}
