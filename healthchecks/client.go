package healthchecks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/IdeaEvolver/cutter-pkg/client"
)

type ServiceResponse struct {
	Status string `json:"status"`
}

type ExternalConfig struct {
	HibbertEndpoint string
	AppId           string
	HibbertUsername string
	HibbertPassword string
	StripeEndpoint  string
	StripeKey       string
	ClientId        string
	ClientSecret    string
	AZCRMUrl        string
	XAppId          string
}

type Client struct {
	Client *client.Client

	Platform    string
	Fulfillment string
	Crm         string
	Study       string

	ExternalConfig ExternalConfig
}

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

func (c *Client) do(ctx context.Context, req *client.Request, ret interface{}) error {
	res, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if ret != nil {
		return json.NewDecoder(res.Body).Decode(&ret)
	}

	return nil
}

func (c *Client) doExternal(ctx context.Context, req *http.Request) string {
	internalClient := &http.Client{}

	resp, _ := internalClient.Do(req)

	return strings.TrimSpace(resp.Status)
}

func (c *Client) PlatformStatus(ctx context.Context) (*ServiceResponse, error) {
	url := fmt.Sprintf("%s/healthcheck", c.Platform)
	req, _ := client.NewRequestWithContext(ctx, "GET", url, nil)

	status := &ServiceResponse{}
	if err := c.do(ctx, req, &status); err != nil {
		return nil, err
	}

	return status, nil
}

func (c *Client) PlatformUIStatus(ctx context.Context) (*ServiceResponse, error) {
	url := "https://dev.cutter.live/sign-in"
	req, _ := client.NewRequestWithContext(ctx, "GET", url, nil)

	status := &ServiceResponse{}
	if err := c.do(ctx, req, &status); err != nil {
		return nil, err
	}

	return status, nil
}

func (c *Client) FulfillmentStatus(ctx context.Context) (*ServiceResponse, error) {
	url := fmt.Sprintf("%s/healthcheck", c.Fulfillment)
	req, _ := client.NewRequestWithContext(ctx, "GET", url, nil)

	status := &ServiceResponse{}
	if err := c.do(ctx, req, &status); err != nil {
		return nil, err
	}

	return status, nil
}

func (c *Client) CrmStatus(ctx context.Context) (*ServiceResponse, error) {
	url := fmt.Sprintf("%s/healthcheck", c.Crm)
	req, _ := client.NewRequestWithContext(ctx, "GET", url, nil)

	status := &ServiceResponse{}
	if err := c.do(ctx, req, &status); err != nil {
		return nil, err
	}

	return status, nil
}

func (c *Client) StudyStatus(ctx context.Context) (*ServiceResponse, error) {
	url := fmt.Sprintf("%s/healthcheck", c.Study)
	req, _ := client.NewRequestWithContext(ctx, "GET", url, nil)

	status := &ServiceResponse{}
	if err := c.do(ctx, req, &status); err != nil {
		return nil, err
	}

	return status, nil
}

func (c *Client) StudyUIStatus(ctx context.Context) (*ServiceResponse, error) {
	url := "https://study.dev.cutter.live/"
	req, _ := client.NewRequestWithContext(ctx, "GET", url, nil)

	status := &ServiceResponse{}
	if err := c.do(ctx, req, &status); err != nil {
		return nil, err
	}

	return status, nil
}

type hibbertResponse struct {
	Token string `json:"token"`
}

func (c *Client) HibbertStatus(ctx context.Context) (*ServiceResponse, error) {
	url := c.ExternalConfig.HibbertEndpoint
	body := struct {
		AppId    string `json:"appId"`
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		AppId:    c.ExternalConfig.AppId,
		Username: c.ExternalConfig.HibbertUsername,
		Password: c.ExternalConfig.HibbertPassword,
	}

	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	httpStatus := c.doExternal(ctx, req)

	status := &ServiceResponse{Status: httpStatus}

	return status, nil
}

func (c *Client) StripeStatus(ctx context.Context) (*ServiceResponse, error) {
	url := c.ExternalConfig.StripeEndpoint

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.ExternalConfig.StripeKey)

	httpStatus := c.doExternal(ctx, req)

	status := &ServiceResponse{Status: httpStatus}

	return status, nil
}

func (c *Client) AZCRMStatus(ctx context.Context) (*ServiceResponse, error) {
	q := url.Values{}
	q.Add("grant_type", "client_credentials")
	q.Add("scope", "openid")
	q.Add("client_id", c.ExternalConfig.ClientId)
	q.Add("client_secret", c.ExternalConfig.ClientSecret)

	url := "https://identityapiqa.a.astrazeneca.com" + "/csdcidentity/oauth/token" + "?" + q.Encode()

	req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("X-App-Id", c.ExternalConfig.XAppId)

	httpStatus := c.doExternal(ctx, req)
	status := &ServiceResponse{Status: httpStatus}

	return status, nil
}
