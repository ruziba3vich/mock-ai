-- name: CreateAdmin :one
INSERT INTO admins(email,display_name,password_hash) VALUES($1,$2,$3) RETURNING id;

-- name: ReserveAdminLogin :one
UPDATE admins SET
 login_attempts=CASE WHEN login_window<now()-interval '15 minutes' THEN 1 ELSE login_attempts+1 END,
 login_window=CASE WHEN login_window<now()-interval '15 minutes' THEN now() ELSE login_window END
WHERE email=$1 AND NOT disabled AND (login_attempts<5 OR login_window<now()-interval '15 minutes')
RETURNING id,email,display_name,password_hash;

-- name: ResetAdminLogin :exec
UPDATE admins SET login_attempts=0,login_window=now() WHERE id=$1;

-- name: CreateAdminSession :exec
INSERT INTO admin_sessions(token_hash,admin_id,expires_at) VALUES($1,$2,$3);

-- name: GetAdminSession :one
SELECT a.id,a.email,a.display_name FROM admins a JOIN admin_sessions s ON s.admin_id=a.id
WHERE s.token_hash=$1 AND s.expires_at>now() AND NOT a.disabled;

-- name: DeleteAdminSession :exec
DELETE FROM admin_sessions WHERE token_hash=$1;
