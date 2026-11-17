-- Modul 11 D.2: satu event kini boleh menghasilkan banyak notifikasi.
-- satu event boleh punya satu notifikasi PER PENERIMA, tetapi tidak boleh dua untuk penerima yang sama.

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_outbox_unique;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id);