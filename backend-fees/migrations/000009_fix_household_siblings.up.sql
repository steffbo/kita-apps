-- Fix household assignments: ensure siblings share the same household
-- The original migration created separate households per parent instead of per family

-- Step 1: For children that share parents, ensure they share the same household
-- Find groups of children that have at least one parent in common (siblings)
-- and consolidate them into a single household

-- Create a temp table to identify sibling groups
CREATE TEMP TABLE sibling_groups AS
WITH parent_children AS (
    -- Get all child-parent relationships
    SELECT cp.child_id, cp.parent_id, c.household_id as child_household_id
    FROM fees.child_parents cp
    JOIN fees.children c ON c.id = cp.child_id
),
sibling_pairs AS (
    -- Find pairs of children that share at least one parent (siblings)
    SELECT DISTINCT 
        pc1.child_id as child1_id,
        pc2.child_id as child2_id,
        pc1.child_household_id as household1_id,
        pc2.child_household_id as household2_id
    FROM parent_children pc1
    JOIN parent_children pc2 ON pc1.parent_id = pc2.parent_id 
        AND pc1.child_id < pc2.child_id
)
SELECT * FROM sibling_pairs
WHERE household1_id IS DISTINCT FROM household2_id;

-- Step 2: For each sibling pair with different households, move child2 to child1's household
-- (We pick the "older" household, i.e., lower ID or the one that was created first)
UPDATE fees.children c
SET household_id = sg.household1_id
FROM sibling_groups sg
WHERE c.id = sg.child2_id
AND c.household_id = sg.household2_id;

-- Step 3: Update parents to be in the same household as their children
-- For each parent, set their household to match one of their children's households
WITH parent_child_households AS (
    SELECT DISTINCT ON (cp.parent_id)
        cp.parent_id,
        c.household_id
    FROM fees.child_parents cp
    JOIN fees.children c ON c.id = cp.child_id
    WHERE c.household_id IS NOT NULL
    ORDER BY cp.parent_id, cp.is_primary DESC, c.created_at
)
UPDATE fees.parents p
SET household_id = pch.household_id
FROM parent_child_households pch
WHERE p.id = pch.parent_id
AND (p.household_id IS NULL OR p.household_id != pch.household_id);

-- Step 4: Clean up orphaned households (households with no children or parents)
DELETE FROM fees.households h
WHERE NOT EXISTS (
    SELECT 1 FROM fees.children c WHERE c.household_id = h.id
)
AND NOT EXISTS (
    SELECT 1 FROM fees.parents p WHERE p.household_id = h.id
);

-- Drop temp table
DROP TABLE IF EXISTS sibling_groups;
