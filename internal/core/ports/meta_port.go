package ports

import "context"

// MetaCAPIClient envía eventos a la Meta Conversions API.
type MetaCAPIClient interface {
	SendLeadEvent(ctx context.Context, event MetaLeadEvent) error
}

// MetaLeadEvent contiene los datos del evento Lead para Meta CAPI.
// Los campos Phone, Email y ExternalID se hashean con SHA-256 antes de enviar.
type MetaLeadEvent struct {
	EventTime      int64  // Unix timestamp
	EventSourceURL string // URL de la landing page
	Fbclid         string // Click ID de Facebook (_ fbc user_data field)
	Phone          string // Teléfono en texto plano — se hashea internamente
	Email          string // Email en texto plano — se hashea internamente
	ExternalID     string // document_number — se hashea internamente
}
