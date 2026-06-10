DROP TRIGGER IF EXISTS validate_parent_household_against_children ON fees.parents;
DROP TRIGGER IF EXISTS validate_child_household_against_parents ON fees.children;
DROP TRIGGER IF EXISTS validate_child_parent_household_link ON fees.child_parents;

DROP FUNCTION IF EXISTS fees.validate_parent_household_against_children();
DROP FUNCTION IF EXISTS fees.validate_child_household_against_parents();
DROP FUNCTION IF EXISTS fees.validate_child_parent_household_link();
