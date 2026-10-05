-- Keep formative practice separate from scored mocks so checking a practice
-- answer can never expose the answer key of a band-estimate exam.
CREATE TYPE exam_purpose AS ENUM ('mock', 'practice');

ALTER TABLE exam_versions
  ADD COLUMN purpose exam_purpose NOT NULL DEFAULT 'mock';

CREATE INDEX exam_versions_published_purpose_idx
  ON exam_versions (purpose, skill, published_at DESC)
  WHERE status = 'published';

INSERT INTO exams (id, slug, created_by) VALUES
  ('30000000-0000-4000-8000-000000000001', 'practice-reading-city-bees', '00000000-0000-4000-8000-000000000001'),
  ('40000000-0000-4000-8000-000000000001', 'practice-listening-train-station', '00000000-0000-4000-8000-000000000001');

INSERT INTO exam_versions (id, exam_id, version, title, description, skill, status, duration_seconds, instructions, purpose, published_at) VALUES
  ('31000000-0000-4000-8000-000000000001', '30000000-0000-4000-8000-000000000001', 1, 'City Bees', 'A short untimed Reading drill.', 'reading', 'published', 600, 'Read the source and check each answer as you work.', 'practice', now()),
  ('41000000-0000-4000-8000-000000000001', '40000000-0000-4000-8000-000000000001', 1, 'Station Information', 'A short untimed Listening drill.', 'listening', 'published', 600, 'Listen carefully and check each answer as you work.', 'practice', now());

INSERT INTO exam_sections (id, exam_version_id, position, title, instructions, passage) VALUES
  ('32000000-0000-4000-8000-000000000001', '31000000-0000-4000-8000-000000000001', 1, 'Bees Above the Streets', 'Questions refer to the passage below.', E'Several city hotels now keep beehives on roofs that were previously unused. Urban beekeepers say the insects can find a surprising variety of food because parks, balconies, and roadside trees flower at different times. This long flowering season may provide a steadier supply of nectar than some agricultural areas, where a single crop blooms for only a few weeks.\n\nHives still require careful management. Beekeepers inspect colonies for disease and make sure each hive has enough stored food before winter. Honey is usually collected in late summer, but responsible keepers leave a sufficient amount for the bees. Some hotels use the honey in their kitchens, while others sell small jars and give the income to local environmental projects.'),
  ('42000000-0000-4000-8000-000000000001', '41000000-0000-4000-8000-000000000001', 1, 'Weekend Rail Changes', 'Play the announcement, then answer each question.', E'Attention please. Due to engineering work this Sunday, trains to Westford will leave from platform six instead of platform two. The first service will depart at eight forty, twenty minutes later than usual. Passengers for Lake Park should change at Milltown and use the free replacement bus from the station entrance. Bicycles cannot be carried on replacement buses, but folding bicycles are permitted. The ticket office opens at eight o clock, and staff will be available beside the main information screen from seven thirty. We apologise for the inconvenience.');

