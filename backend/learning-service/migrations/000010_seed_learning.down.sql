-- Revert the learning-service seed data. Only deletes the seeded ids; rows
-- created at runtime are preserved.
BEGIN;

DELETE FROM learning_questions WHERE id BETWEEN 7301 AND 7315;
DELETE FROM learning_exams      WHERE id BETWEEN 7201 AND 7203;
DELETE FROM learning_lessons    WHERE id BETWEEN 7101 AND 7112;
DELETE FROM learning_courses    WHERE id BETWEEN 7001 AND 7003;

COMMIT;
