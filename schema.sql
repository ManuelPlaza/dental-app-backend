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
    "is_active" BOOLEAN DEFAULT TRUE
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

-- ... (resto del código arriba) ...

-- RELACIONES (Foreign Keys) - ASEGÚRATE DE COPIAR TODO ESTE BLOQUE FINAL
ALTER TABLE "Services" ADD CONSTRAINT "services_category_fk" FOREIGN KEY ("category_id") REFERENCES "ServiceCategories"("id");
ALTER TABLE "Patient" ADD CONSTRAINT "patient_user_fk" FOREIGN KEY ("user_id") REFERENCES "Users"("id");
ALTER TABLE "Patient" ADD CONSTRAINT "patient_referral_fk" FOREIGN KEY ("referred_by_id") REFERENCES "Patient"("id");
ALTER TABLE "Specialists" ADD CONSTRAINT "specialists_user_fk" FOREIGN KEY ("user_id") REFERENCES "Users"("id");

ALTER TABLE "Appointments" ADD CONSTRAINT "appointments_patient_fk" FOREIGN KEY ("patient_id") REFERENCES "Patient"("id");
ALTER TABLE "Appointments" ADD CONSTRAINT "appointments_specialist_fk" FOREIGN KEY ("specialist_id") REFERENCES "Specialists"("id");
ALTER TABLE "Appointments" ADD CONSTRAINT "appointments_service_fk" FOREIGN KEY ("service_id") REFERENCES "Services"("id");

ALTER TABLE "Payments" ADD CONSTRAINT "payments_appointment_fk" FOREIGN KEY ("appointment_id") REFERENCES "Appointments"("id");

ALTER TABLE "MedicalHistory" ADD CONSTRAINT "medical_patient_fk" FOREIGN KEY ("patient_id") REFERENCES "Patient"("id");
ALTER TABLE "MedicalHistory" ADD CONSTRAINT "medical_appointment_fk" FOREIGN KEY ("appointment_id") REFERENCES "Appointments"("id");

ALTER TABLE "LoyaltyTransactions" ADD CONSTRAINT "loyalty_patient_fk" FOREIGN KEY ("patient_id") REFERENCES "Patient"("id");
ALTER TABLE "LoyaltyTransactions" ADD CONSTRAINT "loyalty_appointment_fk" FOREIGN KEY ("appointment_id") REFERENCES "Appointments"("id");