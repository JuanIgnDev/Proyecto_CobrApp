-- Migración de una sola vez para bases de datos que ya tenían datos
-- cargados con los nombres viejos ("compra" y "pago").
--
-- Corré esto UNA SOLA VEZ, ANTES de levantar la app con el código nuevo:
--   sqlite3 data/cobrapp.db < data/migracion_venta_cobro.sql
--
-- Si tu base todavía no tiene datos (o es la primera vez que corrés el
-- proyecto), no hace falta este paso: el schema.sql nuevo ya crea las
-- tablas "venta" y "cobro" directamente.

ALTER TABLE compra RENAME TO venta;
ALTER TABLE pago RENAME TO cobro;

-- La vista vieja quedó apuntando a nombres de tabla que ya no existen,
-- así que hay que recrearla (schema.sql la vuelve a crear con el nombre
-- correcto la próxima vez que arranque la app, pero la borramos ahora
-- para evitar que quede una versión rota mientras tanto).
DROP VIEW IF EXISTS vista_saldo_cliente;

DROP INDEX IF EXISTS idx_compra_cliente;
DROP INDEX IF EXISTS idx_pago_cliente;
