-- Esquema SQLite para Proyecto_CobrApp
-- Basado en clientes_deudas.drawio

PRAGMA foreign_keys = ON;

-- Cliente (en el diagrama: cuentaCliente)
CREATE TABLE IF NOT EXISTS cliente (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre      TEXT NOT NULL,
    apellido    TEXT NOT NULL,
    email       TEXT UNIQUE,
    telefono    TEXT,
    creado_en   TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Compra: cada cliente puede tener muchas compras (1..N)
CREATE TABLE IF NOT EXISTS compra (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id   INTEGER NOT NULL,
    total        REAL NOT NULL CHECK (total >= 0),
    descripcion  TEXT,
    fecha        TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (cliente_id) REFERENCES cliente(id) ON DELETE CASCADE
);

-- Pago: cada cliente puede tener muchos pagos (1..N)
-- OJO: en el diagrama el pago se asocia a la CUENTA del cliente,
-- no a una compra específica (modelo de cuenta corriente).
CREATE TABLE IF NOT EXISTS pago (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id   INTEGER NOT NULL,
    monto        REAL NOT NULL CHECK (monto > 0),
    observacion  TEXT,
    fecha        TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (cliente_id) REFERENCES cliente(id) ON DELETE CASCADE
);

-- Índices para consultas frecuentes (historial por cliente)
CREATE INDEX IF NOT EXISTS idx_compra_cliente ON compra(cliente_id);
CREATE INDEX IF NOT EXISTS idx_pago_cliente ON pago(cliente_id);

-- Vista de saldo/deuda por cliente: total comprado - total pagado
CREATE VIEW IF NOT EXISTS vista_saldo_cliente AS
SELECT
    c.id AS cliente_id,
    c.nombre,
    c.apellido,
    c.email,
    c.telefono,
    COALESCE((SELECT SUM(total) FROM compra WHERE compra.cliente_id = c.id), 0) AS total_comprado,
    COALESCE((SELECT SUM(monto) FROM pago WHERE pago.cliente_id = c.id), 0) AS total_pagado,
    COALESCE((SELECT SUM(total) FROM compra WHERE compra.cliente_id = c.id), 0)
      - COALESCE((SELECT SUM(monto) FROM pago WHERE pago.cliente_id = c.id), 0) AS saldo_pendiente
FROM cliente c;