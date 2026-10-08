-- =====================================================================
-- customer-onboarding-service | PostgreSQL 14+ (disarankan 15/16)
-- Schema : bankservices
-- Versi  : revisi setelah review (alur: customer PENDING dibuat saat case dibuat)
-- Catatan: BEFORE ROW trigger pada tabel partisi butuh PG13+.
-- =====================================================================

BEGIN;

CREATE SCHEMA IF NOT EXISTS bankservices;
SET LOCAL search_path = bankservices, public;

-- ---------------------------------------------------------------------
-- 1. ENUM TYPES
-- ---------------------------------------------------------------------
CREATE TYPE customer_type        AS ENUM ('INDIVIDUAL', 'CORPORATE');
CREATE TYPE customer_status      AS ENUM ('PENDING', 'ACTIVE', 'SUSPENDED', 'CLOSED');
CREATE TYPE risk_level           AS ENUM ('LOW', 'MEDIUM', 'HIGH');
CREATE TYPE onboarding_channel   AS ENUM ('MOBILE', 'CORPORATE_PORTAL', 'INTERNAL_TELLER');
CREATE TYPE case_stage           AS ENUM (
    'INITIATED', 'DOCS_SUBMITTED', 'IDENTITY_VERIFIED', 'RISK_ASSESSED',
    'VERIFICATION_DONE', 'PENDING_REVIEW', 'PENDING_APPROVAL',
    'CIF_CREATION', 'ACCOUNT_PROVISIONING',
    'COMPLETED', 'REJECTED', 'CANCELLED', 'EXPIRED');
CREATE TYPE saga_state           AS ENUM (
    'STARTED', 'IN_PROGRESS', 'WAITING_EXTERNAL',
    'COMPENSATING', 'COMPENSATED', 'COMPLETED', 'FAILED');
CREATE TYPE verification_status  AS ENUM ('PROCESSING', 'VERIFIED', 'REJECTED', 'MANUAL_REVIEW_REQUIRED');
CREATE TYPE verification_subject AS ENUM ('APPLICANT', 'BENEFICIAL_OWNER', 'SIGNATORY');
CREATE TYPE review_status        AS ENUM ('PENDING', 'VERIFIED', 'REJECTED');
CREATE TYPE approval_decision    AS ENUM ('PENDING', 'APPROVED', 'REJECTED');
CREATE TYPE signatory_role       AS ENUM ('DIRECTOR', 'COMMISSIONER', 'AUTHORIZED_PROXY');
CREATE TYPE authority_level      AS ENUM ('SINGLE_SIGNATURE', 'JOINT_SIGNATURE');

-- ---------------------------------------------------------------------
-- 2. SEQUENCE & ID GENERATOR
--    ONB-20260923-000123 | VER-99881
-- ---------------------------------------------------------------------
CREATE SEQUENCE onboarding_case_seq START 1 CACHE 20;
CREATE SEQUENCE verification_seq    START 1 CACHE 20;

CREATE FUNCTION next_case_id() RETURNS varchar
LANGUAGE sql VOLATILE AS $$
    SELECT 'ONB-' || to_char(now() AT TIME ZONE 'Asia/Jakarta', 'YYYYMMDD')
           || '-' || lpad(nextval('bankservices.onboarding_case_seq')::text, 6, '0')
$$;

CREATE FUNCTION next_verification_id() RETURNS varchar
LANGUAGE sql VOLATILE AS $$
    SELECT 'VER-' || lpad(nextval('bankservices.verification_seq')::text, 5, '0')
$$;

-- ---------------------------------------------------------------------
-- 3. TRIGGER FUNCTIONS
-- ---------------------------------------------------------------------
CREATE FUNCTION fn_set_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END $$;

-- Optimistic locking: aplikasi memakai UPDATE ... WHERE case_id=? AND version=?
CREATE FUNCTION fn_case_touch() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    NEW.version    := OLD.version + 1;
    RETURN NEW;
END $$;

CREATE FUNCTION fn_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% pada %.% tidak diizinkan (append-only / audit)',
        TG_OP, TG_TABLE_SCHEMA, TG_TABLE_NAME
        USING ERRCODE = 'restrict_violation';
END $$;

