package domain

import "time"

type MedicalHistory struct {
	ID        uint `json:"id"`
	PatientID uint `json:"patient_id"`

	AppointmentID uint        `json:"appointment_id"`
	Appointment   Appointment `json:"appointment" gorm:"foreignKey:AppointmentID"` // <--- VITAL PARA EL PDF

	Diagnosis   string    `json:"diagnosis"`
	Treatment   string    `json:"treatment"`
	DoctorNotes string    `json:"doctor_notes"`
	Attachments string    `json:"attachments"`
	CreatedAt   time.Time `json:"created_at"`
}

func (MedicalHistory) TableName() string {
	return "\"MedicalHistory\""
}
