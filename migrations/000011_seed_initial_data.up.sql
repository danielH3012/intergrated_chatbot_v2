-- 000011_seed_initial_data.up.sql
-- Seed initial companies, users, schedules, and representative assets from db.sql

-- 1. Companies (Perusahaan)
INSERT INTO public.perusahaan (id_perusahaan, nama_perusahaan)
VALUES
    (1, 'qtera mandiri'),
    (2, 'bunda mulia'),
    (3, 'arista'),
    (4, 'adara group'),
    (5, 'dynamic learning center')
ON CONFLICT (id_perusahaan) DO NOTHING;

-- Synchronize sequence
SELECT setval(pg_get_serial_sequence('public.perusahaan', 'id_perusahaan'), COALESCE(MAX(id_perusahaan), 1), true) FROM public.perusahaan;

-- 2. Users
INSERT INTO public.users (id, username, email, password_hash, role, company, id_perusahaan, created_at, updated_at)
VALUES
    ('1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5', 'admin', 'admin@gmail.com', '$2a$10$vGGOdXoeeWr9cRn.ghIlouFfZiM2rNb11umFDDaTCft5GESi84Gd6', 'admin', 'qtera mandiri', 1, '2026-09-01 11:01:30.807565+07', '2026-09-01 11:01:30.807565+07'),
    ('0fa40396-7f3d-4428-8764-36bcb2431125', 'osas', 'osas@gmail.com', '$2a$10$3NOS3OylaK8pTJIB8kp52eUxh5q4Jxw0DXnTgkL9OcXki02KyjNgK', 'admin', 'bunda mulia', 2, '2026-09-01 10:55:09.911819+07', '2026-09-01 10:55:09.911819+07'),
    ('e9de1308-99a2-4c35-9d7e-304b68c0c36b', 'tordalk', 'danherdjanto@gmail.com', '$2a$10$vZo6UqduOPa/A4JpENiymOKQjJqNXrwmTmkb.QS447njqtWt.kr4u', 'operator', 'qtera mandiri', 1, '2026-08-31 17:18:34.924964+07', '2026-08-31 17:18:34.924964+07'),
    ('9896b88d-1565-47ba-83f1-205b62ca12af', 'sasori', 'danielherdjanto@gmail.com', '$2a$10$Drg2a5pkiAe/fZLlyS6GpuwRtx2P07ft9jpuG2zLygiZgZRMZRATy', 'operator', 'qtera mandiri', 1, '2026-09-01 09:23:50.850115+07', '2026-09-01 09:23:50.850115+07'),
    ('9929339a-102e-42a4-97ed-c014ad6e3f88', 'admin2', 'apasaja@gmail.com', '$2a$10$ukNJCXYotDtchHcqM7pHueFc/A.HnGIDiDjWklNwjliKrDqQQ7PiO', 'admin', 'qtera mandiri', 1, '2026-09-02 16:54:50.868378+07', '2026-09-02 16:54:50.868378+07'),
    ('a675fe31-b828-474c-9d9a-9e216a66f3a4', 'jason', 's32230127@student.ubm.ac.id', '$2a$10$E6bh8Jm.HiWJs1sY15G2YuHueA03LrJFioBn75T0bV0y7bc/gX31K', 'viewer', 'bunda mulia', 2, '2026-09-03 14:12:21.661636+07', '2026-09-03 14:12:21.661636+07'),
    ('08b78e1d-9838-4706-9409-09ba9c13aa26', 'andhika', 's32220168@student.ubm.ac.id', '$2a$10$cfktlyEupj1ZwFQJgjTVW.l3BzRYjGYGLtvWadSAgUkYDZiTmi1aO', 'operator', 'adara group', 4, '2026-09-03 14:19:28.514359+07', '2026-09-03 14:19:28.514359+07'),
    ('d2e60407-f362-42fb-8507-8076159c24c8', 'alvin', 'alvin@gmail.con', '$2a$10$xTsUhAUxg821LdCNl.DwyeTE1LkiHhM8LN5oApIAJRFud4FIw2KJe', 'viewer', 'adara group', 4, '2026-09-03 17:13:35.686657+07', '2026-09-03 17:13:35.686657+07')
ON CONFLICT (id) DO NOTHING;

-- 3. Schedule
INSERT INTO public.schedule (id_schedule, category, start, frequency)
VALUES
    ('922ad420-108a-40aa-8631-d8ef37026731', 'Pemeriksaan Aset Rutin', '2026-09-17', 'Weekly (setiap hari Kamis)')
ON CONFLICT (id_schedule) DO NOTHING;

-- 4. Sample Assets
INSERT INTO public.assets (asset_id, name, category, brand, model_type, purchase_date, purchase_price, location, created_at, perusahaan, status)
VALUES
    ('AST-5DFBE7CF', 'Logitech MX Master 3S', 'IT Equipment > Peripheral', 'Logitech', 'MX Master 3S', '2026-08-01', '1450000', 'Warehouse A', '2026-09-01T15:08:50+07:00', 'qtera mandiri', 'active'),
    ('AST-44337097', 'Dell Latitude 5450', 'IT Equipment > Laptop', 'Dell', 'Latitude 5450', '2026-08-20', '18500000', 'Design Studio', '2026-09-01T15:08:50+07:00', 'qtera mandiri', 'active'),
    ('AST-E4BA751A', 'LG UltraWide 34WN80C', 'IT Equipment > Monitor', 'LG', '34WN80C', '2026-08-15', '6200000', 'Office 2nd Floor', '2026-09-01T15:08:50+07:00', 'qtera mandiri', 'active'),
    ('AST-587CEEAD', 'Toyota HiAce Commuter', 'Vehicle', 'Toyota', 'HiAce Commuter', '2026-05-10', '550000000', 'Warehouse B', '2026-09-01T15:08:50+07:00', 'qtera mandiri', 'active'),
    ('AST-548993A5', 'Herman Miller Aeron Chair', 'Furniture', 'Herman Miller', 'Aeron Chair', '2026-07-02', '15750000', 'Office 3rd Floor', '2026-09-02T16:33:03+07:00', 'qtera mandiri', 'active'),
    ('AST-57BDAD55', 'Site Warranty', 'IT Equipment', 'Generic', 'Warranty', '2026-09-01', '688', 'Warehouse B', '2026-09-02T16:41:56+07:00', 'qtera mandiri', 'active'),
    ('AST-362EAE60', 'Thermostat', 'IT Equipment', 'Generic', 'Thermostat', '2026-09-01', '150000', 'Warehouse B', '2026-09-02T16:41:56+07:00', 'qtera mandiri', 'active')
ON CONFLICT (asset_id) DO NOTHING;