-- [NEW] Saat customer CLOSED (case ditolak/dibatalkan/expired), lepaskan NIB/NPWP
-- agar pengajuan ulang dengan NIB/NPWP yang sama tidak terblokir unique index.
CREATE FUNCTION fn_customer_closed() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE bankservices.corporate_entity
       SET is_active = false
     WHERE customer_id = NEW.customer_id;
    RETURN NEW;
END $$;

-- ---------------------------------------------------------------------
-- 4. TABEL REFERENSI
-- ---------------------------------------------------------------------
CREATE TABLE ref_document_type (
    doc_type    varchar(40) PRIMARY KEY,
    description text        NOT NULL,
    is_active   boolean     NOT NULL DEFAULT true
);
INSERT INTO ref_document_type (doc_type, description) VALUES
    ('KTP',             'Foto KTP'),
    ('SELFIE',          'Foto selfie'),
    ('LIVENESS_VIDEO',  'Video liveness'),                 -- [NEW]
    ('NPWP',            'NPWP pribadi'),
    ('NPWP_BADAN',      'NPWP badan usaha'),
    ('NIB',             'Nomor Induk Berusaha'),
    ('AKTA_PENDIRIAN',  'Akta pendirian perusahaan'),
    ('AKTA_PERUBAHAN',  'Akta perubahan perusahaan'),
    ('SK_MENKUMHAM',    'SK pengesahan Kemenkumham');

CREATE TABLE ref_reject_reason (
    reject_reason varchar(40) PRIMARY KEY,
    description   text        NOT NULL,
    is_active     boolean     NOT NULL DEFAULT true
);
INSERT INTO ref_reject_reason (reject_reason, description) VALUES
    ('FACE_MISMATCH',      'Wajah selfie tidak cocok dengan foto KTP'),
    ('LIVENESS_FAILED',    'Liveness detection gagal'),
    ('DUKCAPIL_MISMATCH',  'Data tidak cocok dengan Dukcapil'),
    ('OCR_UNREADABLE',     'KTP tidak terbaca OCR'),
    ('IMAGE_QUALITY_LOW',  'Kualitas gambar tidak memenuhi syarat');

-- ---------------------------------------------------------------------
-- 5. CUSTOMER (System of Record CIF)
--    Dibuat berstatus PENDING saat case dibuat (dalam 1 transaksi dengan case).
-- ---------------------------------------------------------------------
CREATE TABLE customer (
    customer_id  uuid            PRIMARY KEY DEFAULT gen_random_uuid(),
    type         customer_type   NOT NULL,
    status       customer_status NOT NULL DEFAULT 'PENDING',
    risk_rating  risk_level,
    cif_number   varchar(20),
    nik_hash     char(64),                                  -- [NEW] sha256(NIK + pepper), bukan NIK mentah
    created_at   timestamptz     NOT NULL DEFAULT now(),
    updated_at   timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT ck_customer_active_has_cif
        CHECK (status <> 'ACTIVE' OR cif_number IS NOT NULL),
    CONSTRAINT ck_customer_nik_hash_individual                 -- [NEW]
        CHECK (nik_hash IS NULL OR type = 'INDIVIDUAL'),
    CONSTRAINT uq_customer_id_type UNIQUE (customer_id, type)  -- target FK komposit
);
CREATE UNIQUE INDEX uq_customer_cif_number
    ON customer (cif_number) WHERE cif_number IS NOT NULL;
-- [NEW] cegah CIF ganda untuk orang yang sama; customer CLOSED tidak ikut dihitung
CREATE UNIQUE INDEX uq_customer_nik_hash
    ON customer (nik_hash) WHERE nik_hash IS NOT NULL AND status <> 'CLOSED';

COMMENT ON TABLE  customer            IS 'Customer Information File (CIF). Baris PENDING dibuat saat case dibuat; ACTIVE setelah CIF terbentuk.';
COMMENT ON COLUMN customer.cif_number IS 'Nomor CIF dari Core Banking (via CIF Module).';
COMMENT ON COLUMN customer.nik_hash   IS 'Hash NIK untuk deteksi duplikat. Diisi setelah eKYC VERIFIED.';

