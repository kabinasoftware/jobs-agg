package pracuj

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/kabinasoftware/jobs-agg/models"
	"github.com/kabinasoftware/jobs-agg/worker"
)

var (
	APIGatewayURL = "https://massachusetts.pracuj.pl"
	offerRegexp   = regexp.MustCompile(`(?s)<script[^>]*>({\s*"props":\s*\{.*?\})\s*</script>`)
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

// TODO:
func (w *Worker) GetPagesCount() (int, error) {
	return 1000, nil
}

func (w *Worker) GetOffers(page int) (offer []*models.Offer, x error) {
	baseURL, err := url.Parse(w.baseURL + "/JobOffers/listing/grouped")
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add("wm", "home-office")
	params.Add("pn", fmt.Sprintf("%d", page))
	baseURL.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", baseURL.String(), nil)
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

	var offers *Offers
	err = json.Unmarshal(body, &offers)
	if err != nil {
		return nil, err
	}

	return offers.Setup(w), nil
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

	match := offerRegexp.FindSubmatch(body)
	if len(match) < 2 {
		return nil, fmt.Errorf("could not find JSON data in the response")
	}

	var offer *Offer
	if err = json.Unmarshal(match[1], &offer); err != nil {
		return nil, err
	}

	return offer, nil
}
