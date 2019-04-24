package http

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
)

func redirectWithFlashMessage(c echo.Context, e *echo.Echo, routeName, msgType, msg string) error {
	path := e.Reverse(routeName)
	return c.Redirect(http.StatusFound, fmt.Sprintf("http://%s%s?%s=%s", c.Request().Host, path, msgType, msg))
}