-- ---------------------------------------------------------------------
-- 6. CORPORATE_ENTITY
-- ---------------------------------------------------------------------
CREATE TABLE corporate_entity (
    company_id    uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id   uuid          NOT NULL,
    customer_type customer_type NOT NULL DEFAULT 'CORPORATE',
    nib           varchar(13),
    akta_number   varchar(60),
    npwp_badan    varchar(16),
    legal_status  varchar(30),
    is_active     boolean       NOT NULL DEFAULT true,       -- [NEW] false jika customer CLOSED
    created_at    timestamptz   NOT NULL DEFAULT now(),
    updated_at    timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT uq_corporate_customer UNIQUE (customer_id),
    CONSTRAINT ck_corporate_is_corporate CHECK (customer_type = 'CORPORATE'),
    CONSTRAINT fk_corporate_customer
        FOREIGN KEY (customer_id, customer_type)
        REFERENCES customer (customer_id, type) ON DELETE RESTRICT,
    CONSTRAINT ck_corporate_nib   CHECK (nib ~ '^[0-9]{13}$'),
    CONSTRAINT ck_corporate_npwp  CHECK (npwp_badan ~ '^[0-9]{15,16}$')
);
-- [CHG] unique hanya untuk entitas aktif
CREATE UNIQUE INDEX uq_corporate_nib  ON corporate_entity (nib)        WHERE nib IS NOT NULL AND is_active;
CREATE UNIQUE INDEX uq_corporate_npwp ON corporate_entity (npwp_badan) WHERE npwp_badan IS NOT NULL AND is_active;

COMMENT ON COLUMN corporate_entity.legal_status IS
    'Dibiarkan varchar: nilai belum baku. Jadikan ENUM/ref table setelah domain nilainya final.';

CREATE TRIGGER trg_customer_closed
    AFTER UPDATE OF status ON customer
    FOR EACH ROW
    WHEN (NEW.status = 'CLOSED' AND OLD.status <> 'CLOSED')
    EXECUTE FUNCTION fn_customer_closed();

