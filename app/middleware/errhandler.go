package middleware

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/labstack/echo"
	"hei-comic-api/app/httpio/out"
	validator2 "hei-comic-api/base/validator"
	"net/http"
)

func ErrHandler(err error, ctx echo.Context) {

	var (
		code   = http.StatusInternalServerError
		msg    interface{}
		mycode int
	)

	switch err.(type) {
	case *echo.HTTPError:
		he := err.(*echo.HTTPError)
		code = he.Code
		msg = he.Message
		if he.Internal != nil {
			msg = fmt.Sprintf("%v, %v", err, he.Internal)
		}
	case validator.FieldError:
		code = http.StatusOK
		mycode = 1000
		msg = err.(validator.FieldError).Translate(validator2.Trans)
	case *gerror.Error:
		code = http.StatusOK
		mycode = gerror.Code(err).Code()
		msg = err.Error()
	default:
		if ctx.Echo().Debug {
			msg = err.Error()
		} else {
			msg = http.StatusText(code)
		}
	}
	var result *out.Result
	if _, ok := msg.(string); ok {
		result = &out.Result{
			Code:    mycode,
			Data:    nil,
			Message: msg.(string),
		}
	}

	ctx.Echo().Logger.Error(err)

	// Send response
	if !ctx.Response().Committed {
		if ctx.Request().Method == echo.HEAD { // Issue #608
			err = ctx.NoContent(code)
		} else {
			err = ctx.JSON(code, result)
		}
		if err != nil {
			ctx.Echo().Logger.Error(err)
		}
	}
}
