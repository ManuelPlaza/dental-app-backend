package domain

import "errors"

var (
	ErrDocumentRequired     = errors.New("document_number is required")
	ErrPatientAlreadyExists = errors.New("patient already exists")
)