-- ---------------------------------------------------------------------
-- 7. ONBOARDING_CASE (Saga state)
-- ---------------------------------------------------------------------
CREATE TABLE onboarding_case (
    case_id            varchar(32)        PRIMARY KEY DEFAULT bankservices.next_case_id(),
    customer_id        uuid               NOT NULL,                    -- [CHG] NOT NULL (customer PENDING dibuat bersama case)
    customer_type      customer_type      NOT NULL,
    created_by_subject varchar(64)        NOT NULL,                    -- [NEW] claim "sub" pembuat case (cek kepemilikan)
    applicant_name     varchar(150)       NOT NULL,
    applicant_phone    varchar(20)        NOT NULL,
    applicant_email    varchar(254),
    channel            onboarding_channel NOT NULL,
    current_stage      case_stage         NOT NULL DEFAULT 'INITIATED',
    saga_state         saga_state         NOT NULL DEFAULT 'STARTED',
    risk_level         risk_level,
    version            integer            NOT NULL DEFAULT 0,
    expires_at         timestamptz,
    created_at         timestamptz        NOT NULL DEFAULT now(),
    updated_at         timestamptz        NOT NULL DEFAULT now(),
    CONSTRAINT ck_case_phone_e164 CHECK (applicant_phone ~ '^\+[1-9][0-9]{7,14}$'),
    -- [NEW] case INDIVIDUAL tidak mungkin menunjuk customer CORPORATE
    CONSTRAINT fk_case_customer
        FOREIGN KEY (customer_id, customer_type)
        REFERENCES customer (customer_id, type) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX uq_case_customer ON onboarding_case (customer_id);   -- 1 customer : 1 case
CREATE INDEX ix_case_owner ON onboarding_case (created_by_subject, created_at DESC);  -- [NEW]
CREATE INDEX ix_case_phone ON onboarding_case (applicant_phone);
CREATE INDEX ix_case_open_stage ON onboarding_case (current_stage, updated_at)
    WHERE current_stage NOT IN ('COMPLETED', 'REJECTED', 'CANCELLED', 'EXPIRED');

COMMENT ON COLUMN onboarding_case.created_by_subject IS
    'Nilai claim sub dari JWT. Setiap endpoint /cases/{id}/... wajib memastikan sub pemanggil = kolom ini (atau role staf yang berwenang).';
COMMENT ON COLUMN onboarding_case.current_stage IS
    'API mengembalikan status ringkas (mis. PROCESSING); pemetaan stage -> status API dilakukan di aplikasi.';

-- ---------------------------------------------------------------------
-- 8. IDENTITY_VERIFICATION (hasil eKYC)
-- ---------------------------------------------------------------------
CREATE TABLE identity_verification (
    verification_id      varchar(32)          PRIMARY KEY DEFAULT bankservices.next_verification_id(),
    case_id              varchar(32)          NOT NULL REFERENCES onboarding_case (case_id) ON DELETE RESTRICT,
    subject_type         verification_subject NOT NULL DEFAULT 'APPLICANT',
    attempt_number       smallint             NOT NULL DEFAULT 1,
    status               verification_status  NOT NULL DEFAULT 'PROCESSING',
    face_match_score     numeric(4,3),
    liveness_score       numeric(4,3),
    dukcapil_match       boolean,
    extracted_nik        varchar(16),
    extracted_full_name  varchar(150),
    extracted_birth_date date,
    extracted_address    text,
    reject_reason        varchar(40)          REFERENCES ref_reject_reason (reject_reason),
    created_at           timestamptz          NOT NULL DEFAULT now(),
    completed_at         timestamptz,
    updated_at           timestamptz          NOT NULL DEFAULT now(),
    CONSTRAINT ck_iv_attempt_max3   CHECK (attempt_number BETWEEN 1 AND 3),
    CONSTRAINT ck_iv_face_score     CHECK (face_match_score BETWEEN 0 AND 1),
    CONSTRAINT ck_iv_liveness_score CHECK (liveness_score   BETWEEN 0 AND 1),
    CONSTRAINT ck_iv_nik            CHECK (extracted_nik ~ '^[0-9]{16}$'),
    CONSTRAINT ck_iv_rejected_reason
        CHECK (status <> 'REJECTED' OR reject_reason IS NOT NULL),
    CONSTRAINT ck_iv_verified_complete
        CHECK (status <> 'VERIFIED'
               OR (face_match_score IS NOT NULL AND liveness_score IS NOT NULL
                   AND dukcapil_match IS NOT NULL AND extracted_nik IS NOT NULL))
);
CREATE UNIQUE INDEX uq_iv_applicant_attempt
    ON identity_verification (case_id, attempt_number) WHERE subject_type = 'APPLICANT';
CREATE UNIQUE INDEX uq_iv_applicant_one_processing
    ON identity_verification (case_id) WHERE subject_type = 'APPLICANT' AND status = 'PROCESSING';
CREATE INDEX ix_iv_case ON identity_verification (case_id);

COMMENT ON COLUMN identity_verification.extracted_nik IS
    'PII (UU PDP). Rekomendasi: enkripsi di level aplikasi/KMS; sesuaikan ck_iv_nik jika diubah ke bytea.';
COMMENT ON COLUMN identity_verification.extracted_address IS 'PII - perlakukan sama seperti NIK.';

-- ---------------------------------------------------------------------
-- 9. BENEFICIAL_OWNER & AUTHORIZED_SIGNATORY
-- ---------------------------------------------------------------------
CREATE TABLE beneficial_owner (
    bo_id                uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id           uuid          NOT NULL REFERENCES corporate_entity (company_id) ON DELETE RESTRICT,
    full_name            varchar(150)  NOT NULL,                          -- [NEW]
    verification_id      varchar(32)   REFERENCES identity_verification (verification_id) ON DELETE RESTRICT,
    ownership_percentage numeric(5,2)  NOT NULL,
    status               review_status NOT NULL DEFAULT 'PENDING',
    created_at           timestamptz   NOT NULL DEFAULT now(),
    updated_at           timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT ck_bo_pct CHECK (ownership_percentage > 0 AND ownership_percentage <= 100),
    CONSTRAINT uq_bo_verification UNIQUE (verification_id)
);
CREATE INDEX ix_bo_company ON beneficial_owner (company_id);

CREATE TABLE authorized_signatory (
    signatory_id    uuid            PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      uuid            NOT NULL REFERENCES corporate_entity (company_id) ON DELETE RESTRICT,
    full_name       varchar(150)    NOT NULL,
    role            signatory_role  NOT NULL,
    authority_level authority_level NOT NULL,
    verification_id varchar(32)     REFERENCES identity_verification (verification_id) ON DELETE RESTRICT,
    created_at      timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT uq_signatory_verification UNIQUE (verification_id)
);
CREATE INDEX ix_signatory_company ON authorized_signatory (company_id);

-- ---------------------------------------------------------------------
-- 10. DOCUMENT_REF & APPROVAL_TASK_REF
-- ---------------------------------------------------------------------
CREATE TABLE document_ref (
    document_id         uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id             varchar(32)   NOT NULL REFERENCES onboarding_case (case_id) ON DELETE RESTRICT,
    verification_id     varchar(32)   REFERENCES identity_verification (verification_id) ON DELETE RESTRICT,  -- [NEW]
    doc_type            varchar(40)   NOT NULL REFERENCES ref_document_type (doc_type),
    storage_ref         varchar(255)  NOT NULL,
    verification_status review_status NOT NULL DEFAULT 'PENDING',
    created_at          timestamptz   NOT NULL DEFAULT now(),
    updated_at          timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT uq_document_case_ref UNIQUE (case_id, storage_ref)
);
CREATE INDEX ix_document_case ON document_ref (case_id);
CREATE INDEX ix_document_verification ON document_ref (verification_id) WHERE verification_id IS NOT NULL;

COMMENT ON COLUMN document_ref.verification_id IS
    'Diisi untuk KTP/SELFIE/LIVENESS_VIDEO hasil upload eKYC; NULL untuk dokumen pendukung (NPWP, akta, dst).';

CREATE TABLE approval_task_ref (
    task_id        varchar(64)       PRIMARY KEY,
    case_id        varchar(32)       NOT NULL REFERENCES onboarding_case (case_id) ON DELETE RESTRICT,
    approval_level smallint          NOT NULL DEFAULT 1,            -- [NEW] 1 = Compliance, 2 = Back Office, dst
    approver_role  varchar(30),                                     -- [NEW] mis. COMPLIANCE, BACK_OFFICE
    maker_id       varchar(32)       NOT NULL,
    checker_id     varchar(32),
    decision       approval_decision NOT NULL DEFAULT 'PENDING',
    decided_at     timestamptz,
    created_at     timestamptz       NOT NULL DEFAULT now(),
    CONSTRAINT ck_approval_level    CHECK (approval_level >= 1),
    CONSTRAINT uq_approval_case_level UNIQUE (case_id, approval_level),
    CONSTRAINT ck_approval_four_eyes CHECK (checker_id IS NULL OR checker_id <> maker_id),
    CONSTRAINT ck_approval_decided
        CHECK (decision = 'PENDING' OR (checker_id IS NOT NULL AND decided_at IS NOT NULL))
);
CREATE INDEX ix_approval_case ON approval_task_ref (case_id);

-- ---------------------------------------------------------------------
-- 11. CASE_EVENT_LOG (append-only, partisi bulanan)
-- ---------------------------------------------------------------------
CREATE TABLE case_event_log (
    event_id     uuid         NOT NULL DEFAULT gen_random_uuid(),
    case_id      varchar(32)  NOT NULL REFERENCES onboarding_case (case_id) ON DELETE RESTRICT,
    event_type   varchar(64)  NOT NULL,
    occurred_at  timestamptz  NOT NULL DEFAULT now(),
    payload_json jsonb        NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (event_id, occurred_at)
) PARTITION BY RANGE (occurred_at);

CREATE TABLE case_event_log_default PARTITION OF case_event_log DEFAULT;
CREATE INDEX ix_event_case ON case_event_log (case_id, occurred_at);

CREATE TRIGGER trg_event_log_append_only
    BEFORE UPDATE OR DELETE ON case_event_log
    FOR EACH ROW EXECUTE FUNCTION fn_append_only();

-- [NEW] Pembuat partisi bulanan (idempoten). Panggil dari job bulanan
-- (pg_cron / scheduler aplikasi) SEBELUM bulan berjalan, atau ganti dengan pg_partman.
CREATE FUNCTION ensure_case_event_log_partition(p_month date) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE
    v_from timestamptz := (date_trunc('month', p_month)::timestamp) AT TIME ZONE 'Asia/Jakarta';
    v_to   timestamptz := ((date_trunc('month', p_month) + interval '1 month')::timestamp) AT TIME ZONE 'Asia/Jakarta';
    v_name text        := 'case_event_log_' || to_char(p_month, 'YYYY_MM');
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS bankservices.%I PARTITION OF bankservices.case_event_log FOR VALUES FROM (%L) TO (%L)',
        v_name, v_from, v_to);
