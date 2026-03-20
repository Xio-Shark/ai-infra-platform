ALTER TABLE jobs ADD COLUMN model_version TEXT;
ALTER TABLE jobs ADD COLUMN dataset_version TEXT;
ALTER TABLE jobs ADD COLUMN image_tag TEXT;
ALTER TABLE jobs ADD COLUMN resource_spec TEXT;
