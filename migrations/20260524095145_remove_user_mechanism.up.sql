ALTER TABLE projects DROP COLUMN user_id;

DROP INDEX IF EXISTS idx_projects_user_id;

DROP TABLE users;