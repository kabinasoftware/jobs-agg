package nofluffjobs

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/kabinasoftware/jobs-agg/models"
	"github.com/kabinasoftware/jobs-agg/worker"
)

var (
	APIGatewayURL = "https://nofluffjobs.com/api"
)

type Options struct {
	BaseURL    string
	HTTPClient *http.Client
}

type Worker struct {
	baseURL    string
	HTTPClient *http.Client
}

func Init(opts *Options) worker.Worker {
	if opts == nil {
		opts = &Options{
			BaseURL:    APIGatewayURL,
			HTTPClient: http.DefaultClient,
		}
	}

	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}

	if opts.BaseURL == "" {
		opts.BaseURL = APIGatewayURL
	}

	return &Worker{
		baseURL:    opts.BaseURL,
		HTTPClient: opts.HTTPClient,
	}
}

func (w *Worker) GetPagesCount() (int, error) {
	baseURL, err := url.Parse(w.baseURL + "/search/posting")
	if err != nil {
		return 0, err
	}

	params := url.Values{}
	params.Add("pageFrom", "1")
	params.Add("region", "pl")
	params.Add("language", "pl-PL")
	params.Add("salaryCurrency", "PLN")
	params.Add("salaryPeriod", "month")
	baseURL.RawQuery = params.Encode()

	reqBody := bytes.NewBuffer([]byte(`{"rawSearch": "remote"}`))
	resp, err := w.HTTPClient.Post(baseURL.String(), "application/json", reqBody)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result *OffersCount
	err = json.Unmarshal(body, &result)
	if err != nil {
		return 0, err
	}

	return result.TotalPages, nil
}

func (w *Worker) GetOffers(page int) ([]*models.Offer, error) {
	baseURL, err := url.Parse(w.baseURL + "/search/posting")
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add("pageFrom", strconv.Itoa(page))
	params.Add("region", "pl")
	params.Add("language", "pl-PL")
	params.Add("salaryCurrency", "PLN")
	params.Add("salaryPeriod", "month")
	baseURL.RawQuery = params.Encode()

	reqBody := bytes.NewBuffer([]byte(`{"rawSearch": "remote"}`))
	resp, err := w.HTTPClient.Post(baseURL.String(), "application/json", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var explorer *Offers
	err = json.Unmarshal(body, &explorer)
	if err != nil {
		return nil, err
	}

	return explorer.Setup(w), nil
}

func (w *Worker) getOffer(uri string) (*Offer, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	resp, err := w.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var offer *Offer
	if err = json.Unmarshal(body, &offer); err != nil {
		return nil, err
	}

	return offer, nil
}
