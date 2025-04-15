package models

import "time"

type OfferType string

const (
	OfferTypeFullTime OfferType = "FT"
	OfferTypePartTime OfferType = "PT"
)

type ContractTypeID int

const (
	ContractTypeIDUmowaZlecenie ContractTypeID = iota
	ContractTypeIDUmowaOPrace
	ContractTypeIDB2B
	ContractTypeIDUmowaODziele
)

type Offer struct {
	ID                 string           `db:"id"`
	ParsedCompanyName  string           `db:"parsed_cn"`
	EmployerUserID     *string          `db:"employer_user_id"`
	Closed             bool             `db:"closed"`
	Found              bool             `db:"found"`
	Title              string           `db:"title"`
	Type               OfferType        `db:"type"`
	Experience         *float64         `db:"experience"`
	Description        string           `db:"description"`
	MinSalary          *int             `db:"min_salary"`
	MaxSalary          *int             `db:"max_salary"`
	Hourly             *bool            `db:"hourly"`
	Apply              *string          `db:"apply"`
	Logo               *string          `db:"logo"`
	Banner             *string          `db:"banner"`
	PinnedTo           *time.Time       `db:"pinned_to"`
	Color              *string          `db:"color"`
	Currency           *string          `db:"currency"`
	FastApply          bool             `db:"fast_apply"`
	CoverLetterAllowed bool             `db:"cover_letter_allowed"`
	CreatedAt          *time.Time       `db:"created_at"`
	UpdatedAt          time.Time        `db:"updated_at"`
	Source             *string          `db:"source"`
	ExpiresAt          *time.Time       `db:"expires_at"`
	Contracts          []ContractTypeID `db:"contracts"`
}
