-- 1. Tabla de Usuarios
CREATE TABLE "Users" (
    "id" SERIAL PRIMARY KEY,
    "first_name" VARCHAR(255) NOT NULL,
    "last_name" VARCHAR(255) NOT NULL,
    "email" VARCHAR(255) UNIQUE NOT NULL,
    "password" VARCHAR(255) NOT NULL,
    "state" VARCHAR(50) DEFAULT 'active',
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Categorías
CREATE TABLE "ServiceCategories" (
    "id" SERIAL PRIMARY KEY,
    "name" VARCHAR(255) NOT NULL,
    "description" VARCHAR(255)
);

-- 3. Servicios
CREATE TABLE "Services" (
    "id" SERIAL PRIMARY KEY,
    "category_id" INTEGER NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "description" TEXT,
    "price" DECIMAL(10, 2) NOT NULL,
    "duration_minutes" INTEGER NOT NULL DEFAULT 60,
    "is_active" BOOLEAN DEFAULT TRUE,
    "show_on_web" BOOLEAN DEFAULT FALSE
);

-- 4. Pacientes
CREATE TABLE "Patient" (
    "id" SERIAL PRIMARY KEY,
    "user_id" INTEGER UNIQUE, -- Puede ser Null si no tiene usuario web
    "document_number" VARCHAR(50) UNIQUE NOT NULL,
    "emergency_contact_name" VARCHAR(255),
    "emergency_contact_relationship" VARCHAR(100),
    "emergency_contact_phone" VARCHAR(50), -- Typo corregido
    "birth_date" DATE,
    "loyalty_points_balance" INTEGER DEFAULT 0, -- Agregado para fidelización
    "referred_by_id" INTEGER
);

-- 5. Especialistas
CREATE TABLE "Specialists" (
    "id" SERIAL PRIMARY KEY,
    "user_id" INTEGER UNIQUE NOT NULL,
    "first_name" VARCHAR(255) NOT NULL,
    "last_name" VARCHAR(255) NOT NULL,
    "specialty" VARCHAR(100),
    "license_number" VARCHAR(100) NOT NULL,
    "phone" VARCHAR(50),
    "is_active" BOOLEAN DEFAULT TRUE,
    "is_default" BOOLEAN DEFAULT FALSE,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 6. Citas
CREATE TABLE "Appointments" (
    "id" SERIAL PRIMARY KEY,
    "patient_id" INTEGER NOT NULL,
    "specialist_id" INTEGER NOT NULL,
    "service_id" INTEGER NOT NULL,
    "start_time" TIMESTAMP NOT NULL,
    "end_time" TIMESTAMP NOT NULL,
    "status" VARCHAR(50) DEFAULT 'pending', -- pending, confirmed, completed, cancelled
    "historical_price" DECIMAL(10, 2),
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 7. Pagos
CREATE TABLE "Payments" (
    "id" SERIAL PRIMARY KEY,
    "appointment_id" INTEGER NOT NULL,
    "amount" DECIMAL(10, 2) NOT NULL,
    "method" VARCHAR(50) NOT NULL, -- cash, nequi, loyalty_points
    "reference_code" VARCHAR(100),
    "notes" TEXT, -- Typo corregido
    "payment_date" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 8. Historia Médica
CREATE TABLE "MedicalHistory" (
    "id" SERIAL PRIMARY KEY,
    "patient_id" INTEGER NOT NULL,
    "appointment_id" INTEGER UNIQUE NOT NULL,
    "diagnosis" TEXT,
    "treatment" TEXT,
    "doctor_notes" TEXT,
    "attachments" TEXT, -- Guardaremos JSON como texto por compatibilidad simple
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 9. Transacciones de Fidelización (Puntos)
CREATE TABLE "LoyaltyTransactions" (
    "id" SERIAL PRIMARY KEY,
    "patient_id" INTEGER NOT NULL,
    "appointment_id" INTEGER,
    "amount" INTEGER NOT NULL, -- Positivo gana, negativo gasta
    "type" VARCHAR(50) NOT NULL, -- earned, redeemed
    "description" VARCHAR(255),
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de cola de notificaciones
CREATE TABLE "NotificationQueue" (
    id SERIAL PRIMARY KEY,
    appointment_id INTEGER NOT NULL REFERENCES "Appointments"(id),
    patient_id INTEGER NOT NULL REFERENCES "Patient"(id),
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    scheduled_at TIMESTAMP NOT NULL,
    sent_at TIMESTAMP,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    token VARCHAR(100) UNIQUE,
    error_log TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notification_status ON "NotificationQueue"(status);
CREATE INDEX idx_notification_scheduled ON "NotificationQueue"(scheduled_at);
CREATE INDEX idx_notification_token ON "NotificationQueue"(token);
CREATE INDEX idx_notification_appointment ON "NotificationQueue"(appointment_id);

-- Tabla de logs de notificaciones
CREATE TABLE "NotificationLogs" (
    id SERIAL PRIMARY KEY,
    notification_id INTEGER NOT NULL REFERENCES "NotificationQueue"(id),
    event VARCHAR(50) NOT NULL,
    detail TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notif_log_notification ON "NotificationLogs"(notification_id);

CREATE TABLE "Banners" (
    "id"                SERIAL PRIMARY KEY,
    "title"             VARCHAR(255) NOT NULL,
    "description"       TEXT,
    "image_url_desktop" VARCHAR(500) NOT NULL,
    "image_url_mobile"  VARCHAR(500),
    "redirect_url"      VARCHAR(500),
    "start_time"        TIMESTAMP NOT NULL,
    "end_time"          TIMESTAMP NOT NULL,
    "is_active"         BOOLEAN NOT NULL DEFAULT TRUE,
    "priority"          INTEGER NOT NULL DEFAULT 0,
    "created_at"        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deleted_at"        TIMESTAMP
);

CREATE INDEX idx_banners_start_time ON "Banners" ("start_time");
CREATE INDEX idx_banners_end_time   ON "Banners" ("end_time");
CREATE INDEX idx_banners_is_active  ON "Banners" ("is_active");


-- RELACIONES (Foreign Keys) - ASEGÚRATE DE COPIAR TODO ESTE BLOQUE FINAL
ALTER TABLE "Services" ADD CONSTRAINT "services_category_fk" FOREIGN KEY ("category_id") REFERENCES "ServiceCategories"("id");
ALTER TABLE "Patient" ADD CONSTRAINT "patient_user_fk" FOREIGN KEY ("user_id") REFERENCES "Users"("id");
ALTER TABLE "Patient" ADD CONSTRAINT "patient_referral_fk" FOREIGN KEY ("referred_by_id") REFERENCES "Patient"("id");
ALTER TABLE "Patient" ADD COLUMN  phone VARCHAR(50);
ALTER TABLE "Patient" ADD COLUMN  first_name VARCHAR(255);
ALTER TABLE "Patient" ADD COLUMN  last_name VARCHAR(255);
ALTER TABLE "Patient" ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "Patient" ADD COLUMN email VARCHAR(255);

ALTER TABLE "Specialists" ADD CONSTRAINT "specialists_user_fk" FOREIGN KEY ("user_id") REFERENCES "Users"("id");
ALTER TABLE "Specialists" ALTER COLUMN "user_id" DROP NOT NULL;

ALTER TABLE "Appointments" ADD COLUMN  modification_count INTEGER DEFAULT 0;
ALTER TABLE "Appointments" ADD COLUMN notes TEXT;
ALTER TABLE "Appointments" ADD COLUMN cancellation_reason VARCHAR(100);
ALTER TABLE "Appointments" ADD COLUMN cancellation_notes TEXT;
ALTER TABLE "Appointments" ADD CONSTRAINT "appointments_patient_fk" FOREIGN KEY ("patient_id") REFERENCES "Patient"("id");
ALTER TABLE "Appointments" ADD CONSTRAINT "appointments_specialist_fk" FOREIGN KEY ("specialist_id") REFERENCES "Specialists"("id");
ALTER TABLE "Appointments" ADD CONSTRAINT "appointments_service_fk" FOREIGN KEY ("service_id") REFERENCES "Services"("id");


ALTER TABLE "Payments" ADD CONSTRAINT "payments_appointment_fk" FOREIGN KEY ("appointment_id") REFERENCES "Appointments"("id");
ALTER TABLE "Payments" ADD COLUMN  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "Payments" ADD COLUMN  status VARCHAR(50) DEFAULT 'completed'; -- completed, refunded

ALTER TABLE "MedicalHistory" ADD CONSTRAINT "medical_patient_fk" FOREIGN KEY ("patient_id") REFERENCES "Patient"("id");
ALTER TABLE "MedicalHistory" ADD CONSTRAINT "medical_appointment_fk" FOREIGN KEY ("appointment_id") REFERENCES "Appointments"("id");
ALTER TABLE "MedicalHistory" ADD COLUMN next_appointment_date TIMESTAMP;

ALTER TABLE "LoyaltyTransactions" ADD CONSTRAINT "loyalty_patient_fk" FOREIGN KEY ("patient_id") REFERENCES "Patient"("id");
ALTER TABLE "LoyaltyTransactions" ADD CONSTRAINT "loyalty_appointment_fk" FOREIGN KEY ("appointment_id") REFERENCES "Appointments"("id");

-- =====================================================================
-- MIGRACIÓN: Links de Pago Nequi
-- =====================================================================
CREATE TABLE IF NOT EXISTS "PaymentLinks" (
    id              SERIAL PRIMARY KEY,
    appointment_id  INTEGER NOT NULL,
    amount          NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    token           UUID NOT NULL UNIQUE,
    nequi_txn_id    VARCHAR(100),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','paid','expired','rejected','cancelled')),
    phone_number    VARCHAR(20) NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    paid_at         TIMESTAMPTZ,
    created_by      INTEGER,
    webhook_payload TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_links_token       ON "PaymentLinks"(token);
CREATE INDEX IF NOT EXISTS idx_payment_links_appointment ON "PaymentLinks"(appointment_id);
CREATE INDEX IF NOT EXISTS idx_payment_links_status      ON "PaymentLinks"(status);
CREATE INDEX IF NOT EXISTS idx_payment_links_expires_at  ON "PaymentLinks"(expires_at);
CREATE INDEX IF NOT EXISTS idx_payment_links_nequi_txn   ON "PaymentLinks"(nequi_txn_id) WHERE nequi_txn_id IS NOT NULL;

ALTER TABLE "PaymentLinks"
    ADD CONSTRAINT fk_payment_links_appointment
    FOREIGN KEY (appointment_id) REFERENCES "Appointments"(id);