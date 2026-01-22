-- Create households table
CREATE TABLE fees.households (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    annual_household_income DECIMAL(12,2),
    income_status VARCHAR(50) DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add trigger for updated_at
CREATE TRIGGER update_households_updated_at
    BEFORE UPDATE ON fees.households
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();

-- Create members table (Vereinsmitglieder)
CREATE TABLE fees.members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_number VARCHAR(20) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    street VARCHAR(200),
    street_no VARCHAR(20),
    postal_code VARCHAR(10),
    city VARCHAR(100),
    household_id UUID REFERENCES fees.households(id),
    membership_start DATE NOT NULL,
    membership_end DATE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add trigger for updated_at on members
CREATE TRIGGER update_members_updated_at
    BEFORE UPDATE ON fees.members
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();

-- Add household_id to children
ALTER TABLE fees.children ADD COLUMN household_id UUID REFERENCES fees.households(id);

-- Add household_id and member_id to parents
ALTER TABLE fees.parents ADD COLUMN household_id UUID REFERENCES fees.households(id);
ALTER TABLE fees.parents ADD COLUMN member_id UUID REFERENCES fees.members(id);

-- Create indexes
CREATE INDEX idx_households_name ON fees.households(name);
CREATE INDEX idx_members_member_number ON fees.members(member_number);
CREATE INDEX idx_members_last_name ON fees.members(last_name);
CREATE INDEX idx_members_is_active ON fees.members(is_active);
CREATE INDEX idx_members_household_id ON fees.members(household_id);
CREATE INDEX idx_children_household_id ON fees.children(household_id);
CREATE INDEX idx_parents_household_id ON fees.parents(household_id);
CREATE INDEX idx_parents_member_id ON fees.parents(member_id);

-- Data migration: Create households for existing families and migrate income data
-- This creates one household per unique set of parents linked to children

-- Step 1: Create households based on the first parent of each child (primary or first linked)
-- Group children by their parent combinations to identify families
INSERT INTO fees.households (id, name, annual_household_income, income_status)
SELECT DISTINCT ON (p.id)
    gen_random_uuid(),
    'Familie ' || p.last_name,
    p.annual_household_income,
    p.income_status
FROM fees.parents p
INNER JOIN fees.child_parents cp ON cp.parent_id = p.id
ORDER BY p.id;

-- Step 2: Link children to households via their parents
-- For each child, find their first parent and use that parent's household
WITH parent_households AS (
    SELECT DISTINCT ON (p.id)
        p.id as parent_id,
        h.id as household_id
    FROM fees.parents p
    INNER JOIN fees.households h ON h.name = 'Familie ' || p.last_name
        AND (h.annual_household_income = p.annual_household_income OR (h.annual_household_income IS NULL AND p.annual_household_income IS NULL))
        AND h.income_status = p.income_status
    ORDER BY p.id
)
UPDATE fees.parents p
SET household_id = ph.household_id
FROM parent_households ph
WHERE p.id = ph.parent_id;

-- Step 3: Link children to same household as their primary parent (or first parent if no primary)
WITH child_household AS (
    SELECT DISTINCT ON (cp.child_id)
        cp.child_id,
        p.household_id
    FROM fees.child_parents cp
    INNER JOIN fees.parents p ON p.id = cp.parent_id
    WHERE p.household_id IS NOT NULL
    ORDER BY cp.child_id, cp.is_primary DESC
)
UPDATE fees.children c
SET household_id = ch.household_id
FROM child_household ch
WHERE c.id = ch.child_id;

-- Step 4: Handle any children without linked parents - create solo households
INSERT INTO fees.households (name)
SELECT 'Familie ' || c.last_name
FROM fees.children c
WHERE c.household_id IS NULL;

-- Link orphan children to their new households
UPDATE fees.children c
SET household_id = h.id
FROM fees.households h
WHERE c.household_id IS NULL
AND h.name = 'Familie ' || c.last_name;
