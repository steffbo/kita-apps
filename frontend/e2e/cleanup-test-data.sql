-- ============================================================================
-- Playwright E2E Test Data Cleanup Script
-- ============================================================================
-- This script removes test data created by Playwright E2E tests.
-- Run this against the fees schema to clean up after test runs.
--
-- Test data is identified by naming patterns used in the tests:
-- - Children: member_number starting with T, S, D, P, H, L (single letter + 6 digits)
-- - Children: first_name patterns like 'Test', 'Detail-', 'Parent-Test-', etc.
-- - Parents: first_name patterns like 'Eltern-', 'Card-', 'Multi1-', etc.
-- ============================================================================

BEGIN;

-- Delete payment matches for test fee expectations
DELETE FROM fees.payment_matches
WHERE expectation_id IN (
    SELECT fe.id FROM fees.fee_expectations fe
    JOIN fees.children c ON fe.child_id = c.id
    WHERE c.member_number ~ '^[TSDPHL][0-9]{6}$'
);

-- Delete fee expectations for test children
DELETE FROM fees.fee_expectations
WHERE child_id IN (
    SELECT id FROM fees.children
    WHERE member_number ~ '^[TSDPHL][0-9]{6}$'
);

-- Delete child-parent relationships for test children
DELETE FROM fees.child_parents
WHERE child_id IN (
    SELECT id FROM fees.children
    WHERE member_number ~ '^[TSDPHL][0-9]{6}$'
);

-- Delete child-parent relationships for test parents
DELETE FROM fees.child_parents
WHERE parent_id IN (
    SELECT id FROM fees.parents
    WHERE first_name ~ '^(Eltern-|Card-|Multi[12]-|Suchbar-|Link-|Unlink-|CancelUnlink-)[0-9]{4}$'
       OR first_name IN ('Test', 'Helper', 'LinkHelper')
       OR last_name IN ('Elternteil', 'Testperson', 'Display', 'Erster', 'Zweiter', 'Zuverknüpfen', 'Zuentfernen', 'Bleiben')
);

-- Delete test children
-- Pattern: member_number is a single letter (T, S, D, P, H, L) followed by exactly 6 digits
DELETE FROM fees.children
WHERE member_number ~ '^[TSDPHL][0-9]{6}$';

-- Delete test children by name pattern (backup in case member_number pattern changes)
DELETE FROM fees.children
WHERE first_name ~ '^(Test|Detail-|Parent-Test-|Suchtest-|SearchHelper-|LinkHelper-)[0-9]{0,4}$'
   OR first_name IN ('Helper', 'LinkHelper')
   OR last_name IN ('Testname', 'Nachname', 'Child');

-- Delete test parents
-- Pattern: first_name contains test identifiers
DELETE FROM fees.parents
WHERE first_name ~ '^(Eltern-|Card-|Multi[12]-|Suchbar-|Link-|Unlink-|CancelUnlink-)[0-9]{4}$'
   OR first_name IN ('Test', 'Helper', 'LinkHelper')
   OR last_name IN ('Elternteil', 'Testperson', 'Display', 'Erster', 'Zweiter', 'Zuverknüpfen', 'Zuentfernen', 'Bleiben');

-- Delete orphaned parents (parents with no children linked)
-- This catches any test parents that might have been missed
DELETE FROM fees.parents
WHERE id NOT IN (SELECT DISTINCT parent_id FROM fees.child_parents)
  AND created_at > NOW() - INTERVAL '7 days'
  AND (
    first_name ~ '-[0-9]{4}$'
    OR last_name IN ('Elternteil', 'Testperson', 'Display', 'Erster', 'Zweiter', 'Zuverknüpfen', 'Zuentfernen', 'Bleiben')
  );

COMMIT;

-- ============================================================================
-- Verification queries (run these to check what would be deleted)
-- ============================================================================

-- Preview test children that would be deleted:
-- SELECT id, member_number, first_name, last_name, created_at
-- FROM fees.children
-- WHERE member_number ~ '^[TSDPHL][0-9]{6}$'
-- ORDER BY created_at DESC;

-- Preview test parents that would be deleted:
-- SELECT id, first_name, last_name, email, created_at
-- FROM fees.parents
-- WHERE first_name ~ '^(Eltern-|Card-|Multi[12]-|Suchbar-|Link-|Unlink-|CancelUnlink-)[0-9]{4}$'
--    OR last_name IN ('Elternteil', 'Testperson', 'Display', 'Erster', 'Zweiter', 'Zuverknüpfen', 'Zuentfernen', 'Bleiben')
-- ORDER BY created_at DESC;

-- Count test data:
-- SELECT 'children' as table_name, COUNT(*) as count FROM fees.children WHERE member_number ~ '^[TSDPHL][0-9]{6}$'
-- UNION ALL
-- SELECT 'parents', COUNT(*) FROM fees.parents WHERE first_name ~ '-[0-9]{4}$' OR last_name IN ('Elternteil', 'Testperson', 'Display');
