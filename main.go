package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

const (
	mongoCollectionName = "subscriptions"
	mongoDatabaseName   = "goEchoBasics"
)

var (
	requireAuth = middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		if username == "golang" && password == "echo!" {
			return true, nil
		}
		return false, nil
	})

	host           string
	mailCheckerURL string
	port           = ":4000"
)

func main() {
	setHost()
	setMailCheckerURL()

	repo := newSubscriptionRepo()

	e := echo.New()
	e.HTTPErrorHandler = customHTTPErrorHandler
	e.Renderer = newTemplate(e)

	// Configure middlewares
	e.Use(middleware.Logger())
	// e.Use(middleware.Recover())

	// Configure assets endpoint
	e.Static("/assets", "assets")
	e.GET("/", HomeHandler).Name = "root"
	e.POST("/subscribe", SubscribeHandler(repo, e)).Name = "subscribe"

	// Echo Groups/Nested Routes
	g := e.Group("/subscriptions", requireAuth)
	g.GET("/", FullListHandler(repo), requireAuth).Name = "subscriptions"
	g.GET("/validate", checkEmailHandler(repo, e)).Name = "validate-all-subscriptions"

	// Nesting even more...
	g = g.Group("/:email")
	g.GET("/validate", checkEmailHandler(repo, e)).Name = "validate-email"
	g.DELETE("/", func(c echo.Context) error {
		if err := repo.Remove(map[string]interface{}{"email": c.Param("email")}); err != nil {
			return c.String(http.StatusNotFound, err.Error())
		}
		return nil
	}).Name = "delete-email"

	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	e.Logger.Fatal(e.Start(port))
}

// checkEmailHandler performs a correctness check on a given email provided via URL parameter or
// runs the check in parallel for all subscriptons email found in the database.
// The handler purposes is to exercise the ability of conditionally use a handler and
// how Go make it easy to achieve concurrency.
func checkEmailHandler(repo SubscriptionRepo, e *echo.Echo) echo.HandlerFunc {
	return func(c echo.Context) error {
		if email := c.Param("email"); email != "" {
			resp, err := checkEmail(email, c)
			if err != nil {
				return err
			}

			subscriptions, err := repo.FindAll(map[string]interface{}{"email": email})
			if err != nil {
				return err
			}
			if len(subscriptions) == 0 {
				return errors.New("Could not find a subscription for the given email")
			}
			subscription := subscriptions[0]
			subscription.EmailVerificationResponse = resp

			err = repo.Upsert(subscription)
			if err != nil {
				return err
			}

			return c.JSON(http.StatusOK, resp)
		}

		subscriptions, err := repo.FindAll(map[string]interface{}{})
		if err != nil {
			return err
		}

		var wg sync.WaitGroup
		errCh := make(chan error, 1)
		doneCh := make(chan struct{}, 1)

		wg.Add(len(subscriptions))
		for _, subscription := range subscriptions {
			go func(sub Subscription) {
				resp, err := checkEmail(sub.Email, c)
				if err != nil {
					errCh <- err
					return
				}

				sub.EmailVerificationResponse = resp

				err = repo.Upsert(sub)
				if err != nil {
					errCh <- err
					return
				}

				wg.Done()
			}(subscription)
		}

		go func() {
			wg.Wait()
			doneCh <- struct{}{}
		}()

		select {
		case err = <-errCh:
			return redirectWithFlashMessage(c, e, "subscriptions", "error", "Not all validations were correctly performed. Error "+err.Error())
		case <-doneCh:
			return redirectWithFlashMessage(c, e, "subscriptions", "success", "All emails have been successfully checked")
		case <-time.After(5 * time.Second):
			return redirectWithFlashMessage(c, e, "subscriptions", "error", "5 seconds have passed, the validation might be still happining. Try to refresh the page later.")
		}
	}
}

// FullListHandler renders the subscriptions.html page.
// The user should able to see all subscriptions when the properly authenticated.
// The handler purposes is to show how dependencies can be injected.
func FullListHandler(repo SubscriptionRepo) echo.HandlerFunc {
	return func(c echo.Context) error {
		subscriptions, err := repo.FindAll(map[string]interface{}{})
		if err != nil {
			return err
		}
		return c.Render(http.StatusOK, "subscriptions.html", ViewContext{
			"subscriptions": subscriptions,
			"success":       c.QueryParam("success"),
			"error":         c.QueryParam("error"),
			"page":          "subscriptions",
		})
	}
}

// HomeHandler is the application landing page.
// Visitors should be able to subscribe themselves to a mailist using the subscribe form.
// The handler purposes is to show how simple it is to render dynamic html pages.
func HomeHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "subscribe.html", ViewContext{
		"page":    "subscribe",
		"success": c.QueryParam("success"),
		"error":   c.QueryParam("error"),
	})
}

