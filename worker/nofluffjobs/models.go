package nofluffjobs

import (
	"fmt"
	"strings"
	"time"

	"log/slog"

	"github.com/kabinasoftware/jobs-agg/models"
	"github.com/kabinasoftware/jobs-agg/util"
)

const Source = "nofluffjobs.com"

type Offer struct {
	ID    string `json:"id"`
	Specs struct {
		DailyTasks []string `json:"dailyTasks"`
	} `json:"specs"`
	Title  string `json:"title"`
	Basics struct {
		Category   string   `json:"category"`
		Seniority  []string `json:"seniority"`
		Technology string   `json:"technology"`
	} `json:"basics"`
	Company struct {
		URL  string `json:"url"`
		Logo struct {
			Original    string `json:"original"`
			JobsDetails string `json:"jobs_details"`
		} `json:"logo"`
		Name string `json:"name"`
		Size string `json:"size"`
	} `json:"company"`
	Details struct {
		Position    string `json:"position"`
		Description string `json:"description"`
		CoverPhoto  struct {
			Original string `json:"original"`
		} `json:"coverPhoto"`
	} `json:"details"`
	Benefits struct {
		Benefits    []string `json:"benefits"`
		OfficePerks []string `json:"officePerks"`
	} `json:"benefits"`
	Consents struct {
		InfoClause              string `json:"infoClause"`
		PersonalDataRequestLink string `json:"personalDataRequestLink"`
	} `json:"consents"`
	Essentials struct {
		Contract struct {
			Start    string      `json:"start"`
			Duration interface{} `json:"duration"`
		} `json:"contract"`
		OriginalSalary  Salary `json:"originalSalary"`
		ConvertedSalary Salary `json:"convertedSalary"`
		Methodology     []struct {
			Name   string   `json:"name"`
			Values []string `json:"values"`
		} `json:"methodology"`
		Recruitment struct {
			Languages []struct {
				Code string `json:"code"`
			} `json:"languages"`
			OnlineInterviewAvailable bool `json:"onlineInterviewAvailable"`
		} `json:"recruitment"`
		Requirements struct {
			Musts       []Requirement `json:"musts"`
			Nices       []Requirement `json:"nices"`
			Description string        `json:"description"`
		} `json:"requirements"`
	} `json:"essentials"`
	Posted     int64  `json:"posted"`
	ExpiresAt  string `json:"expiresAt"`
	Status     string `json:"status"`
	PostingURL string `json:"postingUrl"`
	Metadata   struct {
		SectionLanguages struct {
			Description string `json:"description"`
		} `json:"sectionLanguages"`
	} `json:"metadata"`
	Regions   []string `json:"regions"`
	Reference string   `json:"reference"`
	Seo       struct {
		Description string `json:"description"`
	} `json:"seo"`
}

type Salary struct {
	Currency    string      `json:"currency"`
	Types       SalaryTypes `json:"types"`
	Bonus       Bonus       `json:"bonus"`
	DisclosedAt string      `json:"disclosedAt"`
}

type SalaryTypes struct {
	Permanent SalaryDetails `json:"permanent"`
	B2B       SalaryDetails `json:"b2b"`
}

type SalaryDetails struct {
	Period      string    `json:"period"`
	Range       []float32 `json:"range"`
	PaidHoliday bool      `json:"paidHoliday"`
}

type Bonus struct {
	Stock        PerformanceDependent `json:"stock"`
	Compensation PerformanceDependent `json:"compensation"`
	SigningBonus SigningBonus         `json:"signingBonus"`
}

type PerformanceDependent struct {
	Performance bool `json:"performance"`
	Dependent   bool `json:"dependent"`
}

type SigningBonus struct {
	Performance bool  `json:"performance"`
	Dependent   bool  `json:"dependent"`
	Range       []int `json:"range"`
}

type Requirement struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

