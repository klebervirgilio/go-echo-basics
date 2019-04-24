package middlewares

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var RequireAuth = middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
	if username == "golang" && password == "echo!" {
		return true, nil
	}
	return false, nil
})
