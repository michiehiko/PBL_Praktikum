-- Mendaftarkan permission baru ke tabel permissions
INSERT INTO permissions (name, description) VALUES
    ('student:list', 'Melihat daftar seluruh student'),
    ('student:read:any', 'Melihat data student mana pun'),
    ('student:create', 'Menambahkan data student baru'),
    ('student:update:any', 'Mengubah data student mana pun'),
    ('student:delete', 'Menghapus data student')
ON CONFLICT (name) DO NOTHING;

-- Memetakan permission tersebut ke role admin dan staff
INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'student:list'),
    ('admin', 'student:read:any'),
    ('admin', 'student:create'),
    ('admin', 'student:update:any'),
    ('admin', 'student:delete'),
    ('staff', 'student:list'),
    ('staff', 'student:read:any'),
    ('staff', 'student:create')
ON CONFLICT DO NOTHING;

-- Menambahkan owner_id pada tabel students
ALTER TABLE students ADD COLUMN IF NOT EXISTS owner_id INTEGER;

-- Menambahkan constraint FOREIGN KEY
ALTER TABLE students DROP CONSTRAINT IF EXISTS students_owner_id_fkey;
ALTER TABLE students
    ADD CONSTRAINT students_owner_id_fkey
    FOREIGN KEY (owner_id) REFERENCES users (id) 
    ON UPDATE CASCADE 
    ON DELETE SET NULL; 

    UPDATE students SET owner_id = 1 WHERE owner_id IS NULL;