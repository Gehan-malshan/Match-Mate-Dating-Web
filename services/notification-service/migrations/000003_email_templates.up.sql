-- Email uses the same reviewed, privacy-safe wording as the in-app development
-- templates. Channel-specific rows preserve version history and preferences.
INSERT INTO notification_template(id,template_key,locale,channel,category,version,status,subject_template,body_template,allowed_variables,created_at)
SELECT CASE template_key
    WHEN 'account-welcome' THEN '72000000-0000-4000-8000-000000000001'::uuid
    WHEN 'account-verified' THEN '72000000-0000-4000-8000-000000000002'::uuid
    WHEN 'profile-approved' THEN '72000000-0000-4000-8000-000000000003'::uuid
    WHEN 'profile-hidden' THEN '72000000-0000-4000-8000-000000000004'::uuid
    WHEN 'booking-pending' THEN '72000000-0000-4000-8000-000000000005'::uuid
    WHEN 'booking-confirmed' THEN '72000000-0000-4000-8000-000000000006'::uuid
    WHEN 'booking-cancelled' THEN '72000000-0000-4000-8000-000000000007'::uuid
    WHEN 'booking-hold-expired' THEN '72000000-0000-4000-8000-000000000008'::uuid
    WHEN 'booking-payment-review' THEN '72000000-0000-4000-8000-000000000009'::uuid
END, template_key,locale,'EMAIL',category,version,status,subject_template,body_template,allowed_variables,created_at
FROM notification_template
WHERE channel='DEVELOPMENT'
  AND template_key IN ('account-welcome','account-verified','profile-approved','profile-hidden','booking-pending','booking-confirmed','booking-cancelled','booking-hold-expired','booking-payment-review')
ON CONFLICT(template_key,locale,channel,version) DO NOTHING;
