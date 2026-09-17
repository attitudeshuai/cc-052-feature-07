BEGIN;

-- Farms / Cooperatives
CREATE TABLE IF NOT EXISTS farm (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    region_code VARCHAR(16) NOT NULL,
    contact_ref VARCHAR(128),
    cert_no VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Plots
CREATE TABLE IF NOT EXISTS plot (
    id BIGSERIAL PRIMARY KEY,
    farm_id BIGINT NOT NULL REFERENCES farm(id),
    name VARCHAR(255) NOT NULL,
    area_mu DECIMAL(10,2) NOT NULL,
    geojson TEXT,
    soil_type VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Crop batches
CREATE TABLE IF NOT EXISTS crop_batch (
    id BIGSERIAL PRIMARY KEY,
    plot_id BIGINT NOT NULL REFERENCES plot(id),
    crop_id VARCHAR(64) NOT NULL,
    sowing_date DATE NOT NULL,
    harvest_date DATE,
    expected_yield_kg DECIMAL(10,2) NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'growing' CHECK (status IN ('growing','harvested','locked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Input materials dictionary (must be before activity due to FK)
CREATE TABLE IF NOT EXISTS input_material (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'pesticide',
    registration_no VARCHAR(64),
    safe_interval_days INT NOT NULL DEFAULT 0,
    active_ingredient VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Farm activities (fertilize, pesticide, irrigation, weed)
CREATE TABLE IF NOT EXISTS activity (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES crop_batch(id),
    client_uuid VARCHAR(64) NOT NULL,
    kind VARCHAR(16) NOT NULL CHECK (kind IN ('fertilize','pesticide','irrigation','weed')),
    happened_at TIMESTAMPTZ NOT NULL,
    input_id BIGINT REFERENCES input_material(id),
    dose DECIMAL(12,4),
    dose_unit VARCHAR(16),
    operator VARCHAR(128) NOT NULL,
    photos JSONB DEFAULT '[]',
    geo VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique index for offline idempotent batch reporting
CREATE UNIQUE INDEX IF NOT EXISTS idx_activity_client_uuid ON activity(client_uuid);

-- Inspection records
CREATE TABLE IF NOT EXISTS inspection (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES crop_batch(id),
    lab VARCHAR(255) NOT NULL,
    sampled_at TIMESTAMPTZ NOT NULL,
    result VARCHAR(8) NOT NULL CHECK (result IN ('pass','fail')),
    report_url TEXT,
    items JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Trace codes
CREATE TABLE IF NOT EXISTS trace_code (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES crop_batch(id),
    code VARCHAR(16) NOT NULL,
    seq INT NOT NULL DEFAULT 0,
    printed_at TIMESTAMPTZ,
    first_scanned_at TIMESTAMPTZ,
    first_scan_region VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_trace_code_code ON trace_code(code);

-- Indexes for trace queries
CREATE INDEX IF NOT EXISTS idx_trace_code_batch ON trace_code(batch_id);
CREATE INDEX IF NOT EXISTS idx_activity_batch ON activity(batch_id);
CREATE INDEX IF NOT EXISTS idx_inspection_batch ON inspection(batch_id);
CREATE INDEX IF NOT EXISTS idx_crop_batch_plot ON crop_batch(plot_id);
CREATE INDEX IF NOT EXISTS idx_plot_farm ON plot(farm_id);

COMMIT;