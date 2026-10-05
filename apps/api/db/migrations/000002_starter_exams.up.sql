-- Starter content makes a fresh installation usable before the admin authoring
-- interface is available. Fixed identifiers keep the migration deterministic.
INSERT INTO users (id, display_name, email, email_verified_at)
VALUES ('00000000-0000-4000-8000-000000000001', 'Mock AI Content Team', 'content@mock-ai.internal', now());

INSERT INTO user_role_assignments (user_id, role)
VALUES ('00000000-0000-4000-8000-000000000001', 'admin');

INSERT INTO exams (id, slug, created_by) VALUES
  ('10000000-0000-4000-8000-000000000001', 'reading-urban-gardens', '00000000-0000-4000-8000-000000000001'),
  ('20000000-0000-4000-8000-000000000001', 'listening-community-library', '00000000-0000-4000-8000-000000000001');

INSERT INTO exam_versions (id, exam_id, version, title, description, skill, status, duration_seconds, instructions, published_at) VALUES
  ('11000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001', 1, 'Urban Gardens', 'Read two short passages about urban food production and answer questions on detail, meaning, and inference.', 'reading', 'published', 1200, 'Read each passage carefully. You may move between questions at any time. Spelling counts for typed answers.', now()),
  ('21000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001', 1, 'Community Library', 'Listen to two everyday conversations and identify key facts, details, and speaker intentions.', 'listening', 'published', 900, 'Use headphones if possible. Play each section when you are ready and answer while listening. Spelling counts for typed answers.', now());

INSERT INTO exam_sections (id, exam_version_id, position, title, instructions, passage) VALUES
  ('12000000-0000-4000-8000-000000000001', '11000000-0000-4000-8000-000000000001', 1, 'Gardens Above the City', 'Questions 1–5 refer to the passage below.', E'As cities become denser, unused rooftops are increasingly being turned into productive gardens. These spaces can supply herbs and vegetables to nearby residents, but their value is not limited to food. Soil and plants absorb rainwater that might otherwise overwhelm drainage systems during storms. They also shade roof surfaces and can reduce the amount of energy needed to cool the floors below.\n\nRooftop growing is not without challenges. Engineers must first confirm that a building can carry the additional weight of soil, water, and people. Gardeners also need to choose crops that tolerate stronger winds and more intense sunlight than plants at street level. Although installation can be expensive, several building owners report lower energy costs over time.'),
  ('12000000-0000-4000-8000-000000000002', '11000000-0000-4000-8000-000000000001', 2, 'Small Plots, Strong Communities', 'Questions 6–10 refer to the passage below.', E'Community gardens are often measured by the quantity of food they produce, yet researchers argue that their social effects may be equally important. Shared plots give neighbours a reason to meet regularly and exchange practical knowledge. New residents can form local connections, while experienced growers pass techniques to younger volunteers.\n\nA three-year study in Northbridge found that participation remained highest in gardens where members shared responsibility for ordinary tasks. Sites that depended on one enthusiastic organiser were more likely to decline when that person moved away. The researchers therefore recommend written schedules, short training sessions, and regular open meetings. They also warn that gardens should not be presented as a complete solution to food insecurity; affordable shops and reliable transport remain essential.'),
  ('22000000-0000-4000-8000-000000000001', '21000000-0000-4000-8000-000000000001', 1, 'Joining the Library', 'Questions 1–5 refer to this conversation. Press play when ready.', E'Good morning. I would like to join the library. Certainly. Do you live in the borough? Yes, I moved to King Street last month. Great. Please bring one form of photo identification and a recent bill showing your address. Is there a membership fee? Standard membership is free. We only charge two pounds if you need a replacement card. How many books can I borrow? Up to twelve at a time, for three weeks. You can renew them online unless another reader has reserved them. I am also interested in the study rooms. Those can be booked for two hours per day. Bookings open seven days in advance.'),
  ('22000000-0000-4000-8000-000000000002', '21000000-0000-4000-8000-000000000001', 2, 'Weekend Workshop', 'Questions 6–10 refer to this announcement. Press play when ready.', E'This Saturday the library is holding a practical workshop on digital photography. It begins at ten fifteen in the media room on the second floor and finishes at twelve thirty. The session is intended for beginners, so no previous experience is required. Participants should bring a camera or a phone with a working camera. The tutor, Maya Chen, will explain composition and natural lighting before leading a short practice walk in the garden. The workshop costs five pounds, including printed notes and a hot drink. Places are limited to sixteen, and registration closes at six o clock on Thursday evening.');

