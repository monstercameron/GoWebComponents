SELECT COUNT(1)
FROM workspace_memberships wm
JOIN workspaces w ON w.id = wm.workspace_id
WHERE wm.user_id = ?
  AND LOWER(COALESCE(wm.status, 'active')) = 'active'
  AND LOWER(COALESCE(w.status, 'active')) = 'active';
