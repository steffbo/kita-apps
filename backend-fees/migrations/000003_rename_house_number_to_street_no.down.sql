-- Revert: Rename street_no back to house_number in children table
ALTER TABLE children RENAME COLUMN street_no TO house_number;

-- Revert: Rename street_no back to house_number in parents table
ALTER TABLE parents RENAME COLUMN street_no TO house_number;
