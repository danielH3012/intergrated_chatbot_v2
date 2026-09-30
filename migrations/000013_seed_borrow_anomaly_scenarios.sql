-- =============================================================================
-- 000013: Seed Borrow Anomaly Scenarios & Asset Conditions
-- =============================================================================

-- 1. Set Damaged and Missing conditions for testing AI query ("aset rusak ada berapa")
UPDATE public.assets SET condition = 'Damaged' WHERE asset_id IN ('AST-57BDAD55', 'AST-362EAE60');
UPDATE public.assets SET condition = 'Missing' WHERE asset_id IN ('AST-587CEEAD');

-- 2. Ensure Audit Schedule exists for Group 'qtera mandiri'
DELETE FROM public.schedule WHERE category = 'Audit Internal Semester 2';
INSERT INTO public.schedule (id_schedule, category, start, frequency, group_name)
VALUES ('b817fa91-4473-455b-bf98-0c679a941a31', 'Audit Internal Semester 2', '2026-10-04', 'Semi-Annual', 'qtera mandiri');

-- 3. Clean previous test borrow data to ensure idempotency
DELETE FROM public.borrow_extensions WHERE borrower_id IN (
    '9896b88d-1565-47ba-83f1-205b62ca12af', 'e9de1308-99a2-4c35-9d7e-304b68c0c36b', 
    '08b78e1d-9838-4706-9409-09ba9c13aa26', '0fa40396-7f3d-4428-8764-36bcb2431125', 
    'd2e60407-f362-42fb-8507-8076159c24c8'
);
DELETE FROM public.borrow_transactions WHERE transaction_code LIKE 'TRX-BRW-%' OR transaction_code LIKE 'HIST-%';

-- =============================================================================
-- Historical Baseline Data for Anomaly Calculations
-- =============================================================================

-- Hist 1: User 'tordalk' has 2 late returns in past 6 months
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status, created_at)
VALUES 
('c1000000-0000-0000-0000-000000000001', 'HIST-LATE-01', 'e9de1308-99a2-4c35-9d7e-304b68c0c36b', 'tordalk', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW() - INTERVAL '40 days', 'completed', NOW() - INTERVAL '40 days'),
('c1000000-0000-0000-0000-000000000002', 'HIST-LATE-02', 'e9de1308-99a2-4c35-9d7e-304b68c0c36b', 'tordalk', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW() - INTERVAL '20 days', 'completed', NOW() - INTERVAL '20 days');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, approved_duration_days, is_approved, status, borrowed_at, due_date, returned_at)
VALUES
('d1000000-0000-0000-0000-000000000001', 'c1000000-0000-0000-0000-000000000001', 'AST-5DFBE7CF', 7, 7, true, 'returned', NOW() - INTERVAL '40 days', NOW() - INTERVAL '33 days', NOW() - INTERVAL '30 days'), -- late by 3 days
('d1000000-0000-0000-0000-000000000002', 'c1000000-0000-0000-0000-000000000002', 'AST-44337097', 7, 7, true, 'returned', NOW() - INTERVAL '20 days', NOW() - INTERVAL '13 days', NOW() - INTERVAL '9 days');  -- late by 4 days

