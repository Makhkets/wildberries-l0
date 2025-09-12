-- Удаление триггера
DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;

-- Удаление функции
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Удаление индексов
DROP INDEX IF EXISTS idx_items_track_number;
DROP INDEX IF EXISTS idx_items_nm_id;
DROP INDEX IF EXISTS idx_items_chrt_id;
DROP INDEX IF EXISTS idx_items_order_id;

DROP INDEX IF EXISTS idx_payment_transaction;
DROP INDEX IF EXISTS idx_payment_order_id;
DROP INDEX IF EXISTS idx_delivery_order_id;

DROP INDEX IF EXISTS idx_orders_date_created;
DROP INDEX IF EXISTS idx_orders_customer_id;
DROP INDEX IF EXISTS idx_orders_track_number;
DROP INDEX IF EXISTS idx_orders_order_uid;

-- Удаление таблиц (в обратном порядке из-за внешних ключей)
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS payment;
DROP TABLE IF EXISTS delivery;
DROP TABLE IF EXISTS orders;
