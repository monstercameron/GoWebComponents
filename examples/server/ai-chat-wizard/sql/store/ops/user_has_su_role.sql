SELECT EXISTS (
    SELECT 1
    FROM su_user_roles ur
    JOIN su_roles r ON r.role_key = ur.role_key
    WHERE ur.user_id = ?
      AND ur.role_key = 'su'
      AND r.is_enabled <> 0
);
