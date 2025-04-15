package pracuj

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/kabinasoftware/jobs-agg/models"
	"github.com/kabinasoftware/jobs-agg/util"
)

const Source = "pracuj.pl"

type (
	Offer struct {
		Props struct {
			PageProps struct {
				OfferId         string `json:"offerId"`
				DehydratedState struct {
					Queries []struct {
						State struct {
							Data struct {
								TextSections []struct {
									SectionType  string   `json:"sectionType"`
									PlainText    string   `json:"plainText"`
									TextElements []string `json:"textElements"`
								} `json:"textSections"`
							} `json:"data"`
						} `json:"state"`
					} `json:"queries"`
				} `json:"dehydratedState"`
			} `json:"pageProps"`
		} `json:"props"`
	}
)

func (offer *Offer) CreateSingleDescription() string {
	var description string
	for _, query := range offer.Props.PageProps.DehydratedState.Queries {
		for _, section := range query.State.Data.TextSections {
			description += section.PlainText + "\n\n"
		}
	}
	return description
}

type (
	Offers struct {
		GroupedOffers []GroupedOffer `json:"groupedOffers"`
	}
	GroupedOffer struct {
		GroupID                   string    `json:"groupId"`
		JobTitle                  string    `json:"jobTitle"`
		CompanyName               string    `json:"companyName"`
		CompanyProfileAbsoluteURI string    `json:"companyProfileAbsoluteUri"`
		CompanyID                 int       `json:"companyId"`
		CompanyLogoURI            string    `json:"companyLogoUri"`
		LastPublicated            time.Time `json:"lastPublicated"`
		ExpirationDate            time.Time `json:"expirationDate"`
		SalaryDisplayText         string    `json:"salaryDisplayText"`
		JobDescription            string    `json:"jobDescription"`
		IsSuperOffer              bool      `json:"isSuperOffer"`
		IsFranchise               bool      `json:"isFranchise"`
		IsOptionalCv              bool      `json:"isOptionalCv"`
		IsOneClickApply           bool      `json:"isOneClickApply"`
		IsJobiconCompany          bool      `json:"isJobiconCompany"`
		Offers                    []struct {
			PartitionID      int64         `json:"partitionId"`
			OfferAbsoluteURI string        `json:"offerAbsoluteUri"`
			DisplayWorkplace string        `json:"displayWorkplace"`
			IsWholePoland    bool          `json:"isWholePoland"`
			AppliedProducts  []interface{} `json:"appliedProducts"`
		} `json:"offers"`
		PositionLevels             []string      `json:"positionLevels"`
		TypesOfContract            []string      `json:"typesOfContract"`
		WorkSchedules              []string      `json:"workSchedules"`
		WorkModes                  []string      `json:"workModes"`
		PrimaryAttributes          []interface{} `json:"primaryAttributes"`
		CommonOfferID              string        `json:"commonOfferId"`
		SearchEngineRelevancyScore float32       `json:"searchEngineRelevancyScore"`
		MobileBannerURI            string        `json:"mobileBannerUri"`
		DesktopBannerURI           string        `json:"desktopBannerUri"`
		AppliedProducts            []interface{} `json:"appliedProducts"`
	}
)

