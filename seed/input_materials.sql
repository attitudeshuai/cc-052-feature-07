-- Seed data for input materials dictionary
INSERT INTO input_material (name, type, registration_no, safe_interval_days, active_ingredient) VALUES
('草甘膦', 'pesticide', 'PD2018-001', 14, '草甘膦'),
('吡虫啉', 'pesticide', 'PD2019-002', 7, '吡虫啉'),
('阿维菌素', 'pesticide', 'PD2020-003', 3, '阿维菌素'),
('复合肥(15-15-15)', 'fertilizer', NULL, 0, 'N-P2O5-K2O 15-15-15'),
('尿素', 'fertilizer', NULL, 0, '尿素(含氮46%)'),
('有机肥', 'fertilizer', NULL, 0, '有机质≥45%')
ON CONFLICT DO NOTHING;