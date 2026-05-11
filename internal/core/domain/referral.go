package domain

// TopReferrer resume el desempeño de un paciente como referidor.
type TopReferrer struct {
	DocumentNumber string `json:"document_number"`
	Name           string `json:"name"`
	ReferralCount  int    `json:"referral_count"`
}

// PatientReferralStatus es el contexto del paciente en el grafo de referidos.
// Se inyecta al chatbot cuando se detecta una cédula en la conversación.
type PatientReferralStatus struct {
	ReferralCount  int    `json:"referral_count"`   // cuántas personas ha referido
	ReferredByName string `json:"referred_by_name"` // quién lo refirió (vacío si vino solo)
}