func (o *Offers) Setup(client *Worker) []*models.Offer {
	pracaOffers := make([]*models.Offer, 0)

	for _, groupedOffer := range o.GroupedOffers {
		if len(groupedOffer.Offers) > 0 {
			offer := groupedOffer.Offers[0]
			off, err := client.getOffer(offer.OfferAbsoluteURI)
			if err != nil {
				slog.Error("failed to get offer", "error", err.Error(), "layer", "agg_worker")
				continue
			}
			desc := off.CreateSingleDescription()

			src := Source
			newOffer := &models.Offer{
				Title:             groupedOffer.JobTitle,
				ParsedCompanyName: groupedOffer.CompanyName,
				Description:       desc,
				CreatedAt:         &groupedOffer.LastPublicated,
				ExpiresAt:         &groupedOffer.ExpirationDate,
				Source:            &src,
				Apply:             &offer.OfferAbsoluteURI,
				Logo:              &groupedOffer.CompanyLogoURI,
				Banner:            &groupedOffer.DesktopBannerURI,
			}

			salaryRangeParts := strings.Split(groupedOffer.SalaryDisplayText, "–")
			if len(salaryRangeParts) == 2 {
				minSalaryParts := strings.Fields(strings.TrimSpace(salaryRangeParts[0]))
				maxSalaryParts := strings.Fields(strings.TrimSpace(salaryRangeParts[1]))

				if len(minSalaryParts) >= 2 && len(maxSalaryParts) >= 2 {
					minSalary, err := strconv.Atoi(minSalaryParts[0] + minSalaryParts[1])
					if err != nil {
						slog.Error("failed to convert min salary", "error", err.Error(), "layer", "agg_worker")
						continue
					}
					newOffer.MinSalary = &minSalary
					maxSalary, err := strconv.Atoi(maxSalaryParts[0] + maxSalaryParts[1])
					if err != nil {
						slog.Error("failed to convert max salary", "error", err.Error(), "layer", "agg_worker")
						continue
					}
					newOffer.MaxSalary = &maxSalary
				}

				currency := maxSalaryParts[2]
				newOffer.Currency = &currency

				if currency != "" {
					var cur string
					if currency == "zł" {
						cur = "PLN"
					} else if currency == "€" {
						cur = "EUR"
					} else if currency == "£" {
						cur = "GBP"
					} else if currency == "$" {
						cur = "USD"
					} else {
						continue
					}

					if cur != "" {
						currency, err := util.GetExchangeRate(cur, "PLN")
						if err != nil {
							slog.Error("failed to get exchange rate", "error", err.Error(), "layer", "agg_worker")
							continue
						}

						if newOffer.MinSalary != nil {
							minSalaryPLN := int(float64(*newOffer.MinSalary) * currency)
							minSalaryPLN = (minSalaryPLN / 100) * 100
							newOffer.MinSalary = &minSalaryPLN
						}

						if newOffer.MaxSalary != nil {
							maxSalaryPLN := int(float64(*newOffer.MaxSalary) * currency)
							maxSalaryPLN = (maxSalaryPLN / 100) * 100
							newOffer.MaxSalary = &maxSalaryPLN
						}

						pln := "PLN"
						newOffer.Currency = &pln
					}
				}
			}

			findedPositionLevels := make([]float64, 0)
			for _, pl := range groupedOffer.PositionLevels {
				if pl == "Starszy specjalista (Senior)" {
					findedPositionLevels = append(findedPositionLevels, 3)
				}
				if pl == "Specjalista (Mid / Regular)" {
					findedPositionLevels = append(findedPositionLevels, 2)
				}
				if pl == "Młodszy specjalista (Junior)" {
					findedPositionLevels = append(findedPositionLevels, 1)
				}
			}

			if len(findedPositionLevels) > 0 {
				lowestLevel := findedPositionLevels[0]
				for _, level := range findedPositionLevels {
					if level < lowestLevel {
						lowestLevel = level
					}
				}
				newOffer.Experience = &lowestLevel
			}

			for _, toc := range groupedOffer.TypesOfContract {
				if toc == "Kontrakt B2B" {
					newOffer.Contracts = append(newOffer.Contracts, models.ContractTypeIDB2B)
				}
				if toc == "Umowa zlecenie" {
					newOffer.Contracts = append(newOffer.Contracts, models.ContractTypeIDUmowaZlecenie)
				}
				if toc == "Umowa o pracę" {
					newOffer.Contracts = append(newOffer.Contracts, models.ContractTypeIDUmowaOPrace)
				}
			}

			for _, ws := range groupedOffer.WorkSchedules {
				if ws == "Część etatu" {
					newOffer.Type = models.OfferTypePartTime
				}
				if ws == "Dodatkowa / tymczasowa" {
					newOffer.Type = models.OfferTypePartTime
				}
			}

			pracaOffers = append(pracaOffers, newOffer)
		}
	}

	return pracaOffers
}
