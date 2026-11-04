-- Migration 007: Dosen Pembimbing TA
CREATE TABLE IF NOT EXISTS supervisors (
    id               SERIAL       PRIMARY KEY,
    name             VARCHAR(120) NOT NULL,
    quota            INTEGER      NOT NULL CHECK (quota > 0),
    supervised_count INTEGER      NOT NULL DEFAULT 0,
    version          INTEGER      NOT NULL DEFAULT 1,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT supervisors_supervised_within_quota
        CHECK (supervised_count >= 0 AND supervised_count <= quota)
);

CREATE TABLE IF NOT EXISTS supervisions (
    student_id    INTEGER     PRIMARY KEY REFERENCES students(id) ON DELETE CASCADE,
    supervisor_id INTEGER     NOT NULL REFERENCES supervisors(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed data dosen pembimbing untuk pengujian
INSERT INTO supervisors (name, quota) VALUES
    ('Dr. Lu Guang', 3),
    ('Dr. Cheng Xiaoshi',  3),
    ('Prof. Qiao Ling', 3)
ON CONFLICT DO NOTHING;

-- Permission baru untuk admin
INSERT INTO permissions (name, description) VALUES
    ('supervision:manage', 'Mengelola penetapan dosen pembimbing')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'supervision:manage')
ON CONFLICT DO NOTHING;