package http

import (
	"errors"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/klebervirgilio/go-echo-basics/config"
	"github.com/klebervirgilio/go-echo-basics/core"

	"github.com/labstack/echo"
)

// ViewContext transits data beetwen the handler and the template.
type ViewContext map[string]interface{}

// checkEmailHandler performs a correctness check on a given email provided via URL parameter or
// runs the check in parallel for all subscriptons email found in the database.
// The handler purposes is to exercise the ability of conditionally use a handler and
// how Go make it easy to achieve concurrency.
func checkEmailHandler(repo core.Repository, e *echo.Echo, mailChecker core.MailChecker, config *config.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		if email := c.Param("email"); email != "" {
			resp, err := mailChecker.Validate(email)
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
			go func(sub core.Subscription) {
				e.Logger.Info("Checking email ", sub.Email)
				resp, err := mailChecker.Validate(sub.Email)
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
func FullListHandler(repo core.Repository) echo.HandlerFunc {
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
func SubscribeHandler(repo core.Repository, e *echo.Echo) echo.HandlerFunc {
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

		err := repo.Upsert(core.Subscription{Email: email, Name: fullName})
		if err != nil {
			return err
		}

		if hd := c.Request().Header["Authorization"]; len(hd) != 0 {
			return redirectWithFlashMessage(c, e, "root", "subscriptions", "You have been successfully subscribed")
		}

		return redirectWithFlashMessage(c, e, "root", "success", "You have been successfully subscribed")
	}

}