INSERT INTO questions (id, section_id, position, type, prompt, explanation) VALUES
  ('33000000-0000-4000-8000-000000000001', '32000000-0000-4000-8000-000000000001', 1, 'multiple_choice', 'Why can city bees have a steady supply of nectar?', 'Different urban plants flower at different times, creating a longer flowering season.'),
  ('33000000-0000-4000-8000-000000000002', '32000000-0000-4000-8000-000000000001', 2, 'true_false_not_given', 'Every city hotel sells its honey to guests.', 'The passage says some hotels use honey in kitchens and others sell it, so the statement about every hotel is false.'),
  ('33000000-0000-4000-8000-000000000003', '32000000-0000-4000-8000-000000000001', 3, 'short_answer', 'When is honey usually collected?', 'The second paragraph states that honey is usually collected in late summer.'),
  ('33000000-0000-4000-8000-000000000004', '32000000-0000-4000-8000-000000000001', 4, 'sentence_completion', 'Before winter, beekeepers check that hives contain enough stored ______.', 'The missing noun is taken directly from the sentence about winter preparation.'),
  ('33000000-0000-4000-8000-000000000005', '32000000-0000-4000-8000-000000000001', 5, 'multiple_choice', 'What may receive money from honey sales?', 'The final sentence says the income may be given to local environmental projects.'),
  ('43000000-0000-4000-8000-000000000001', '42000000-0000-4000-8000-000000000001', 1, 'short_answer', 'Which platform will Westford trains use?', 'The announcement changes the departure point to platform six.'),
  ('43000000-0000-4000-8000-000000000002', '42000000-0000-4000-8000-000000000001', 2, 'short_answer', 'At what time will the first service depart?', 'The first service departure time is stated as eight forty.'),
  ('43000000-0000-4000-8000-000000000003', '42000000-0000-4000-8000-000000000001', 3, 'multiple_choice', 'Where should passengers board the replacement bus?', 'The bus leaves from the station entrance.'),
  ('43000000-0000-4000-8000-000000000004', '42000000-0000-4000-8000-000000000001', 4, 'true_false_not_given', 'Folding bicycles are allowed on replacement buses.', 'The announcement explicitly says folding bicycles are permitted.'),
  ('43000000-0000-4000-8000-000000000005', '42000000-0000-4000-8000-000000000001', 5, 'short_answer', 'When will staff be available beside the information screen?', 'Staff availability begins at seven thirty.');

INSERT INTO question_options (question_id, position, label, value) VALUES
  ('33000000-0000-4000-8000-000000000001', 1, 'A', 'The city has only one crop'),
  ('33000000-0000-4000-8000-000000000001', 2, 'B', 'Plants flower at different times'),
  ('33000000-0000-4000-8000-000000000001', 3, 'C', 'Hotels feed every colony'),
  ('33000000-0000-4000-8000-000000000002', 1, 'T', 'True'),
  ('33000000-0000-4000-8000-000000000002', 2, 'F', 'False'),
  ('33000000-0000-4000-8000-000000000002', 3, 'NG', 'Not Given'),
  ('33000000-0000-4000-8000-000000000005', 1, 'A', 'Agricultural companies'),
  ('33000000-0000-4000-8000-000000000005', 2, 'B', 'Environmental projects'),
  ('33000000-0000-4000-8000-000000000005', 3, 'C', 'Road construction'),
  ('43000000-0000-4000-8000-000000000003', 1, 'A', 'Platform two'),
  ('43000000-0000-4000-8000-000000000003', 2, 'B', 'The station entrance'),
  ('43000000-0000-4000-8000-000000000003', 3, 'C', 'The ticket office'),
  ('43000000-0000-4000-8000-000000000004', 1, 'T', 'True'),
  ('43000000-0000-4000-8000-000000000004', 2, 'F', 'False'),
  ('43000000-0000-4000-8000-000000000004', 3, 'NG', 'Not Given');

INSERT INTO accepted_answers (question_id, value, position) VALUES
  ('33000000-0000-4000-8000-000000000001', 'Plants flower at different times', 1),
  ('33000000-0000-4000-8000-000000000002', 'False', 1),
  ('33000000-0000-4000-8000-000000000003', 'late summer', 1),
  ('33000000-0000-4000-8000-000000000004', 'food', 1),
  ('33000000-0000-4000-8000-000000000005', 'Environmental projects', 1),
  ('43000000-0000-4000-8000-000000000001', 'platform six', 1),
  ('43000000-0000-4000-8000-000000000001', '6', 2),
  ('43000000-0000-4000-8000-000000000002', '8:40', 1),
  ('43000000-0000-4000-8000-000000000002', 'eight forty', 2),
  ('43000000-0000-4000-8000-000000000003', 'The station entrance', 1),
  ('43000000-0000-4000-8000-000000000004', 'True', 1),
  ('43000000-0000-4000-8000-000000000005', '7:30', 1),
  ('43000000-0000-4000-8000-000000000005', 'seven thirty', 2);
