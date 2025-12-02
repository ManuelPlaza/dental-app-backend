package services

import (
	"fmt"
	"time"

	"dental-app/internal/core/domain"

	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/pdf"
	"github.com/johnfercher/maroto/pkg/props"
)

// Esta función recibe la Historia y devuelve el archivo PDF en bytes (memoria)
func GenerateHistoryPDF(history domain.MedicalHistory) ([]byte, error) {
	m := pdf.NewMaroto(consts.Portrait, consts.A4)

	// 1. ENCABEZADO DE LA CLÍNICA
	m.RegisterHeader(func() {
		m.Row(20, func() {
			m.Col(12, func() {
				m.Text("DENTAL APP - HISTORIA CLÍNICA", props.Text{
					Top:   5,
					Style: consts.Bold,
					Align: consts.Center,
					Size:  14,
				})
			})
		})
	})

	// 2. INFORMACIÓN DEL PACIENTE
	m.Row(10, func() {
		m.Col(12, func() {
			m.Text("INFORMACIÓN DEL PACIENTE", props.Text{Style: consts.Bold})
		})
	})
	m.Row(20, func() {
		m.Col(6, func() {
			m.Text(fmt.Sprintf("Nombre: %s %s", history.Appointment.Patient.FirstName, history.Appointment.Patient.LastName))
			m.Text(fmt.Sprintf("Documento: %s", history.Appointment.Patient.DocumentNumber), props.Text{Top: 6})
		})
		m.Col(6, func() {
			m.Text(fmt.Sprintf("Fecha Consulta: %s", history.Appointment.StartTime.Format("2006-01-02 15:04")))
			m.Text(fmt.Sprintf("ID Cita: #%d", history.AppointmentID), props.Text{Top: 6})
		})
	})

	m.Line(1.0)

	// 3. INFORMACIÓN DEL DOCTOR
	m.Row(10, func() {
		m.Col(12, func() {
			m.Text("PROFESIONAL TRATANTE", props.Text{Style: consts.Bold, Top: 5})
		})
	})
	m.Row(20, func() {
		m.Col(12, func() {
			m.Text(fmt.Sprintf("Dr(a). %s %s", history.Appointment.Specialist.FirstName, history.Appointment.Specialist.LastName))
			m.Text(fmt.Sprintf("Licencia: %s", history.Appointment.Specialist.LicenseNumber), props.Text{Top: 6})
			m.Text(fmt.Sprintf("Especialidad: %s", history.Appointment.Specialist.Specialty), props.Text{Top: 12})
		})
	})

	m.Line(1.0)

	// 4. DETALLES CLÍNICOS (Diagnóstico y Tratamiento)
	m.Row(10, func() {
		m.Col(12, func() {
			m.Text("DETALLE DE LA ATENCIÓN", props.Text{Style: consts.Bold, Top: 5})
		})
	})

	// Diagnóstico
	m.Row(15, func() {
		m.Col(12, func() {
			m.Text("Diagnóstico:", props.Text{Style: consts.Bold})
			m.Text(history.Diagnosis, props.Text{Top: 5, Style: consts.Italic})
		})
	})

	// Tratamiento (Puede ser largo, usamos Row más grande)
	m.Row(30, func() {
		m.Col(12, func() {
			m.Text("Tratamiento Realizado:", props.Text{Style: consts.Bold})
			m.Text(history.Treatment, props.Text{Top: 5})
		})
	})

	// Notas
	m.Row(20, func() {
		m.Col(12, func() {
			m.Text("Notas / Observaciones:", props.Text{Style: consts.Bold})
			m.Text(history.DoctorNotes, props.Text{Top: 5})
		})
	})

	// 5. PIE DE PÁGINA
	m.RegisterFooter(func() {
		m.Row(10, func() {
			m.Col(12, func() {
				m.Text(fmt.Sprintf("Generado el: %s", time.Now().Format("2006-01-02 15:04:05")), props.Text{
					Style: consts.Italic,
					Size:  8,
					Align: consts.Right,
				})
			})
		})
	})

	// Generar los bytes
	buffer, err := m.Output()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
