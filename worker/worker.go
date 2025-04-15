package worker

import "github.com/kabinasoftware/jobs-agg/models"

type Worker interface {
	GetOffers(page int) ([]*models.Offer, error)
	GetPagesCount() (totalPages int, error error)
}
