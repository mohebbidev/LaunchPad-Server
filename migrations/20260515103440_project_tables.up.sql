
DROP TABLE IF EXISTS project_settings;

DROP TABLE IF EXISTS projects;

DROP TABLE IF EXISTS users;

-- Create Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create Projects Table
CREATE TABLE projects (
    -- id BIGSERIAL PRIMARY KEY,
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    unique_key VARCHAR(64) UNIQUE NOT NULL, -- Shorter, user-friendly key for URLs
    source_type VARCHAR(50) NOT NULL DEFAULT 'zip', -- e.g., 'zip', 'git_repo'
    source_location VARCHAR(512) NOT NULL, -- Path to zip, Git URL, etc.
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- e.g., 'pending', 'building', 'running', 'failed', 'stopped'
    port INTEGER NULL, -- The port the project is running on (NULL if not running)
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deployed_at TIMESTAMP WITH TIME ZONE NULL, -- When it was last successfully deployed


    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE -- If user is deleted, delete their projects too
);


CREATE UNIQUE INDEX idx_projects_unique_key ON projects (unique_key);

CREATE INDEX idx_projects_user_id ON projects (user_id);

CREATE INDEX idx_projects_status ON projects (status);



CREATE TABLE project_settings (
    project_id UUID PRIMARY KEY, -- Corresponds to projects.id
    settings JSONB NOT NULL DEFAULT '{}'::jsonb, -- Flexible key-value settings
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE -- If project is deleted, delete its settings
);

-- GIN INDEX
CREATE INDEX idx_project_settings_settings ON project_settings USING GIN (settings);



-- UPDATE TRUGGERS
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_projects_updated_at
BEFORE UPDATE ON projects
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Trigger for project_settings table
CREATE TRIGGER update_project_settings_updated_at BEFORE
UPDATE ON project_settings FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column ();