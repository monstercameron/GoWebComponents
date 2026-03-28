DELETE FROM user_auth_blocks
WHERE user_id = ?
  AND block_key = ?;
