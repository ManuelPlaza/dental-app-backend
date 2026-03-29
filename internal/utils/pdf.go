package utils

import (
	"bytes"
	"dental-app/internal/core/domain"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

const (
	pdfPageW    = 210.0
	pdfPageH    = 297.0
	pdfML       = 15.0
	pdfMR       = 15.0
	pdfContentW = pdfPageW - pdfML - pdfMR
	pdfHeaderH  = 44.0
	pdfFooterH  = 18.0
	pdfFooterY  = pdfPageH - pdfFooterH
	// Contenido empieza justo debajo del header + 4mm de respiro
	pdfContentY = pdfHeaderH + 4.0
)

func drawHeader(f *gofpdf.Fpdf, tr func(string) string) {
	// Fondo navy
	f.SetFillColor(15, 23, 42)
	f.Rect(0, 0, pdfPageW, pdfHeaderH, "F")

	// Logo
	f.Image("assets/logo.png", 6, 6, 30, 30, false, "", 0, "")

	// Nombre clínica
	f.SetFont("Helvetica", "B", 15)
	f.SetTextColor(255, 255, 255)
	f.SetXY(40, 10)
	f.Cell(110, 8, tr("TÉCNICA DENTAL JC"))

	// Subtítulo
	f.SetFont("Helvetica", "", 7)
	f.SetTextColor(140, 170, 190)
	f.SetXY(40, 21)
	f.Cell(110, 5, tr("ESTÉTICA ORAL & PREVENCIÓN"))

	// Contacto (derecha)
	f.SetFont("Helvetica", "", 6.5)
	f.SetTextColor(120, 155, 180)
	f.SetXY(pdfPageW-78, 11)
	f.CellFormat(63, 4, "Cali, Valle del Cauca — Colombia", "", 1, "R", false, 0, "")
	f.SetX(pdfPageW - 78)
	f.CellFormat(63, 4, "Tel: 317 819 3836", "", 1, "R", false, 0, "")

	// Línea gold
	f.SetDrawColor(212, 160, 23)
	f.SetLineWidth(1.0)
	f.Line(0, pdfHeaderH, pdfPageW, pdfHeaderH)

	f.SetDrawColor(0, 0, 0)
	f.SetLineWidth(0.2)
	f.SetTextColor(15, 23, 42)
}

func drawFooter(f *gofpdf.Fpdf, tr func(string) string) {
	// Línea gold
	f.SetDrawColor(212, 160, 23)
	f.SetLineWidth(0.8)
	f.Line(0, pdfFooterY, pdfPageW, pdfFooterY)

	// Fondo navy
	f.SetFillColor(15, 23, 42)
	f.Rect(0, pdfFooterY, pdfPageW, pdfFooterH, "F")

	// Texto legal
	f.SetFont("Helvetica", "I", 6)
	f.SetTextColor(120, 155, 180)
	f.SetXY(pdfML, pdfFooterY+4)
	f.CellFormat(140, 4,
		tr("Documento confidencial — uso interno Técnica Dental JC."),
		"", 1, "L", false, 0, "")
	f.SetX(pdfML)
	f.CellFormat(140, 4,
		fmt.Sprintf("Generado: %s", time.Now().Format("02/01/2006 15:04:05")),
		"", 0, "L", false, 0, "")

	// Número de página
	f.SetFont("Helvetica", "", 7)
	f.SetTextColor(180, 205, 220)
	f.SetXY(pdfPageW-pdfMR-35, pdfFooterY+4)
	f.CellFormat(35, 4, fmt.Sprintf(tr("Página %d"), f.PageNo()), "", 0, "R", false, 0, "")

	f.SetTextColor(15, 23, 42)
	f.SetDrawColor(0, 0, 0)
	f.SetLineWidth(0.2)
}

func GenerateHistoryPDF(history domain.MedicalHistory) ([]byte, error) {
	// Sin margen top automático — lo manejamos manualmente
	f := gofpdf.New("P", "mm", "A4", "")
	f.SetMargins(pdfML, pdfContentY, pdfMR)
	f.SetAutoPageBreak(false, 0) // Manejamos page breaks manualmente
	tr := f.UnicodeTranslatorFromDescriptor("")

	// --- Preparar datos ---
	patientName, patientDoc := "N/A", "N/A"
	if history.Appointment.Patient.ID != 0 {
		patientName = tr(fmt.Sprintf("%s %s",
			history.Appointment.Patient.FirstName,
			history.Appointment.Patient.LastName))
		patientDoc = history.Appointment.Patient.DocumentNumber
	}

	doctorName, doctorLicense, doctorSpecialty := "N/A", "N/A", "N/A"
	if history.Appointment.Specialist.ID != 0 {
		doctorName = tr(fmt.Sprintf("%s %s",
			history.Appointment.Specialist.FirstName,
			history.Appointment.Specialist.LastName))
		doctorLicense = history.Appointment.Specialist.LicenseNumber
		doctorSpecialty = tr(history.Appointment.Specialist.Specialty)
	}

	consultDate := "N/A"
	if !history.Appointment.StartTime.IsZero() {
		consultDate = history.Appointment.StartTime.Format("02/01/2006 15:04")
	}

	serviceName := "Servicio"
	if history.Appointment.Service.Name != "" {
		serviceName = tr(history.Appointment.Service.Name)
	}

	nextAppt := ""
	if history.NextAppointmentDate != nil && !history.NextAppointmentDate.IsZero() {
		nextAppt = history.NextAppointmentDate.Format("02/01/2006")
	}

	// Helper: verificar si necesitamos nueva página
	needsNewPage := func(requiredMM float64) bool {
		return f.GetY()+requiredMM > pdfFooterY-5
	}

	addNewPage := func() {
		drawFooter(f, tr)
		f.AddPage()
		drawHeader(f, tr)
		f.SetY(pdfContentY)
	}

	// ========================
	// PÁGINA 1
	// ========================
	f.AddPage()
	drawHeader(f, tr)
	f.SetY(pdfContentY)

	// ========================
	// BANDA TÍTULO
	// ========================
	y := f.GetY()
	f.SetFillColor(248, 250, 252)
	f.SetDrawColor(226, 232, 240)
	f.SetLineWidth(0.3)
	f.Rect(pdfML, y, pdfContentW, 11, "FD")

	f.SetFillColor(8, 145, 178)
	f.Rect(pdfML, y, 3.5, 11, "F")

	f.SetFont("Helvetica", "B", 9)
	f.SetTextColor(15, 23, 42)
	f.SetXY(pdfML+6, y+3)
	f.Cell(95, 5, tr(fmt.Sprintf("RESUMEN DE ATENCIÓN — %s", serviceName)))

	f.SetFont("Helvetica", "", 7.5)
	f.SetTextColor(100, 116, 139)
	f.SetXY(pdfML+95, y+3)
	f.CellFormat(pdfContentW-95, 5,
		fmt.Sprintf("Ref. #%d  |  Fecha: %s", history.AppointmentID, consultDate),
		"", 0, "R", false, 0, "")

	f.SetY(y + 15)

	// ========================
	// GRID PACIENTE / PROFESIONAL
	// ========================
	y = f.GetY()
	colW := (pdfContentW - 5) / 2

	drawDataBox := func(x, startY, width float64, title string, rows [][2]string) float64 {
		rowH := 7.5
		boxH := 8.0 + float64(len(rows))*rowH + 3.0
		f.SetFillColor(241, 245, 249)
		f.SetDrawColor(226, 232, 240)
		f.SetLineWidth(0.3)
		f.Rect(x, startY, width, boxH, "FD")
		f.SetFont("Helvetica", "B", 7)
		f.SetTextColor(8, 145, 178)
		f.SetXY(x+3, startY+3)
		f.Cell(width-6, 4, title)
		for i, row := range rows {
			ry := startY + 9 + float64(i)*rowH
			f.SetFont("Helvetica", "B", 7)
			f.SetTextColor(100, 116, 139)
			f.SetXY(x+3, ry)
			f.Cell(28, 4, row[0])
			f.SetFont("Helvetica", "", 8.5)
			f.SetTextColor(15, 23, 42)
			f.SetXY(x+33, ry)
			f.Cell(width-36, 4, row[1])
		}
		return boxH
	}

	leftH := drawDataBox(pdfML, y, colW, "DATOS DEL PACIENTE", [][2]string{
		{"Nombre:", patientName},
		{"Documento:", patientDoc},
		{tr("Paciente ID:"), fmt.Sprintf("#%d", history.PatientID)},
	})
	rightH := drawDataBox(pdfML+colW+5, y, colW, "PROFESIONAL TRATANTE", [][2]string{
		{"Dr(a):", doctorName},
		{"Licencia:", doctorLicense},
		{tr("Especialidad:"), doctorSpecialty},
	})
	if rightH > leftH {
		leftH = rightH
	}
	f.SetY(y + leftH + 6)

	// ========================
	// SECCIONES CLÍNICAS
	// ========================
	addSection := func(title, content string, useTeal bool) {
		if content == "" {
			return
		}
		if needsNewPage(30) {
			addNewPage()
		}

		sY := f.GetY()

		// Encabezado sección
		if useTeal {
			f.SetFillColor(8, 145, 178)
		} else {
			f.SetFillColor(15, 23, 42)
		}
		f.Rect(pdfML, sY, pdfContentW, 8, "F")
		f.SetFont("Helvetica", "B", 8.5)
		f.SetTextColor(255, 255, 255)
		f.SetXY(pdfML+5, sY+1.8)
		f.Cell(pdfContentW-10, 5, tr(title))

		f.SetY(sY + 11)
		contentStartY := f.GetY()

		// Establecer fuente antes de SplitLines para que use las métricas correctas
		f.SetFont("Helvetica", "", 9)
		f.SetTextColor(15, 23, 42)
		lineH := 5.5

		// Dividir el texto en líneas que caben en el ancho disponible
		rawLines := f.SplitLines([]byte(tr(content)), pdfContentW-12)

		for _, line := range rawLines {
			if needsNewPage(lineH + 8) {
				// Cerrar caja en la página actual
				curY := f.GetY()
				f.SetDrawColor(226, 232, 240)
				f.SetLineWidth(0.3)
				f.Rect(pdfML, contentStartY-3, pdfContentW, curY-contentStartY+5, "D")
				f.SetDrawColor(8, 145, 178)
				f.SetLineWidth(1.8)
				f.Line(pdfML, contentStartY-3, pdfML, curY+2)
				f.SetDrawColor(0, 0, 0)
				f.SetLineWidth(0.2)

				addNewPage()
				contentStartY = f.GetY()
				f.SetFont("Helvetica", "", 9)
				f.SetTextColor(15, 23, 42)
			}
			f.SetX(pdfML + 6)
			f.CellFormat(pdfContentW-12, lineH, string(line), "", 1, "L", false, 0, "")
		}

		contentEndY := f.GetY()

		// Borde y barra acento
		f.SetDrawColor(226, 232, 240)
		f.SetLineWidth(0.3)
		f.Rect(pdfML, contentStartY-3, pdfContentW, contentEndY-contentStartY+5, "D")
		f.SetDrawColor(8, 145, 178)
		f.SetLineWidth(1.8)
		f.Line(pdfML, contentStartY-3, pdfML, contentEndY+2)

		f.SetDrawColor(0, 0, 0)
		f.SetLineWidth(0.2)
		f.SetY(contentEndY + 7)
	}

	addSection("DIAGNÓSTICO", history.Diagnosis, true)
	addSection("PROCEDIMIENTO REALIZADO", history.Treatment, false)
	addSection("NOTAS Y OBSERVACIONES", history.DoctorNotes, true)
	if history.Attachments != "" {
		addSection("ADJUNTOS Y REFERENCIAS", history.Attachments, false)
	}
	if nextAppt != "" {
		addSection(tr("PRÓXIMA CITA"), tr(fmt.Sprintf("Fecha programada: %s", nextAppt)), true)
	}

	// ========================
	// ZONA DE FIRMAS
	// ========================
	if needsNewPage(45) {
		addNewPage()
	}
	f.Ln(6)
	sigY := f.GetY()

	f.SetDrawColor(212, 160, 23)
	f.SetLineWidth(0.8)
	f.Line(pdfML, sigY, pdfML+pdfContentW, sigY)
	f.Ln(8)
	sigY = f.GetY()

	sigColW := pdfContentW / 3
	for i, col := range []struct{ label, name, id string }{
		{"FIRMA PROFESIONAL", doctorName, fmt.Sprintf("Reg. %s", doctorLicense)},
		{"SELLO / REGISTRO", tr("Técnica Dental JC"), "Cali, Colombia"},
		{"FIRMA PACIENTE", patientName, fmt.Sprintf("C.C. %s", patientDoc)},
	} {
		x := pdfML + float64(i)*sigColW
		f.SetDrawColor(100, 116, 139)
		f.SetLineWidth(0.4)
		f.Line(x+5, sigY+18, x+sigColW-5, sigY+18)
		f.SetFont("Helvetica", "B", 7)
		f.SetTextColor(100, 116, 139)
		f.SetXY(x, sigY+20)
		f.CellFormat(sigColW, 4, col.label, "", 1, "C", false, 0, "")
		f.SetFont("Helvetica", "B", 8)
		f.SetTextColor(15, 23, 42)
		f.SetX(x)
		f.CellFormat(sigColW, 4, col.name, "", 1, "C", false, 0, "")
		f.SetFont("Helvetica", "", 7)
		f.SetTextColor(100, 116, 139)
		f.SetX(x)
		f.CellFormat(sigColW, 4, col.id, "", 0, "C", false, 0, "")
	}

	// Footer última página
	drawFooter(f, tr)

	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
