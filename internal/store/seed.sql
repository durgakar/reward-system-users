INSERT INTO clients (id, email, first_name, last_name) VALUES
('c-001', 'alice@example.com', 'Alice', 'Nguyen'),
('c-002', 'bob@example.com', 'Bob', 'Smith'),
('c-003', 'cara@example.com', 'Cara', 'Lee'),
('c-004', 'dan@example.com', 'Dan', 'Patel'),
('c-005', 'eva@example.com', 'Eva', 'Garcia');

INSERT INTO client_profiles (client_id, lifetime_spend_usd, last_order_total_usd, last_order_at, orders_last_90_days, average_order_usd, preferred_category, days_since_last_order) VALUES
('c-001', 820.50, 145.00, '2026-05-20', 4, 95.00, 'electronics', 10),
('c-002', 310.00, 45.00, '2026-05-01', 2, 55.00, 'home', 29),
('c-003', 1200.00, 210.00, '2026-05-18', 5, 120.00, 'electronics', 12),
('c-004', 180.00, 0.00, '2026-02-01', 0, 60.00, 'fashion', 118),
('c-005', 540.00, 88.00, '2026-05-25', 3, 72.00, 'electronics', 5);

INSERT INTO segments (id, description, match_json) VALUES
('high_spender', 'Clients with lifetime spend over $500', '{"all":[{"field":"lifetime_spend_usd","operator":"gte","value":500}]}'),
('frequent_buyer', 'At least 3 orders in the last 90 days', '{"all":[{"field":"orders_last_90_days","operator":"gte","value":3}]}'),
('at_risk', 'No purchase in 60+ days but historically active', '{"all":[{"field":"days_since_last_order","operator":"gte","value":60},{"field":"lifetime_spend_usd","operator":"gte","value":200}]}'),
('electronics_fan', 'Prefers electronics category', '{"all":[{"field":"preferred_category","operator":"eq","value":"electronics"}]}');

INSERT INTO rules (id, name, description, segment, condition_json, actions_json, enabled) VALUES
('high_order_bonus', '500 points for orders $100+', 'Award 500 points when the latest order exceeds $100 USD', 'high_spender',
 '{"field":"last_order_total_usd","operator":"gte","value":100}',
 '[{"type":"award_points","points":500},{"type":"send_email","template":"high_spender_bonus","subject":"You earned {{.Points}} reward points, {{.Client.FirstName}}!"}]', TRUE),
('frequent_buyer_thanks', 'Thank frequent buyers', 'Send a thank-you email to frequent buyers (no points)', 'frequent_buyer',
 '{"field":"orders_last_90_days","operator":"gte","value":3}',
 '[{"type":"send_email","template":"frequent_buyer_thanks","subject":"Thanks for shopping with us, {{.Client.FirstName}}"}]', TRUE),
('win_back_at_risk', 'Win-back bonus for inactive clients', '250 points when at-risk clients return with any order', 'at_risk',
 '{"field":"last_order_total_usd","operator":"gte","value":1}',
 '[{"type":"award_points","points":250},{"type":"send_email","template":"welcome_back","subject":"Welcome back — {{.Points}} points added to your account"}]', TRUE),
('electronics_milestone', 'Electronics category milestone', '100 points when electronics fans spend $75+ on latest order', 'electronics_fan',
 '{"field":"last_order_total_usd","operator":"gte","value":75}',
 '[{"type":"send_email","template":"category_bonus","subject":"Electronics bonus — {{.Points}} points for you"},{"type":"award_points","points":100}]', TRUE);
