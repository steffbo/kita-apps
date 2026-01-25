-- Create Springer group and migrate data

-- 1. Create the Springer group
INSERT INTO groups (name, description, color) VALUES
    ('Springer', 'Flexible Mitarbeiter ohne feste Gruppe', '#64748B');

-- 2. Assign Springer group to employees without a PERMANENT group
INSERT INTO group_assignments (employee_id, group_id, assignment_type)
SELECT
    e.id,
    (SELECT id FROM groups WHERE name = 'Springer'),
    'PERMANENT'
FROM employees e
WHERE e.id NOT IN (
    SELECT employee_id
    FROM group_assignments
    WHERE assignment_type = 'PERMANENT'
);

-- 3. Update schedule entries with NULL group_id to Springer
UPDATE schedule_entries
SET group_id = (SELECT id FROM groups WHERE name = 'Springer')
WHERE group_id IS NULL;

-- 4. Make group_id NOT NULL in schedule_entries
ALTER TABLE schedule_entries
ALTER COLUMN group_id SET NOT NULL;
