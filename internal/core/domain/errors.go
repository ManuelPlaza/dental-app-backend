package domain

import "errors"

var (
	ErrDocumentRequired        = errors.New("document_number is required")
	ErrPatientAlreadyExists    = errors.New("patient already exists")
	ErrInvalidPayload          = errors.New("invalid payload")
	ErrSpecialistNotFound      = errors.New("specialist not found")
	ErrSpecialistInactive      = errors.New("specialist is inactive")
	ErrLicenseRequired         = errors.New("license_number is required")
	ErrSpecialistAlreadyExists = errors.New("specialist already exists")
)