END $$;

-- partisi awal: Oktober 2026 s.d. Maret 2027
SELECT ensure_case_event_log_partition((date '2026-10-01' + make_interval(months => n))::date)
FROM generate_series(0, 5) AS n;

-- ---------------------------------------------------------------------
-- 12. POLA PENDUKUNG (Outbox, Inbox, Idempotency)
-- ---------------------------------------------------------------------
CREATE TABLE outbox_event (
    outbox_id    bigint       GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id     uuid         NOT NULL DEFAULT gen_random_uuid(),   -- [NEW] ID stabil untuk dedup consumer
    aggregate_id varchar(64)  NOT NULL,
    topic        varchar(120) NOT NULL,
    event_key    varchar(64)  NOT NULL,
    payload      jsonb        NOT NULL,
    headers      jsonb        NOT NULL DEFAULT '{}'::jsonb,
    created_at   timestamptz  NOT NULL DEFAULT now(),
    published_at timestamptz,
    CONSTRAINT uq_outbox_event_id UNIQUE (event_id)
);
CREATE INDEX ix_outbox_unpublished ON outbox_event (outbox_id) WHERE published_at IS NULL;
CREATE INDEX ix_outbox_published   ON outbox_event (published_at) WHERE published_at IS NOT NULL;  -- [NEW] untuk purge

