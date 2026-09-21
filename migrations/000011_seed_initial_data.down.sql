-- 000011_seed_initial_data.down.sql
-- Remove seeded data

DELETE FROM public.assets WHERE asset_id IN (
    'AST-5DFBE7CF', 'AST-44337097', 'AST-E4BA751A', 'AST-587CEEAD', 'AST-548993A5', 'AST-57BDAD55', 'AST-362EAE60'
);

DELETE FROM public.schedule WHERE id_schedule = '922ad420-108a-40aa-8631-d8ef37026731';

DELETE FROM public.users WHERE id IN (
    '1c4f40db-e21a-4dcc-8fcf-9d177fa94bb5',
    '0fa40396-7f3d-4428-8764-36bcb2431125',
    'e9de1308-99a2-4c35-9d7e-304b68c0c36b',
    '9896b88d-1565-47ba-83f1-205b62ca12af',
    '9929339a-102e-42a4-97ed-c014ad6e3f88',
    'a675fe31-b828-474c-9d9a-9e216a66f3a4',
    '08b78e1d-9838-4706-9409-09ba9c13aa26',
    'd2e60407-f362-42fb-8507-8076159c24c8'
);

DELETE FROM public.perusahaan WHERE id_perusahaan IN (1, 2, 3, 4, 5);
