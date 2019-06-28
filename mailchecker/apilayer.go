package apilayer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/klebervirgilio/go-echo-basics/config"
	"github.com/klebervirgilio/go-echo-basics/core"
)

func New(c *config.Config) core.MailChecker {
	endpoint := fmt.Sprintf(c.MustGetString("mailChecker.url"), c.MustGetString("mailChecker.accessKey"))
	return APILayer{endpoint}
}

type APILayer struct {
	endpoint string
}

func (a APILayer) Validate(email string) (core.EmailVerificationResponse, error) {
	url := fmt.Sprintf("%s%s", a.endpoint, email)
	var resp core.EmailVerificationResponse

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
	json.NewDecoder(bytes.NewReader(b)).Decode(&resp)

	return resp, nil
}
