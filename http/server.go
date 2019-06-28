package http

import (
	"net/http"

	"github.com/klebervirgilio/go-echo-basics/config"
	"github.com/klebervirgilio/go-echo-basics/mailchecker"
	mongorepository "github.com/klebervirgilio/go-echo-basics/storage"
	"github.com/klebervirgilio/go-echo-basics/core"
	"github.com/klebervirgilio/go-echo-basics/http/middlewares"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func NewServer() *Server {
	cfg := config.New()
	repository := mongorepository.NewMongoRepo(cfg)
	mailChecker := apilayer.New(cfg)

	return &Server{
		repository,
		cfg,
		mailChecker,
	}
}

type Server struct {
	SubscriptionRepository core.Repository
	Config                 *config.Config
	MailChecker            core.MailChecker
}

func (s Server) Serve() {
	e := echo.New()
	e.HTTPErrorHandler = customHTTPErrorHandler
	e.Renderer = newTemplate(e)

	// Configure middlewares
	e.Use(middleware.Logger())
	// e.Use(middleware.Recover())

	// Configure assets endpoint
	e.Static("/assets", "assets")
	e.GET("/", HomeHandler).Name = "root"
	e.POST("/subscribe", SubscribeHandler(s.SubscriptionRepository, e)).Name = "subscribe"

	// Echo Groups/Nested Routes
	g := e.Group("/subscriptions", middlewares.RequireAuth)
	g.GET("/", FullListHandler(s.SubscriptionRepository), middlewares.RequireAuth).Name = "subscriptions"
	g.GET("/validate", checkEmailHandler(s.SubscriptionRepository, e, s.MailChecker, s.Config)).Name = "validate-all-subscriptions"

	// Nesting even more...
	g = g.Group("/:email")
	g.GET("/validate", checkEmailHandler(s.SubscriptionRepository, e, s.MailChecker, s.Config)).Name = "validate-email"
	g.DELETE("/", func(c echo.Context) error {
		if err := s.SubscriptionRepository.Remove(map[string]interface{}{"email": c.Param("email")}); err != nil {
			return c.String(http.StatusNotFound, err.Error())
		}
		return nil
	}).Name = "delete-email"

	e.Logger.Fatal(e.Start(s.Config.GetString("bindAddr")))
}
