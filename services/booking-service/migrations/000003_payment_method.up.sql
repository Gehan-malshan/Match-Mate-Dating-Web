ALTER TABLE booking ADD COLUMN payment_method text NOT NULL DEFAULT 'ONLINE' CHECK (payment_method IN ('ONLINE','AT_VENUE'));
