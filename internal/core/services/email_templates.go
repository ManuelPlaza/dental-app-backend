package services

import (
	"bytes"
	"html/template"
	"time"
)

var bogotaLocEmail, _ = time.LoadLocation("America/Bogota")

type EmailTemplate struct {
	Subject string
	Body    string
}

type confirmationData struct {
	PatientName    string
	SpecialistName string
	Date           string
	ConfirmURL     string
}

type reminderData struct {
	PatientName    string
	SpecialistName string
	Date           string
}

var confirmationTmpl = template.Must(template.New("confirmation").Parse(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family:Arial,sans-serif;background:#f4f4f4;padding:20px;margin:0;">
  <div style="max-width:600px;margin:auto;background:white;border-radius:12px;overflow:hidden;box-shadow:0 2px 10px rgba(0,0,0,0.1);">
    <div style="background:#0ea5e9;padding:30px;text-align:center;">
      <h1 style="color:white;margin:0;font-size:24px;">🦷 Dental JC</h1>
      <p style="color:rgba(255,255,255,0.85);margin:8px 0 0;">Alfonso López</p>
    </div>
    <div style="padding:30px;">
      <h2 style="color:#1e293b;">Confirma tu cita</h2>
      <p style="color:#475569;">Hola <strong>{{.PatientName}}</strong>,</p>
      <p style="color:#475569;">Tu cita con <strong>{{.SpecialistName}}</strong> está programada para:</p>
      <div style="background:#f0f9ff;border-left:4px solid #0ea5e9;padding:16px;border-radius:6px;margin:20px 0;">
        <p style="margin:0;color:#0369a1;font-size:18px;font-weight:bold;">📅 {{.Date}}</p>
      </div>
      <div style="text-align:center;margin:30px 0;">
        <a href="{{.ConfirmURL}}" style="display:inline-block;padding:14px 32px;background:#0ea5e9;color:white;border-radius:8px;text-decoration:none;font-weight:bold;font-size:16px;">
          ✅ Confirmar mi cita
        </a>
      </div>
      <p style="color:#94a3b8;font-size:13px;">Si no puedes asistir, comunícate con nosotros con al menos 2 horas de anticipación.</p>
    </div>
    <div style="background:#f8fafc;padding:20px;text-align:center;">
      <p style="color:#94a3b8;font-size:12px;margin:0;">© 2026 Dental JC - Alfonso López</p>
    </div>
  </div>
</body>
</html>`))

var reminder24hTmpl = template.Must(template.New("reminder24h").Parse(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family:Arial,sans-serif;background:#f4f4f4;padding:20px;margin:0;">
  <div style="max-width:600px;margin:auto;background:white;border-radius:12px;overflow:hidden;">
    <div style="background:#f59e0b;padding:30px;text-align:center;">
      <h1 style="color:white;margin:0;">⏰ Recordatorio</h1>
      <p style="color:rgba(255,255,255,0.85);margin:8px 0 0;">Tu cita es mañana</p>
    </div>
    <div style="padding:30px;">
      <p style="color:#475569;">Hola <strong>{{.PatientName}}</strong>,</p>
      <p style="color:#475569;">Te recordamos que mañana tienes cita con <strong>{{.SpecialistName}}</strong>:</p>
      <div style="background:#fffbeb;border-left:4px solid #f59e0b;padding:16px;border-radius:6px;margin:20px 0;">
        <p style="margin:0;color:#92400e;font-size:18px;font-weight:bold;">📅 {{.Date}}</p>
      </div>
      <p style="color:#475569;">Recuerda llegar 10 minutos antes de tu cita.</p>
    </div>
    <div style="background:#f8fafc;padding:20px;text-align:center;">
      <p style="color:#94a3b8;font-size:12px;margin:0;">© 2026 Dental JC - Alfonso López</p>
    </div>
  </div>
</body>
</html>`))

var reminderFinalTmpl = template.Must(template.New("reminderFinal").Parse(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family:Arial,sans-serif;background:#f4f4f4;padding:20px;margin:0;">
  <div style="max-width:600px;margin:auto;background:white;border-radius:12px;overflow:hidden;">
    <div style="background:#10b981;padding:30px;text-align:center;">
      <h1 style="color:white;margin:0;">🦷 ¡Tu cita es hoy!</h1>
      <p style="color:rgba(255,255,255,0.85);margin:8px 0 0;">Te esperamos</p>
    </div>
    <div style="padding:30px;">
      <p style="color:#475569;">Hola <strong>{{.PatientName}}</strong>,</p>
      <p style="color:#475569;">¡Hoy es tu cita con <strong>{{.SpecialistName}}</strong>!</p>
      <div style="background:#ecfdf5;border-left:4px solid #10b981;padding:16px;border-radius:6px;margin:20px 0;">
        <p style="margin:0;color:#065f46;font-size:18px;font-weight:bold;">📅 {{.Date}}</p>
      </div>
      <p style="color:#475569;">Recuerda llegar 10 minutos antes. ¡Te esperamos!</p>
    </div>
    <div style="background:#f8fafc;padding:20px;text-align:center;">
      <p style="color:#94a3b8;font-size:12px;margin:0;">© 2026 Dental JC - Alfonso López</p>
    </div>
  </div>
</body>
</html>`))

func renderConfirmation(data confirmationData) (string, error) {
	var buf bytes.Buffer
	if err := confirmationTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderReminder24h(data reminderData) (string, error) {
	var buf bytes.Buffer
	if err := reminder24hTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderReminderFinal(data reminderData) (string, error) {
	var buf bytes.Buffer
	if err := reminderFinalTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
