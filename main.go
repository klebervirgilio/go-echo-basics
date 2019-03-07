package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	repo := MongoRepo{NewMongoClient()}

	e := echo.New()
	e.HTTPErrorHandler = customHTTPErrorHandler
	e.Renderer = newTemplate()

	// Configure middlewares
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Configure assets endpoit
	e.Static("/assets", "assets")

	e.GET("/full-list", FullListHandler(repo), middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		if username == "golang" && password == "echo!" {
			return true, nil
		}
		return false, nil
	}))

	e.GET("/", HomeHandler)

	e.POST("/save", SubscribeHandler(repo))

	e.DELETE("/delete/:email", func(c echo.Context) error {
		if err := repo.Remove(map[string]interface{}{"email": c.Param("email")}); err != nil {
			return c.String(http.StatusNotFound, err.Error())
		}

		return nil
	})

	e.Logger.Fatal(e.Start(":4000"))
}

func FullListHandler(repo SubscriptionRepo) echo.HandlerFunc {
	return func(c echo.Context) error {
		subscriptions, err := repo.FindAll(map[string]interface{}{})
		if err != nil {
			return err
		}
		return c.Render(http.StatusOK, "full-list.html", ViewContext{
			"subscriptions": subscriptions,
			"success":       c.QueryParam("success"),
			"error":         c.QueryParam("alert"),
			"page":          "full-list",
		})
	}
}

func HomeHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "subscribe.html", ViewContext{
		"page":    "subcribe",
		"success": c.QueryParam("success"),
		"error":   c.QueryParam("alert"),
	})
}

func SubscribeHandler(repo SubscriptionRepo) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := c.FormValue("email")
		fullName := c.FormValue("full-name")

		if email == "" || fullName == "" {
			return c.Render(http.StatusUnprocessableEntity, "subscribe.html", ViewContext{
				"page":     "subcribe",
				"email":    email,
				"fullName": fullName,
				"error":    "Invalid name or e-mail",
			})
		}

		if match, err := regexp.MatchString("@", email); !match || err != nil {
			return c.Render(http.StatusUnprocessableEntity, "subscribe.html", ViewContext{
				"page":     "subcribe",
				"email":    email,
				"fullName": fullName,
				"error":    "Invalid e-mail",
			})
		}

		err := repo.Create(Subscription{email, fullName})
		if err != nil {
			return err
		}

		if hd := c.Request().Header["Authorization"]; len(hd) != 0 {
			return c.Redirect(http.StatusFound, "http://localhost:4000/full-list?success=You have been succesfully subscribed")
		}

		return c.Redirect(http.StatusFound, "http://localhost:4000/?success=You have been succesfully subscribed")
	}

}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplate() echo.Renderer {
	return &Template{
		templates: template.Must(template.ParseGlob("pages/*.html")),
	}
}

type ViewContext map[string]interface{}

type Subscription struct {
	Email string `bson:"email"`
	Name  string `bson:"fullName"`
}

type SubscriptionRepo interface {
	FindAll(selector map[string]interface{}) ([]Subscription, error)
	Remove(selector map[string]interface{}) error
	Create(subscription Subscription) error
}

type MongoRepo struct {
	client MongoClient
}

func (m MongoRepo) FindAll(selector map[string]interface{}) ([]Subscription, error) {
	coll, cs := m.client.GetSession()
	defer cs()

	var subscriptions []Subscription
	err := coll.Find(selector).All(&subscriptions)

	return subscriptions, err
}

func (m MongoRepo) Remove(selector map[string]interface{}) error {
	coll, cs := m.client.GetSession()
	defer cs()

	return coll.Remove(selector)
}

func (m MongoRepo) Create(subscription Subscription) error {
	coll, cs := m.client.GetSession()
	defer cs()

	return coll.Insert(bson.M{"email": subscription.Email, "fullName": subscription.Name})
}

type MongoClient struct {
	session *mgo.Session
}

func NewMongoClient() MongoClient {
	mongoURL := os.Getenv("MONGO_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://mongo_user:mongo_secret@localhost:27017"
	}
	mongo, err := mgo.Dial(mongoURL)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	if err := mongo.Ping(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	return MongoClient{session: mongo}
}

func (m MongoClient) GetSession() (*mgo.Collection, func()) {
	s := m.session.Copy()
	return s.DB("").C("subscriptions"), func() {
		s.Close()
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

	errorPage := fmt.Sprintf("pages/%d.html", code)
	if err := c.File(errorPage); err != nil {
		c.Logger().Error(err)
	}

	c.Logger().Error(err)
}