INSERT INTO questions (id, section_id, position, type, prompt) VALUES
  ('13000000-0000-4000-8000-000000000001', '12000000-0000-4000-8000-000000000001', 1, 'multiple_choice', 'What is one benefit of rooftop gardens during storms?'),
  ('13000000-0000-4000-8000-000000000002', '12000000-0000-4000-8000-000000000001', 2, 'true_false_not_given', 'Rooftop gardens always produce enough food for every resident in a building.'),
  ('13000000-0000-4000-8000-000000000003', '12000000-0000-4000-8000-000000000001', 3, 'short_answer', 'Who must confirm that a roof can support the garden?'),
  ('13000000-0000-4000-8000-000000000004', '12000000-0000-4000-8000-000000000001', 4, 'multiple_choice', 'Which conditions may be stronger at roof level?'),
  ('13000000-0000-4000-8000-000000000005', '12000000-0000-4000-8000-000000000001', 5, 'sentence_completion', 'Some owners recover installation costs through lower ______ costs.'),
  ('13000000-0000-4000-8000-000000000006', '12000000-0000-4000-8000-000000000002', 6, 'multiple_choice', 'What benefit do shared plots offer new residents?'),
  ('13000000-0000-4000-8000-000000000007', '12000000-0000-4000-8000-000000000002', 7, 'true_false_not_given', 'The Northbridge study lasted for three years.'),
  ('13000000-0000-4000-8000-000000000008', '12000000-0000-4000-8000-000000000002', 8, 'short_answer', 'Who may pass growing techniques to younger volunteers?'),
  ('13000000-0000-4000-8000-000000000009', '12000000-0000-4000-8000-000000000002', 9, 'multiple_select', 'Which TWO practices do the researchers recommend?'),
  ('13000000-0000-4000-8000-000000000010', '12000000-0000-4000-8000-000000000002', 10, 'yes_no_not_given', 'The researchers believe community gardens can replace affordable shops.'),
  ('23000000-0000-4000-8000-000000000001', '22000000-0000-4000-8000-000000000001', 1, 'short_answer', 'Which street does the new member live on?'),
  ('23000000-0000-4000-8000-000000000002', '22000000-0000-4000-8000-000000000001', 2, 'multiple_select', 'Which TWO documents must the member bring?'),
  ('23000000-0000-4000-8000-000000000003', '22000000-0000-4000-8000-000000000001', 3, 'short_answer', 'How much does a replacement card cost?'),
  ('23000000-0000-4000-8000-000000000004', '22000000-0000-4000-8000-000000000001', 4, 'multiple_choice', 'For how long can books normally be borrowed?'),
  ('23000000-0000-4000-8000-000000000005', '22000000-0000-4000-8000-000000000001', 5, 'short_answer', 'How many hours per day may a study room be booked?'),
  ('23000000-0000-4000-8000-000000000006', '22000000-0000-4000-8000-000000000002', 6, 'short_answer', 'At what time does the workshop begin?'),
  ('23000000-0000-4000-8000-000000000007', '22000000-0000-4000-8000-000000000002', 7, 'multiple_choice', 'Where will the workshop take place?'),
  ('23000000-0000-4000-8000-000000000008', '22000000-0000-4000-8000-000000000002', 8, 'true_false_not_given', 'Participants need previous photography experience.'),
  ('23000000-0000-4000-8000-000000000009', '22000000-0000-4000-8000-000000000002', 9, 'short_answer', 'What is the tutor’s surname?'),
  ('23000000-0000-4000-8000-000000000010', '22000000-0000-4000-8000-000000000002', 10, 'short_answer', 'How many places are available?');

