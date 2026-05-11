package domain

// TopReferrer resume el desempeño de un paciente como referidor.
type TopReferrer struct {
	DocumentNumber string `json:"document_number"`
	Name           string `json:"name"`
	ReferralCount  int    `json:"referral_count"`
}