COMMENT ON TABLE outbox_event IS
    'Jalankan SATU poller dulu (urutan event per caseId terjaga). Poller banyak instance + SKIP LOCKED dapat menukar urutan; gunakan Debezium bila perlu skala.';

CREATE TABLE inbox_event (
    event_id    varchar(64)  NOT NULL,
    topic       varchar(120) NOT NULL,
    received_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (topic, event_id)
);

CREATE TABLE idempotency_key (
    scope           varchar(256) NOT NULL,        -- [CHG] METHOD + path + sub pemanggil
    idempotency_key uuid         NOT NULL,
    request_hash    char(64)     NOT NULL,        -- sha256 payload; beda hash = ONB-4090
    response_status smallint,                     -- NULL = masih diproses
    response_body   jsonb,
    created_at      timestamptz  NOT NULL DEFAULT now(),
    expires_at      timestamptz  NOT NULL DEFAULT now() + interval '24 hours',
    PRIMARY KEY (scope, idempotency_key)          -- [CHG] key milik satu scope, tidak bocor antar user
);
CREATE INDEX ix_idem_expires ON idempotency_key (expires_at);

-- ---------------------------------------------------------------------
-- 13. TRIGGER updated_at
-- ---------------------------------------------------------------------
CREATE TRIGGER trg_customer_upd  BEFORE UPDATE ON customer              FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
CREATE TRIGGER trg_corporate_upd BEFORE UPDATE ON corporate_entity      FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
CREATE TRIGGER trg_case_upd      BEFORE UPDATE ON onboarding_case       FOR EACH ROW EXECUTE FUNCTION fn_case_touch();
CREATE TRIGGER trg_iv_upd        BEFORE UPDATE ON identity_verification FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
CREATE TRIGGER trg_bo_upd        BEFORE UPDATE ON beneficial_owner      FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
CREATE TRIGGER trg_document_upd  BEFORE UPDATE ON document_ref          FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMIT;

-- ---------------------------------------------------------------------
-- [NEW] Hardening append-only (jalankan setelah role aplikasi ada; sesuaikan nama role)
--   REVOKE UPDATE, DELETE, TRUNCATE ON bankservices.case_event_log FROM onboarding_app;
-- ---------------------------------------------------------------------