-- Hist 2: User 'andhika' has 3 currently active loans
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status, created_at)
VALUES 
('c2000000-0000-0000-0000-000000000001', 'HIST-ACT-01', '08b78e1d-9838-4706-9409-09ba9c13aa26', 'andhika', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW() - INTERVAL '10 days', 'active', NOW() - INTERVAL '10 days'),
('c2000000-0000-0000-0000-000000000002', 'HIST-ACT-02', '08b78e1d-9838-4706-9409-09ba9c13aa26', 'andhika', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW() - INTERVAL '5 days', 'active', NOW() - INTERVAL '5 days'),
('c2000000-0000-0000-0000-000000000003', 'HIST-ACT-03', '08b78e1d-9838-4706-9409-09ba9c13aa26', 'andhika', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW() - INTERVAL '3 days', 'active', NOW() - INTERVAL '3 days');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, approved_duration_days, is_approved, status, borrowed_at, due_date)
VALUES
('d2000000-0000-0000-0000-000000000001', 'c2000000-0000-0000-0000-000000000001', 'AST-1C66C313', 14, 14, true, 'borrowed', NOW() - INTERVAL '10 days', NOW() + INTERVAL '4 days'),
('d2000000-0000-0000-0000-000000000002', 'c2000000-0000-0000-0000-000000000002', 'AST-FD95E810', 14, 14, true, 'borrowed', NOW() - INTERVAL '5 days', NOW() + INTERVAL '9 days'),
('d2000000-0000-0000-0000-000000000003', 'c2000000-0000-0000-0000-000000000003', 'AST-D87DBDB0', 14, 14, true, 'borrowed', NOW() - INTERVAL '3 days', NOW() + INTERVAL '11 days');

-- Hist 3: User 'alvin' rapid consecutive borrow on same asset AST-E4BA751A
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status, created_at)
VALUES 
('c3000000-0000-0000-0000-000000000001', 'HIST-CYCLE-01', 'd2e60407-f362-42fb-8507-8076159c24c8', 'alvin', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'adara group', NOW() - INTERVAL '30 days', 'completed', NOW() - INTERVAL '30 days'),
('c3000000-0000-0000-0000-000000000002', 'HIST-CYCLE-02', 'd2e60407-f362-42fb-8507-8076159c24c8', 'alvin', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'adara group', NOW() - INTERVAL '15 days', 'completed', NOW() - INTERVAL '15 days');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, approved_duration_days, is_approved, status, borrowed_at, due_date, returned_at)
VALUES
('d3000000-0000-0000-0000-000000000001', 'c3000000-0000-0000-0000-000000000001', 'AST-E4BA751A', 7, 7, true, 'returned', NOW() - INTERVAL '30 days', NOW() - INTERVAL '23 days', NOW() - INTERVAL '23 days'),
('d3000000-0000-0000-0000-000000000002', 'c3000000-0000-0000-0000-000000000002', 'AST-E4BA751A', 7, 7, true, 'returned', NOW() - INTERVAL '22 days', NOW() - INTERVAL '15 days', NOW() - INTERVAL '15 days'); -- reborrowed next day (cycle 1)

-- =============================================================================
-- Active Pending Approval Transactions for Testing (The 8 Scenarios)
-- =============================================================================

-- 1. TRX-BRW-001: Clean / Normal (0 Anomaly Signals)
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status)
VALUES ('a1000000-0000-0000-0000-000000000001', 'TRX-BRW-001', '9896b88d-1565-47ba-83f1-205b62ca12af', 'sasori', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW(), 'pending_approval');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, status)
VALUES ('b1000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001', 'AST-5DFBE7CF', 5, 'pending');

-- 2. TRX-BRW-002: High Duration (1 Anomaly Signal -> Review Recommended without joint narrative)
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status)
VALUES ('a1000000-0000-0000-0000-000000000002', 'TRX-BRW-002', '9896b88d-1565-47ba-83f1-205b62ca12af', 'sasori', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW(), 'pending_approval');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, status)
VALUES ('b1000000-0000-0000-0000-000000000002', 'a1000000-0000-0000-0000-000000000002', 'AST-44337097', 45, 'pending'); -- Dell Latitude Laptop requested for 45 days (> 2x default 14 days)

-- 3. TRX-BRW-003: Late Return User (Signal 3: >= 2 late returns in 6 months)
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status)
VALUES ('a1000000-0000-0000-0000-000000000003', 'TRX-BRW-003', 'e9de1308-99a2-4c35-9d7e-304b68c0c36b', 'tordalk', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW(), 'pending_approval');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, status)
VALUES ('b1000000-0000-0000-0000-000000000003', 'a1000000-0000-0000-0000-000000000003', 'AST-18A527A6', 7, 'pending');