INSERT INTO question_options (question_id, position, label, value) VALUES
  ('13000000-0000-4000-8000-000000000001', 1, 'A', 'They increase wind speed'), ('13000000-0000-4000-8000-000000000001', 2, 'B', 'They absorb rainwater'), ('13000000-0000-4000-8000-000000000001', 3, 'C', 'They replace drainage systems'),
  ('13000000-0000-4000-8000-000000000002', 1, 'T', 'True'), ('13000000-0000-4000-8000-000000000002', 2, 'F', 'False'), ('13000000-0000-4000-8000-000000000002', 3, 'NG', 'Not Given'),
  ('13000000-0000-4000-8000-000000000004', 1, 'A', 'Wind and sunlight'), ('13000000-0000-4000-8000-000000000004', 2, 'B', 'Rain and shade'), ('13000000-0000-4000-8000-000000000004', 3, 'C', 'Cold and pollution'),
  ('13000000-0000-4000-8000-000000000006', 1, 'A', 'Free transport'), ('13000000-0000-4000-8000-000000000006', 2, 'B', 'Local connections'), ('13000000-0000-4000-8000-000000000006', 3, 'C', 'Paid employment'),
  ('13000000-0000-4000-8000-000000000007', 1, 'T', 'True'), ('13000000-0000-4000-8000-000000000007', 2, 'F', 'False'), ('13000000-0000-4000-8000-000000000007', 3, 'NG', 'Not Given'),
  ('13000000-0000-4000-8000-000000000009', 1, 'A', 'Written schedules'), ('13000000-0000-4000-8000-000000000009', 2, 'B', 'Private meetings'), ('13000000-0000-4000-8000-000000000009', 3, 'C', 'Short training sessions'), ('13000000-0000-4000-8000-000000000009', 4, 'D', 'One permanent organiser'),
  ('13000000-0000-4000-8000-000000000010', 1, 'Y', 'Yes'), ('13000000-0000-4000-8000-000000000010', 2, 'N', 'No'), ('13000000-0000-4000-8000-000000000010', 3, 'NG', 'Not Given'),
  ('23000000-0000-4000-8000-000000000002', 1, 'A', 'Photo identification'), ('23000000-0000-4000-8000-000000000002', 2, 'B', 'A recent bill'), ('23000000-0000-4000-8000-000000000002', 3, 'C', 'A bank card'), ('23000000-0000-4000-8000-000000000002', 4, 'D', 'A passport photo'),
  ('23000000-0000-4000-8000-000000000004', 1, 'A', 'Two weeks'), ('23000000-0000-4000-8000-000000000004', 2, 'B', 'Three weeks'), ('23000000-0000-4000-8000-000000000004', 3, 'C', 'Four weeks'),
  ('23000000-0000-4000-8000-000000000007', 1, 'A', 'The entrance hall'), ('23000000-0000-4000-8000-000000000007', 2, 'B', 'The media room'), ('23000000-0000-4000-8000-000000000007', 3, 'C', 'The garden'),
  ('23000000-0000-4000-8000-000000000008', 1, 'T', 'True'), ('23000000-0000-4000-8000-000000000008', 2, 'F', 'False'), ('23000000-0000-4000-8000-000000000008', 3, 'NG', 'Not Given');

INSERT INTO accepted_answers (question_id, value, position) VALUES
  ('13000000-0000-4000-8000-000000000001', 'They absorb rainwater', 1), ('13000000-0000-4000-8000-000000000002', 'Not Given', 1),
  ('13000000-0000-4000-8000-000000000003', 'engineers', 1), ('13000000-0000-4000-8000-000000000004', 'Wind and sunlight', 1),
  ('13000000-0000-4000-8000-000000000005', 'energy', 1), ('13000000-0000-4000-8000-000000000006', 'Local connections', 1),
  ('13000000-0000-4000-8000-000000000007', 'True', 1), ('13000000-0000-4000-8000-000000000008', 'experienced growers', 1),
  ('13000000-0000-4000-8000-000000000009', 'Written schedules', 1), ('13000000-0000-4000-8000-000000000009', 'Short training sessions', 2),
  ('13000000-0000-4000-8000-000000000010', 'No', 1),
  ('23000000-0000-4000-8000-000000000001', 'King Street', 1),
  ('23000000-0000-4000-8000-000000000002', 'Photo identification', 1), ('23000000-0000-4000-8000-000000000002', 'A recent bill', 2),
  ('23000000-0000-4000-8000-000000000003', 'two pounds', 1), ('23000000-0000-4000-8000-000000000003', '£2', 2),
  ('23000000-0000-4000-8000-000000000004', 'Three weeks', 1), ('23000000-0000-4000-8000-000000000005', 'two', 1), ('23000000-0000-4000-8000-000000000005', '2', 2),
  ('23000000-0000-4000-8000-000000000006', '10:15', 1), ('23000000-0000-4000-8000-000000000006', 'ten fifteen', 2),
  ('23000000-0000-4000-8000-000000000007', 'The media room', 1), ('23000000-0000-4000-8000-000000000008', 'False', 1),
  ('23000000-0000-4000-8000-000000000009', 'Chen', 1), ('23000000-0000-4000-8000-000000000010', 'sixteen', 1), ('23000000-0000-4000-8000-000000000010', '16', 2);

INSERT INTO band_conversions (exam_version_id, minimum_raw_score, band)
SELECT version_id, minimum_raw_score, band
FROM (VALUES
  ('11000000-0000-4000-8000-000000000001'::uuid),
  ('21000000-0000-4000-8000-000000000001'::uuid)
) AS versions(version_id)
CROSS JOIN (VALUES
  (0::numeric, 3.0::numeric), (2, 4.0), (4, 5.0), (5, 5.5), (6, 6.0),
  (7, 6.5), (8, 7.0), (9, 8.0), (10, 9.0)
) AS bands(minimum_raw_score, band);
