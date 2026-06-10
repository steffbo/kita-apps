CREATE OR REPLACE FUNCTION fees.validate_child_parent_household_link()
RETURNS TRIGGER AS $$
DECLARE
    child_household_id UUID;
    parent_household_id UUID;
BEGIN
    SELECT household_id INTO child_household_id
    FROM fees.children
    WHERE id = NEW.child_id;

    SELECT household_id INTO parent_household_id
    FROM fees.parents
    WHERE id = NEW.parent_id;

    IF child_household_id IS NOT NULL
       AND parent_household_id IS NOT NULL
       AND child_household_id <> parent_household_id THEN
        RAISE EXCEPTION 'child_parent household mismatch: child % household %, parent % household %',
            NEW.child_id, child_household_id, NEW.parent_id, parent_household_id
            USING ERRCODE = '23514';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fees.validate_child_household_against_parents()
RETURNS TRIGGER AS $$
DECLARE
    mismatched_parent RECORD;
BEGIN
    IF NEW.household_id IS NOT NULL THEN
        SELECT p.id, p.household_id
        INTO mismatched_parent
        FROM fees.child_parents cp
        JOIN fees.parents p ON p.id = cp.parent_id
        WHERE cp.child_id = NEW.id
          AND p.household_id IS NOT NULL
          AND p.household_id <> NEW.household_id
        LIMIT 1;

        IF FOUND THEN
            RAISE EXCEPTION 'child household mismatch: child % household %, linked parent % household %',
                NEW.id, NEW.household_id, mismatched_parent.id, mismatched_parent.household_id
                USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fees.validate_parent_household_against_children()
RETURNS TRIGGER AS $$
DECLARE
    mismatched_child RECORD;
BEGIN
    IF NEW.household_id IS NOT NULL THEN
        SELECT c.id, c.household_id
        INTO mismatched_child
        FROM fees.child_parents cp
        JOIN fees.children c ON c.id = cp.child_id
        WHERE cp.parent_id = NEW.id
          AND c.household_id IS NOT NULL
          AND c.household_id <> NEW.household_id
        LIMIT 1;

        IF FOUND THEN
            RAISE EXCEPTION 'parent household mismatch: parent % household %, linked child % household %',
                NEW.id, NEW.household_id, mismatched_child.id, mismatched_child.household_id
                USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS validate_child_parent_household_link ON fees.child_parents;
CREATE CONSTRAINT TRIGGER validate_child_parent_household_link
AFTER INSERT OR UPDATE ON fees.child_parents
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW EXECUTE FUNCTION fees.validate_child_parent_household_link();

DROP TRIGGER IF EXISTS validate_child_household_against_parents ON fees.children;
CREATE CONSTRAINT TRIGGER validate_child_household_against_parents
AFTER INSERT OR UPDATE OF household_id ON fees.children
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW EXECUTE FUNCTION fees.validate_child_household_against_parents();

DROP TRIGGER IF EXISTS validate_parent_household_against_children ON fees.parents;
CREATE CONSTRAINT TRIGGER validate_parent_household_against_children
AFTER INSERT OR UPDATE OF household_id ON fees.parents
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW EXECUTE FUNCTION fees.validate_parent_household_against_children();