-- 4. TRX-BRW-004: User with >= 3 Active Loans (Signal 2)
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status)
VALUES ('a1000000-0000-0000-0000-000000000004', 'TRX-BRW-004', '08b78e1d-9838-4706-9409-09ba9c13aa26', 'andhika', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW(), 'pending_approval');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, status)
VALUES ('b1000000-0000-0000-0000-000000000004', 'a1000000-0000-0000-0000-000000000004', 'AST-1CEE1138', 7, 'pending');

-- 5. TRX-BRW-005: Bulk Assets (> 5 Assets in single request - Signal 5)
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status)
VALUES ('a1000000-0000-0000-0000-000000000005', 'TRX-BRW-005', '0fa40396-7f3d-4428-8764-36bcb2431125', 'osas', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'bunda mulia', NOW(), 'pending_approval');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, status)
VALUES 
('b1000000-0000-0000-0000-000000000005', 'a1000000-0000-0000-0000-000000000005', 'AST-1C66C313', 5, 'pending'),
('b1000000-0000-0000-0000-000000000006', 'a1000000-0000-0000-0000-000000000005', 'AST-FD95E810', 5, 'pending'),
('b1000000-0000-0000-0000-000000000007', 'a1000000-0000-0000-0000-000000000005', 'AST-D87DBDB0', 5, 'pending'),
('b1000000-0000-0000-0000-000000000008', 'a1000000-0000-0000-0000-000000000005', 'AST-EABA6E94', 5, 'pending'),
('b1000000-0000-0000-0000-000000000009', 'a1000000-0000-0000-0000-000000000005', 'AST-18A527A6', 5, 'pending'),
('b1000000-0000-0000-0000-000000000010', 'a1000000-0000-0000-0000-000000000005', 'AST-1CEE1138', 5, 'pending');

-- 6. TRX-BRW-006: Audit Avoidance (Signal 6: within 7 days of audit schedule)
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status)
VALUES ('a1000000-0000-0000-0000-000000000006', 'TRX-BRW-006', '9896b88d-1565-47ba-83f1-205b62ca12af', 'sasori', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW(), 'pending_approval');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, status)
VALUES ('b1000000-0000-0000-0000-000000000011', 'a1000000-0000-0000-0000-000000000006', 'AST-548993A5', 7, 'pending');

-- 7. TRX-BRW-007: Consecutive Rapid Borrow (Signal 7: re-borrowing AST-E4BA751A within <= 1 day of prior return)
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status)
VALUES ('a1000000-0000-0000-0000-000000000007', 'TRX-BRW-007', 'd2e60407-f362-42fb-8507-8076159c24c8', 'alvin', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'adara group', NOW(), 'pending_approval');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, status)
VALUES ('b1000000-0000-0000-0000-000000000012', 'a1000000-0000-0000-0000-000000000007', 'AST-E4BA751A', 7, 'pending');

-- 8. TRX-BRW-008: Multi-Anomaly (>= 2 Signals: High Duration + Frequent Late Return -> Triggers AI joint narrative)
INSERT INTO public.borrow_transactions (id, transaction_code, borrower_id, borrower_name, manager_id, manager_name, group_name, request_date, status)
VALUES ('a1000000-0000-0000-0000-000000000008', 'TRX-BRW-008', 'e9de1308-99a2-4c35-9d7e-304b68c0c36b', 'tordalk', '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'qtera mandiri', NOW(), 'pending_approval');

INSERT INTO public.borrow_items (id, transaction_id, asset_id, duration_days, status)
VALUES ('b1000000-0000-0000-0000-000000000013', 'a1000000-0000-0000-0000-000000000008', 'AST-44337097', 60, 'pending'); -- Laptop 60 days + tordalk has 2 prior late returns
