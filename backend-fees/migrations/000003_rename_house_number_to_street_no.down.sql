-- Revert: Rename street_no back to house_number in children table
ALTER TABLE fees.children RENAME COLUMN street_no TO house_number;

-- Revert: Rename street_no back to house_number in parents table
ALTER TABLE fees.parents RENAME COLUMN street_no TO house_number;
