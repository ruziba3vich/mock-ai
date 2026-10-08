ALTER TABLE exam_sections
ADD COLUMN source_image_urls jsonb NOT NULL DEFAULT '[]'::jsonb
CHECK (jsonb_typeof(source_image_urls) = 'array');

ALTER TABLE private_media DROP CONSTRAINT private_media_mime_type_check;
ALTER TABLE private_media
ADD CONSTRAINT private_media_mime_type_check
CHECK (mime_type IN ('audio/mpeg', 'image/png'));