func (o *Offers) Setup(client *Worker) []*models.Offer {
	offers := make([]*models.Offer, 0)
	requestCount := 0

	for _, posting := range o.Postings {
		requestCount++
		offer, err := client.getOffer(fmt.Sprintf("%s/posting/%s", APIGatewayURL, posting.ID))
		if err != nil {
			slog.Error("error getting offer",
				"error", err.Error(),
				"request_count", requestCount,
				"layer", "agg_worker")
			return nil
		}

		src := Source
		var logo *string
		if offer.Company.Logo.JobsDetails != "" {
			l := fmt.Sprintf("https://static.nofluffjobs.com/%s", offer.Company.Logo.Original)
			logo = &l
		}
		var banner *string
		if offer.Details.CoverPhoto.Original != "" {
			b := fmt.Sprintf("https://static.nofluffjobs.com/%s", offer.Details.CoverPhoto.Original)
			banner = &b
		}
		apply := fmt.Sprintf("https://nofluffjobs.com/pl/job/%s", offer.PostingURL)
		newOffer := &models.Offer{
			Title:             offer.Title,
			ParsedCompanyName: offer.Company.Name,
			Source:            &src,
			Apply:             &apply,
			Logo:              logo,
			Banner:            banner,
		}

		salary := offer.Essentials.OriginalSalary
		if salary.Currency != "" {
			if len(salary.Types.B2B.Range) == 2 {
				minSalary := int(salary.Types.B2B.Range[0])
				maxSalary := int(salary.Types.B2B.Range[1])
				newOffer.MinSalary = &minSalary
				newOffer.MaxSalary = &maxSalary
			}

			if newOffer.MinSalary == nil && len(salary.Types.Permanent.Range) == 2 {
				minSalary := int(salary.Types.Permanent.Range[0])
				maxSalary := int(salary.Types.Permanent.Range[1])
				newOffer.MinSalary = &minSalary
				newOffer.MaxSalary = &maxSalary
			}

			if salary.Currency != "PLN" {
				cur, rateErr := util.GetExchangeRate(salary.Currency, "PLN")
				if rateErr != nil {
					slog.Error("failed to get exchange rate", "error", rateErr.Error(), "layer", "agg_worker")
					continue
				}

				if newOffer.MinSalary != nil {
					minSalaryPLN := int(float64(*newOffer.MinSalary) * cur)
					minSalaryPLN = (minSalaryPLN / 100) * 100
					newOffer.MinSalary = &minSalaryPLN
				}

				if newOffer.MaxSalary != nil {
					maxSalaryPLN := int(float64(*newOffer.MaxSalary) * cur)
					maxSalaryPLN = (maxSalaryPLN / 100) * 100
					newOffer.MaxSalary = &maxSalaryPLN
				}

			}

			if salary.Types.B2B.Period == "Hour" || salary.Types.Permanent.Period == "Hour" {
				hourly := true
				newOffer.Hourly = &hourly
			}

			pln := "PLN"
			newOffer.Currency = &pln
		}

		newOffer.Description += offer.Details.Description
		newOffer.Description += "\n\n"

		newOffer.Description += "Daily tasks: \n"
		for _, task := range offer.Specs.DailyTasks {
			newOffer.Description += task + "\n"
		}

		newOffer.Description = strings.Replace(newOffer.Description, "<h1>", "<p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "</h1>", "</p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "<h2>", "<p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "</h2>", "</p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "<h3>", "<p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "</h3>", "</p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "<h4>", "<p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "</h4>", "</p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "<h5>", "<p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "</h5>", "</p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "<h6>", "<p>", -1)
		newOffer.Description = strings.Replace(newOffer.Description, "</h6>", "</p>", -1)

		parsedTime, err := time.Parse("2006-01-02T15:04:05", offer.ExpiresAt)
		if err != nil {
			slog.Error("Error parsing time", "error", err.Error(), "layer", "agg_worker")
			continue
		}
		newOffer.ExpiresAt = &parsedTime

		tm := time.Unix(offer.Posted/1000, 0)
		newOffer.CreatedAt = &tm

		for _, pl := range offer.Basics.Seniority {
			if pl == "Senior" {
				exp := 3.0
				newOffer.Experience = &exp
			}
			if pl == "Mid" {
				exp := 2.0
				newOffer.Experience = &exp
			}
			if pl == "Junior" {
				exp := 1.0
				newOffer.Experience = &exp
			}
		}

		if strings.ContainsAny(offer.Seo.Description, "B2B") {
			newOffer.Contracts = append(newOffer.Contracts, models.ContractTypeIDB2B)
		}

		if strings.ContainsAny(offer.Seo.Description, "UoP") {
			newOffer.Contracts = append(newOffer.Contracts, models.ContractTypeIDUmowaOPrace)
		}

		offers = append(offers, newOffer)
	}

	slog.Info("completed processing offers", "total_requests", requestCount)
	return offers
}

type Offers struct {
	Postings []struct {
		ID string `json:"id"`
	} `json:"postings"`
	TotalCount int `json:"totalCount"`
	TotalPages int `json:"totalPages"`
}

type OffersCount struct {
	TotalCount int `json:"totalCount"`
	TotalPages int `json:"totalPages"`
}
