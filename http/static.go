package http

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo"
)

// Template holds the compiled views and serve as a `Renderer` to Echo.
type Template struct {
	templates *template.Template
}

// Render executes the template with a given context.
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	fmt.Println(name, data)
	return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplate(e *echo.Echo) echo.Renderer {
	files, err := filepath.Glob("http/pages/*.html")
	if err != nil || len(files) == 0 {
		log.Fatalf("Fail to load tempates: %s", err)
	}

	var t *template.Template

	for _, file := range files {
		if t == nil {
			t = template.New(file)
		}
		t = template.Must(t.New(file).Funcs(template.FuncMap{
			"urlFor": func(routeName string, params ...interface{}) string {
				return e.Reverse(routeName, params...)
			},
		}).ParseFiles(file))
	}

	return &Template{
		templates: t,
	}
}

func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}

	if code == http.StatusUnauthorized {
		c.HTML(http.StatusUnauthorized, `You don't have access to this page. Please, head back to the <a href="/" >home page</a> .`)
		return
	}

	errorPage := fmt.Sprintf("http/pages/%d.html", code)
	if err := c.File(errorPage); err != nil {
		c.Logger().Error(err)
	}

	c.Logger().Error(err)
}