// SubscribeHandler handles the subscribe form submission
// The handler purposes is to perform a very basic validation in the request inputs with the regexp package as well as
// introduce Echo's Redirect function.
func SubscribeHandler(repo SubscriptionRepo, e *echo.Echo) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := c.FormValue("email")
		fullName := c.FormValue("full-name")

		if email == "" || fullName == "" {
			return c.Render(http.StatusUnprocessableEntity, "subscribe.html", ViewContext{
				"page":     "subscribe",
				"email":    email,
				"fullName": fullName,
				"error":    "Invalid name or e-mail",
			})
		}

		emailRE := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
		if !emailRE.MatchString(email) {
			return c.Render(http.StatusUnprocessableEntity, "subscribe.html", ViewContext{
				"page":     "subscribe",
				"email":    email,
				"fullName": fullName,
				"error":    "Invalid e-mail",
			})
		}

		err := repo.Upsert(Subscription{Email: email, Name: fullName})
		if err != nil {
			return err
		}

		if hd := c.Request().Header["Authorization"]; len(hd) != 0 {
			return redirectWithFlashMessage(c, e, "root", "subscriptions", "You have been successfully subscribed")
		}

		return redirectWithFlashMessage(c, e, "root", "success", "You have been successfully subscribed")
	}

}

// Template holds the compiled views and serve as a `Renderer` to Echo.
type Template struct {
	templates *template.Template
}

// Render executes the template with a given context.
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplate(e *echo.Echo) echo.Renderer {
	files, _ := filepath.Glob("pages/*.html")
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

// ViewContext transits data beetwen the handler and the template.
type ViewContext map[string]interface{}

// Subscription represents a mailist subscription data.
type Subscription struct {
	EmailVerificationResponse `bson:"emailVerificationResponse"`
	Email                     string `bson:"email"`
	Name                      string `bson:"fullName"`
}

// EmailVerificationResponse represents the mail checker response.
type EmailVerificationResponse struct {
	Email      string  `json:"email"`
	Suggestion string  `json:"did_you_mean"`
	Valid      bool    `json:"format_valid"`
	Score      float64 `json:"score"`
}

// SubscriptionRepo abstracts the application persistance layer.
type SubscriptionRepo interface {
	FindAll(selector map[string]interface{}) ([]Subscription, error)
	Remove(selector map[string]interface{}) error
	Upsert(subscription Subscription) error
}

func newSubscriptionRepo() SubscriptionRepo {
	return MongoRepo{newMongoClient()}
}

// MongoRepo its a concrete implementation of SubscriptionRepo.
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

func (m MongoRepo) Upsert(subscription Subscription) error {
	coll, cs := m.client.GetSession()
	defer cs()
	_, err := coll.Upsert(bson.M{"email": subscription.Email}, subscription)
	return err
}

// MongoClient wraps the mgo package.
type MongoClient struct {
	session *mgo.Session
}

func newMongoClient() MongoClient {
	mongoURL := os.Getenv("MONGO_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://mongo_user:mongo_secret@localhost:27017"
	}
	mongo, err := mgo.Dial(mongoURL)
	if err != nil {
		log.Fatal(err)
	}
	return MongoClient{session: mongo}
}

func (m MongoClient) GetSession() (*mgo.Collection, func()) {
	s := m.session.Copy()
	return s.DB(mongoDatabaseName).C(mongoCollectionName), s.Close
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

func checkEmail(email string, c echo.Context) (EmailVerificationResponse, error) {
	fmt.Println("Validing email: ", email)

	url := fmt.Sprintf("%s%s", mailCheckerURL, email)
	var resp EmailVerificationResponse

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return resp, err
	}

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return resp, err
	}
	if res.StatusCode != 200 {
		return resp, errors.New("Request to email verifier failed")
	}

	b, _ := ioutil.ReadAll(res.Body)
	fmt.Println("Response: ", string(b), "for email: ", email)

	json.NewDecoder(bytes.NewReader(b)).Decode(&resp)

	return resp, nil
}

func setHost() {
	host = "http://localhost:4000"
	if h := os.Getenv("HOST"); h != "" {
		host = h
		if p := os.Getenv("PORT"); p != "" {
			host = host + p
		}
	}
}

func setMailCheckerURL() {
	if k := os.Getenv("MAIL_CHECKER_ACCESS_KEY"); k != "" {
		mailCheckerURL = fmt.Sprintf("http://apilayer.net/api/check?access_key=%s&smtp=1&format=&email=", k)
	} else {
		fmt.Println("WARN: THIS APPLICATION IS USING https://apilayer.com/ TO CHECK EMAIL CORRECTNESS. MAKE SURE YOU HAVE IT SET AS AN ENVIRONMENT VARIABLE: `MAIL_CHECKER_ACCESS_KEY`")
	}
}

func redirectWithFlashMessage(c echo.Context, e *echo.Echo, routeName, msgType, msg string) error {
	path := e.Reverse(routeName)
	return c.Redirect(http.StatusFound, fmt.Sprintf("%s%s?%s=%s", host, path, msgType, msg))
}
