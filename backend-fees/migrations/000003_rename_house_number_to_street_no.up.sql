-- Rename house_number to street_no in children table
ALTER TABLE children RENAME COLUMN house_number TO street_no;

-- Rename house_number to street_no in parents table
ALTER TABLE parents RENAME COLUMN house_number TO street_no;
