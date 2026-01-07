package services

import (
	"fmt"
	"time"

	"dental-app/internal/core/domain"

	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/pdf"
	"github.com/johnfercher/maroto/pkg/props"
)

func GenerateHistoryPDF(history domain.MedicalHistory) ([]byte, error) {
	m := pdf.NewMaroto(consts.Portrait, consts.A4)

	// =========
	// Fallbacks seguros (por si no vienen preloads)
	// =========
	patientName := "N/A"
	patientDoc := "N/A"
	if history.Appointment.Patient.ID != 0 {
		patientName = fmt.Sprintf("%s %s", history.Appointment.Patient.FirstName, history.Appointment.Patient.LastName)
		patientDoc = history.Appointment.Patient.DocumentNumber
	}

	doctorName := "N/A"
	doctorLicense := "N/A"
	doctorSpecialty := "N/A"
	if history.Appointment.Specialist.ID != 0 {
		doctorName = fmt.Sprintf("%s %s", history.Appointment.Specialist.FirstName, history.Appointment.Specialist.LastName)
		doctorLicense = history.Appointment.Specialist.LicenseNumber
		doctorSpecialty = history.Appointment.Specialist.Specialty
	}

	consultDate := "N/A"
	if !history.Appointment.StartTime.IsZero() {
		consultDate = history.Appointment.StartTime.Format("2006-01-02 15:04")
	}

	// =========
	// 1) HEADER (Logo + Título)
	// =========
	m.RegisterHeader(func() {
		m.Row(24, func() {
			m.Col(3, func() {
				// Logo (ajusta la ruta si lo ubicas en otro lado)
				// Si tu versión de Maroto NO soporta FileImage, dímelo y lo ajusto al método correcto.
				m.FileImage("assets/logo.png", props.Rect{
					Top:     2,
					Left:    2,
					Percent: 120,
				})
			})
			m.Col(9, func() {
				m.Text("TECNICA DENTAL JC - ALFONSO LOPEZ", props.Text{
					Top:   4,
					Style: consts.Bold,
					Size:  16,
				})
				m.Text("Historia Clínica", props.Text{
					Top:   12,
					Style: consts.Italic,
					Size:  11,
				})
			})
		})

		m.Row(6, func() {
			m.Col(12, func() {
				m.Text(fmt.Sprintf("Cita #%d", history.AppointmentID), props.Text{
					Align: consts.Right,
					Size:  9,
					Top:   1,
				})
			})
		})

		m.Line(0.8)
	})

	// =========
	// 2) INFORMACIÓN DEL PACIENTE
	// =========
	m.Row(10, func() {
		m.Col(12, func() {
			m.Text("INFORMACIÓN DEL PACIENTE", props.Text{
				Style: consts.Bold,
				Top:   2,
			})
		})
	})

	m.Row(22, func() {
		m.Col(6, func() {
			m.Text(fmt.Sprintf("Nombre: %s", patientName), props.Text{Size: 10})
			m.Text(fmt.Sprintf("Documento: %s", patientDoc), props.Text{Top: 6, Size: 10})
		})
		m.Col(6, func() {
			m.Text(fmt.Sprintf("Fecha consulta: %s", consultDate), props.Text{Size: 10})
			m.Text(fmt.Sprintf("Paciente ID: #%d", history.PatientID), props.Text{Top: 6, Size: 10})
		})
	})

	m.Line(0.6)

	// =========
	// 3) PROFESIONAL TRATANTE
	// =========
	m.Row(10, func() {
		m.Col(12, func() {
			m.Text("PROFESIONAL TRATANTE", props.Text{
				Style: consts.Bold,
				Top:   3,
			})
		})
	})

	m.Row(22, func() {
		m.Col(12, func() {
			m.Text(fmt.Sprintf("Dr(a). %s", doctorName), props.Text{Size: 10})
			m.Text(fmt.Sprintf("Licencia: %s", doctorLicense), props.Text{Top: 6, Size: 10})
			m.Text(fmt.Sprintf("Especialidad: %s", doctorSpecialty), props.Text{Top: 12, Size: 10})
		})
	})

	m.Line(0.6)

	// =========
	// 4) DETALLE CLÍNICO
	// =========
	m.Row(10, func() {
		m.Col(12, func() {
			m.Text("DETALLE DE LA ATENCIÓN", props.Text{
				Style: consts.Bold,
				Top:   3,
			})
		})
	})

	// Diagnóstico (aumentamos altura para evitar cortes)
	m.Row(22, func() {
		m.Col(12, func() {
			m.Text("Diagnóstico:", props.Text{Style: consts.Bold, Size: 10})
			m.Text(history.Diagnosis, props.Text{Top: 6, Style: consts.Italic, Size: 10})
		})
	})

	// Tratamiento (puede ser largo -> subimos altura)
	m.Row(44, func() {
		m.Col(12, func() {
			m.Text("Tratamiento Realizado:", props.Text{Style: consts.Bold, Size: 10})
			m.Text(history.Treatment, props.Text{Top: 6, Size: 10})
		})
	})

	// Notas (puede ser largo -> subimos altura)
	m.Row(34, func() {
		m.Col(12, func() {
			m.Text("Notas / Observaciones:", props.Text{Style: consts.Bold, Size: 10})
			m.Text(history.DoctorNotes, props.Text{Top: 6, Size: 10})
		})
	})

	// Adjuntos (si aplica)
	if history.Attachments != "" {
		m.Row(20, func() {
			m.Col(12, func() {
				m.Text("Adjuntos:", props.Text{Style: consts.Bold, Size: 10})
				m.Text(history.Attachments, props.Text{Top: 6, Size: 10})
			})
		})
	}

	// =========
	// 5) FOOTER
	// =========
	m.RegisterFooter(func() {
		m.Line(0.3)
		m.Row(10, func() {
			m.Col(12, func() {
				m.Text(
					fmt.Sprintf("Generado el: %s", time.Now().Format("2006-01-02 15:04:05")),
					props.Text{Style: consts.Italic, Size: 8, Align: consts.Right},
				)
			})
		})
	})

	// Generar bytes
	buffer, err := m.Output()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
