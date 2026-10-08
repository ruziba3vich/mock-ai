DELETE FROM private_media WHERE mime_type = 'image/png';
ALTER TABLE private_media DROP CONSTRAINT private_media_mime_type_check;
ALTER TABLE private_media
ADD CONSTRAINT private_media_mime_type_check
CHECK (mime_type = 'audio/mpeg');
ALTER TABLE exam_sections DROP COLUMN source_image_urls;
