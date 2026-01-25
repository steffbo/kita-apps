ALTER TABLE schedule_entries
ALTER COLUMN group_id DROP NOT NULL;

UPDATE schedule_entries
SET group_id = NULL
WHERE group_id = (SELECT id FROM groups WHERE name = 'Springer');

DELETE FROM group_assignments
WHERE group_id = (SELECT id FROM groups WHERE name = 'Springer');

DELETE FROM groups WHERE name = 'Springer';
