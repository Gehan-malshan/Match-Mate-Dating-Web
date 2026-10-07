ALTER TABLE event ADD COLUMN payment_options text NOT NULL DEFAULT 'ONLINE' CHECK (payment_options IN ('ONLINE','AT_VENUE','BOTH'));
