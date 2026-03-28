UPDATE email_verification_tokens
SET
    status = 'verified',
    verified_at = ?
WHERE id = ?
  AND status = 'pending';
