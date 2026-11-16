-- Fixture for the MySQL live tests (PUFFSQL_LIVE_MYSQL=1). Load it as a user that may create
-- databases, e.g.:  mysql -uroot -p < internal/mysql/testdata/puff_shop.sql
--
-- The extreme BIGINT UNSIGNED value lives in a plain column on purpose: storing it in an
-- AUTO_INCREMENT key exhausts the counter and every later INSERT fails with error 1467.

DROP DATABASE IF EXISTS puff_shop;
CREATE DATABASE puff_shop CHARACTER SET utf8mb4;
USE puff_shop;

CREATE TABLE customers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '客戶編號',
  name VARCHAR(100) NOT NULL COMMENT '姓名',
  email VARCHAR(200) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  ref_no BIGINT UNSIGNED NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_customers_email (email)
) ENGINE=InnoDB;

CREATE TABLE `order items` (
  order_id BIGINT NOT NULL,
  line_no INT NOT NULL,
  customer_id BIGINT UNSIGNED NOT NULL,
  amount DECIMAL(38,10) NOT NULL DEFAULT 0,
  qty INT NOT NULL DEFAULT 1,
  total DECIMAL(38,10) GENERATED ALWAYS AS (amount * qty) STORED,
  flag BIT(1) NOT NULL DEFAULT b'0',
  ratio FLOAT NULL,
  note TEXT NULL,
  payload BLOB NULL,
  shipped DATE NULL,
  PRIMARY KEY (order_id, line_no),
  KEY ix_items_customer (customer_id, amount DESC),
  CONSTRAINT fk_items_customer FOREIGN KEY (customer_id) REFERENCES customers (id) ON DELETE CASCADE,
  CONSTRAINT ck_items_qty CHECK (qty > 0)
) ENGINE=InnoDB;

CREATE VIEW v_customer_totals AS
  SELECT c.id, c.name, SUM(i.total) AS total
  FROM customers c LEFT JOIN `order items` i ON i.customer_id = c.id
  GROUP BY c.id, c.name;

CREATE TRIGGER trg_items_bi BEFORE INSERT ON `order items`
  FOR EACH ROW SET NEW.note = COALESCE(NEW.note, 'auto');

INSERT INTO customers (id, name, email, ref_no) VALUES
  (1, 'O''Brien', NULL, NULL),
  (2, 'Max', 'max@example.test', 18446744073709551615);

INSERT INTO `order items` (order_id, line_no, customer_id, amount, qty, flag, ratio, payload, shipped) VALUES
  (9007199254740993, 1, 1, 1234567890123456789012345678.0123456789, 2, b'1', 1.1, x'010203', '2026-01-02'),
  (2, 1, 2, 12.50, 1, b'0', NULL, NULL, NULL);
